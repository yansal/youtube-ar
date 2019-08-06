package store

import (
	"context"
	"database/sql"

	"github.com/yansal/youtube-ar/model"
)

// New returns a new store.
func New(db *sql.DB) *Store {
	return &Store{db: db}
}

// Store is a store.
type Store struct {
	db *sql.DB
}

// CreateURL creates url.
func (s *Store) CreateURL(ctx context.Context, url *model.URL) error {
	query := `insert into urls(url, retries) values($1, $2) returning id, created_at, updated_at, status`
	args := []interface{}{url.URL, url.Retries}
	return s.db.QueryRowContext(ctx, query, args...).Scan(&url.ID, &url.CreatedAt, &url.UpdatedAt, &url.Status)
}

// LockURL locks url.
func (s *Store) LockURL(ctx context.Context, url *model.URL) error {
	query := `update urls set status = $1 where id = $2 and status = 'pending' returning url, created_at, updated_at`
	args := []interface{}{url.Status, url.ID}
	return s.db.QueryRowContext(ctx, query, args...).Scan(&url.URL, &url.CreatedAt, &url.UpdatedAt)
}

// UnlockURL unlocks url.
func (s *Store) UnlockURL(ctx context.Context, url *model.URL) error {
	query := `update urls set status = $1, file = $2, error = $3 where id = $4 and status = 'processing' returning created_at, updated_at`
	args := []interface{}{url.Status, url.File, url.Error, url.ID}
	return s.db.QueryRowContext(ctx, query, args...).Scan(&url.CreatedAt, &url.UpdatedAt)
}

// CreateLog create log.
func (s *Store) CreateLog(ctx context.Context, urlID int64, log *model.Log) error {
	query := `update urls set logs = array_append(logs, $1) where id = $2`
	args := []interface{}{log.Log, urlID}
	return s.db.QueryRowContext(ctx, query, args...).Scan()
}

// GetURL gets the url with id.
func (s *Store) GetURL(ctx context.Context, id int64) (*model.URL, error) {
	var (
		query = `select id, url, created_at, updated_at, status, error, file, retries, logs from urls where id = $1`
		args  = []interface{}{id}
		url   model.URL
	)
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&url.ID, &url.URL, &url.CreatedAt, &url.UpdatedAt, &url.Status, &url.Error, &url.File, &url.Retries, &url.Logs,
	); err != nil {
		return nil, err
	}
	return &url, nil
}

// ListURLs lists urls.
func (s *Store) ListURLs(ctx context.Context, page *model.Page) ([]model.URL, error) {
	// TODO: add filters
	var (
		query string
		args  []interface{}
	)
	if page.Cursor == 0 {
		query = `select id, url, created_at, updated_at, status, error, file, retries from urls order by id desc limit $1`
		args = []interface{}{page.Limit}
	} else {
		query = `select id, url, created_at, updated_at, status, error, file, retries from urls where id < $1 order by id desc limit $2`
		args = []interface{}{page.Cursor, page.Limit}
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	var urls []model.URL
	for rows.Next() {
		var url model.URL
		if err := rows.Scan(&url.ID, &url.URL, &url.CreatedAt, &url.UpdatedAt, &url.Status, &url.Error, &url.File, &url.Retries); err != nil {
			return nil, err
		}
		urls = append(urls, url)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return urls, nil
}

// ListLogs list logs.
func (s *Store) ListLogs(ctx context.Context, urlID int64, page *model.Page) ([]model.Log, error) {
	query := `select unnest(logs[$1:$2]) from urls where id = $3`
	cursor := page.Cursor + 1
	args := []interface{}{cursor, cursor + page.Limit - 1, urlID}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	var logs []model.Log
	for rows.Next() {
		var log model.Log
		if err := rows.Scan(&log.Log); err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return logs, nil
}

// CreateYoutubeVideo creates v.
func (s *Store) CreateYoutubeVideo(ctx context.Context, v *model.YoutubeVideo) error {
	query := `insert into youtube_videos(youtube_id) values($1) on conflict do nothing returning id, created_at`
	args := []interface{}{v.YoutubeID}
	return s.db.QueryRowContext(ctx, query, args...).Scan(&v.ID, &v.CreatedAt)
}
