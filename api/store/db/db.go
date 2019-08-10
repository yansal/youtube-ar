package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"os"
	"time"

	"github.com/lib/pq"
	"github.com/yansal/sqldriver"
	"github.com/yansal/youtube-ar/api/log"
)

// New returns a new DB.
func New(logger log.Logger) (*sql.DB, error) {
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
	return sql.OpenDB(connector), nil
}
