package sql

import (
	"context"
	"database/sql/driver"

	"github.com/yansal/sql/hooks"
	"github.com/yansal/youtube-ar/api/log"
)

// Wrap wraps driver.Connector.
func Wrap(connector driver.Connector, logger log.Logger) driver.Connector {
	c := hooks.Wrap(connector)

	c.ExecHook = func(ctx context.Context, info hooks.ExecInfo) {
		fields := []log.Field{
			log.String("query", info.Query),
			log.Stringer("duration", info.Duration),
		}
		if info.Err != nil {
			fields = append(fields, log.Error("error", info.Err))
		}
		logger.Log(ctx, "sql exec", fields...)
	}
	c.QueryHook = func(ctx context.Context, info hooks.QueryInfo) {
		fields := []log.Field{
			log.String("query", info.Query),
			log.Stringer("duration", info.Duration),
		}
		if info.Err != nil {
			fields = append(fields, log.Error("error", info.Err))
		}
		logger.Log(ctx, "sql query", fields...)
	}

	c.BeginHook = func(ctx context.Context, info hooks.BeginInfo) {
		fields := []log.Field{
			log.Stringer("duration", info.Duration),
		}
		if info.Err != nil {
			fields = append(fields, log.Error("error", info.Err))
		}
		logger.Log(ctx, "sql begin", fields...)
	}
	c.CommitHook = func(info hooks.CommitInfo) {
		fields := []log.Field{
			log.Stringer("duration", info.Duration),
		}
		if info.Err != nil {
			fields = append(fields, log.Error("error", info.Err))
		}
		logger.Log(context.Background(), "sql commit", fields...)
	}
	c.RollbackHook = func(info hooks.RollbackInfo) {
		fields := []log.Field{
			log.Stringer("duration", info.Duration),
		}
		if info.Err != nil {
			fields = append(fields, log.Error("error", info.Err))
		}
		logger.Log(context.Background(), "sql rollback", fields...)
	}

	return c
}
