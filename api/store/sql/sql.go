package sql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
)

// QueryStructer is an abstract type with a QueryStruct method.
type QueryStructer interface {
	QueryStruct(context.Context, interface{}, string, ...interface{}) error
}

// QueryStructSlicer is an abstract type with a QueryStructsLice method.
type QueryStructSlicer interface {
	QueryStructSlice(context.Context, interface{}, string, ...interface{}) error
}

// Execer is an abstract type with a ExecContext method.
type Execer interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
}

// Querier is the querier interface.
type Querier interface {
	Execer
	QueryStructer
	QueryStructSlicer

	Begin(ctx context.Context) (Querier, error)
	Commit() error
	Rollback() error
}

// NewDB returns a new db.
func NewDB(db *sql.DB) *DB {
	return &DB{db: db}
}

// DB wraps a sql.DB.
type DB struct {
	db *sql.DB
}

// PingContext pings.
func (db *DB) PingContext(ctx context.Context) error {
	return db.db.PingContext(ctx)
}

// ExecContext is the query context method.
func (db *DB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return db.db.ExecContext(ctx, query, args...)
}

// QueryContext is the query context method.
func (db *DB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return db.db.QueryContext(ctx, query, args...)
}

// QueryStructSlice executes a query and scans the returned rows to dest, which must be a pointer to a slice of structs.
func (db *DB) QueryStructSlice(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return queryStructSlice(ctx, db, dest, query, args...)
}

// QueryStruct executes a query and scans the returned row to dest, which must be a pointer to a struct.
func (db *DB) QueryStruct(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return queryStruct(ctx, db, dest, query, args...)
}

// Begin begins an sql transaction.
func (db *DB) Begin(ctx context.Context) (Querier, error) {
	tx, err := db.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &Tx{tx: tx}, nil
}

// Commit is the commit method.
func (db *DB) Commit() error {
	return errors.New("nothing to commit")
}

// Rollback is the rollback method.
func (db *DB) Rollback() error {
	return errors.New("nothing to rollback")
}

// Tx wraps a sql.Tx.
type Tx struct {
	tx *sql.Tx
}

// ExecContext is the query context method.
func (tx *Tx) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return tx.tx.ExecContext(ctx, query, args...)
}

// QueryContext is the query context method.
func (tx *Tx) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return tx.tx.QueryContext(ctx, query, args...)
}

// QueryStructSlice executes a query and scans the returned rows to dest, which must be a pointer to a slice of structs.
func (tx *Tx) QueryStructSlice(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return queryStructSlice(ctx, tx, dest, query, args...)
}

// QueryStruct executes a query and scans the returned row to dest, which must be a pointer to a struct.
func (tx *Tx) QueryStruct(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return queryStruct(ctx, tx, dest, query, args...)
}

// Begin begins an sql transaction.
func (tx *Tx) Begin(ctx context.Context) (Querier, error) {
	return nil, errors.New("TODO: nest transaction")
}

// Commit is the commit method.
func (tx *Tx) Commit() error {
	return tx.tx.Commit()
}

// Rollback is the rollback method.
func (tx *Tx) Rollback() error {
	return tx.tx.Rollback()
}

type querier interface {
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
}

func queryStructSlice(ctx context.Context, db querier, dest interface{}, query string, args ...interface{}) error {
	slicevalue := reflect.ValueOf(dest).Elem()
	structtype := slicevalue.Type().Elem()

	var structfields []reflect.StructField
	for i := 0; i < structtype.NumField(); i++ {
		structfields = append(structfields, structtype.Field(i))
	}

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	var fieldindexes [][]int
	for _, col := range columns {
		var ok bool
		for _, field := range structfields {
			if col == field.Tag.Get("sql") {
				fieldindexes = append(fieldindexes, field.Index)
				ok = true
				break
			}
		}
		if !ok {
			return fmt.Errorf("couldn't map column %s", col)
		}
	}

	for rows.Next() {
		scannedvalue := reflect.New(structtype).Elem()
		var dests []interface{}
		for _, index := range fieldindexes {
			dests = append(dests, scannedvalue.FieldByIndex(index).Addr().Interface())
		}
		if err := rows.Scan(dests...); err != nil {
			return err
		}
		slicevalue.Set(reflect.Append(slicevalue, scannedvalue))
	}
	return rows.Err()
}

func queryStruct(ctx context.Context, db querier, dest interface{}, query string, args ...interface{}) error {
	structvalue := reflect.ValueOf(dest).Elem()
	structtype := structvalue.Type()

	var fields []reflect.StructField
	for i := 0; i < structtype.NumField(); i++ {
		fields = append(fields, structtype.Field(i))
	}

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return err
		}
		return sql.ErrNoRows
	}

	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	var dests []interface{}
	for _, col := range columns {
		var ok bool
		for _, field := range fields {
			if col == field.Tag.Get("sql") {
				dests = append(dests, structvalue.FieldByIndex(field.Index).Addr().Interface())
				ok = true
				break
			}
		}
		if !ok {
			return fmt.Errorf("couldn't map column %s", col)
		}
	}

	if err := rows.Scan(dests...); err != nil {
		return err
	}
	return rows.Close()
}
