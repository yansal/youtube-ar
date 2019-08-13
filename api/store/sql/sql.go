package sql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"os"
	"reflect"
	"time"

	"github.com/lib/pq"
	"github.com/yansal/sqldriver"
	"github.com/yansal/youtube-ar/api/log"
)

// New returns a new DB.
func New(logger log.Logger) (*DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = `sslmode=disable`
	}
	pqconnector, err := pq.NewConnector(dsn)
	if err != nil {
		return nil, err
	}

	connector := &sqldriver.Connector{
		Connector: pqconnector,
		ExecContextFunc: func(ctx context.Context, query string, args []driver.NamedValue, duration time.Duration, err error) {
			fields := []log.Field{
				log.String("query", query),
				log.Stringer("duration", duration),
			}
			if err != nil {
				fields = append(fields, log.Error("error", err))
			}
			logger.Log(ctx, "sql exec", fields...)
		},
		QueryContextFunc: func(ctx context.Context, query string, args []driver.NamedValue, duration time.Duration, err error) {
			fields := []log.Field{
				log.String("query", query),
				log.Stringer("duration", duration),
			}
			if err != nil {
				fields = append(fields, log.Error("error", err))
			}
			logger.Log(ctx, "sql query", fields...)
		},
	}
	return &DB{DB: sql.OpenDB(connector)}, nil
}

// DB wraps a sql.DB.
type DB struct {
	*sql.DB
}

// QueryStructSlice executes a query and scans the returned rows to dest, which must be a pointer to a slice of structs.
func (db *DB) QueryStructSlice(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
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

// QueryStruct executes a query and scans the returned row to dest, which must be a pointer to a struct.
func (db *DB) QueryStruct(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
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
