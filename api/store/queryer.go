package store

import (
	"context"
	"database/sql"
)

// Queryer is the db interface used in store functions.
type Queryer interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}
