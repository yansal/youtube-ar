package cmd

import (
	"database/sql"
	"os"

	"github.com/lib/pq"
	"github.com/yansal/youtube-ar/api/log"
	logsql "github.com/yansal/youtube-ar/api/log/sql"
	storesql "github.com/yansal/youtube-ar/api/store/sql"
)

// newSQLDB returns a new sql DB.
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

	return &storesql.DB{DB: sql.OpenDB(connector)}, nil
}
