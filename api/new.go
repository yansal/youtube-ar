package main

import (
	"database/sql"
	"os"

	"github.com/go-redis/redis"
	"github.com/lib/pq"
	brokerredis "github.com/yansal/youtube-ar/api/broker/redis"
	"github.com/yansal/youtube-ar/api/log"
	logsql "github.com/yansal/youtube-ar/api/log/sql"
	storesql "github.com/yansal/youtube-ar/api/store/sql"
)

func newSQLDB(log log.Logger) (*storesql.DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = `sslmode=disable`
	}
	pqconnector, err := pq.NewConnector(dsn)
	if err != nil {
		return nil, err
	}

	connector := logsql.Wrap(pqconnector, log)

	return storesql.NewDB(sql.OpenDB(connector)), nil
}

func newRedis(log log.Logger) (*redis.Client, error) {
	url := os.Getenv("REDIS_URL")
	if url == "" {
		url = `redis://`
	}
	return brokerredis.New(url, log)
}
