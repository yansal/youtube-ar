package sql

import (
	"context"
	"database/sql/driver"
	"time"

	"github.com/yansal/sqldriver"
	"github.com/yansal/youtube-ar/api/log"
)

// Wrap wraps driver.Connector.
func Wrap(connector driver.Connector, logger log.Logger) driver.Connector {
	return &sqldriver.Connector{
		Connector: connector,
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
}
