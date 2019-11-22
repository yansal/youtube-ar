package main

import (
	"context"
	"database/sql"
	"os"

	"github.com/go-redis/redis"
	"github.com/lib/pq"
	brokerredis "github.com/yansal/youtube-ar/api/broker/redis"
	"github.com/yansal/youtube-ar/api/log"
	logsql "github.com/yansal/youtube-ar/api/log/sql"
)

func newSQLDB(log log.Logger) (*sql.DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = `sslmode=disable`
	}
	pqconnector, err := pq.NewConnector(dsn)
	if err != nil {
		return nil, err
	}

	connector := logsql.Wrap(pqconnector, log)

	db := sql.OpenDB(connector)
	if err := db.PingContext(context.Background()); err != nil {
		return nil, err
	}
	return db, nil
}

func newRedis(log log.Logger) (*redis.Client, error) {
	url := os.Getenv("REDIS_URL")
	if url == "" {
		url = `redis://`
	}
	client, err := brokerredis.New(url, log)
	if err != nil {
		return nil, err
	}
	if err := client.Ping().Err(); err != nil {
		return nil, err
	}
	return client, nil
}
