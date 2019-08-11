package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/yansal/youtube-ar/api/model"
	"github.com/yansal/youtube-ar/api/query"
	"github.com/yansal/youtube-ar/api/store/querybuilder"
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
	query, args := querybuilder.NewInsert("urls", []string{"url", "retries"}).
		Values(url.URL, url.Retries).
		Returning("id", "created_at", "updated_at", "status").
		Build()
	return s.db.QueryRowContext(ctx, query, args...).Scan(&url.ID, &url.CreatedAt, &url.UpdatedAt, &url.Status)
}

// LockURL locks url.
func (s *Store) LockURL(ctx context.Context, url *model.URL) error {
	query, args := querybuilder.NewUpdate("urls",
		map[string]interface{}{"status": url.Status}).
		Where(querybuilder.NewBoolExpr(
			querybuilder.NewIdentifier("id").Equal(url.ID)).
			And(querybuilder.NewIdentifier("status").Equal("pending")),
		).
		Returning("url", "created_at", "updated_at").
		Build()
	return s.db.QueryRowContext(ctx, query, args...).Scan(&url.URL, &url.CreatedAt, &url.UpdatedAt)
}

// UnlockURL unlocks url.
func (s *Store) UnlockURL(ctx context.Context, url *model.URL) error {
	query, args := querybuilder.NewUpdate("urls",
		map[string]interface{}{"status": url.Status, "file": url.File, "error": url.Error}).
		Where(querybuilder.NewBoolExpr(
			querybuilder.NewIdentifier("id").Equal(url.ID)).
			And(querybuilder.NewIdentifier("status").Equal("processing")),
		).
		Returning("created_at", "updated_at").
		Build()
	return s.db.QueryRowContext(ctx, query, args...).Scan(&url.CreatedAt, &url.UpdatedAt)
}

// AppendLog create log.
func (s *Store) AppendLog(ctx context.Context, urlID int64, log *model.Log) error {
	query, args := querybuilder.NewUpdate("urls",
		map[string]interface{}{"logs": querybuilder.NewCallExpr("array_append", "logs", querybuilder.NewBindValue(log.Log))}).
		Where(querybuilder.NewIdentifier("id").Equal(urlID)).
		Build()
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

// GetURL gets the url with id.
func (s *Store) GetURL(ctx context.Context, id int64) (*model.URL, error) {
	query, args := querybuilder.NewSelect("id", "url", "created_at", "updated_at", "status", "error", "file", "retries", "logs").
		From("urls").
		Where(querybuilder.NewBoolExpr(
			querybuilder.NewIdentifier("id").Equal(id)).And(
			querybuilder.NewIdentifier("deleted_at").IsNull())).
		Build()

	var url model.URL
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&url.ID, &url.URL, &url.CreatedAt, &url.UpdatedAt, &url.Status, &url.Error, &url.File, &url.Retries, &url.Logs,
	); err != nil {
		return nil, err
	}
	return &url, nil
}

// DeleteURL deletes the url with id.
func (s *Store) DeleteURL(ctx context.Context, id int64) error {
	query, args := querybuilder.NewUpdate("urls",
		map[string]interface{}{"deleted_at": time.Now()}).
		Where(querybuilder.NewIdentifier("id").Equal(id)).
		Build()

	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

// ListURLs lists urls.
func (s *Store) ListURLs(ctx context.Context, q *query.URLs) ([]model.URL, error) {
	query, args := buildListURLs(q)
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

func buildListURLs(q *query.URLs) (string, []interface{}) {
	stmt := querybuilder.NewSelect(
		"id", "url", "created_at", "updated_at", "status", "error", "file", "retries",
	).From("urls")

	expr := querybuilder.NewIdentifier("deleted_at").IsNull()
	if q.Status != nil {
		expr = querybuilder.NewBoolExpr(expr).And(
			querybuilder.NewIdentifier("status").In(q.Status),
		)
	}
	if q.Cursor != 0 {
		expr = querybuilder.NewBoolExpr(expr).And(
			querybuilder.NewIdentifier("id").LessThan(q.Cursor),
		)
	}

	return stmt.Where(expr).OrderBy("id desc").Limit(q.Limit).Build()
}

// ListLogs list logs.
func (s *Store) ListLogs(ctx context.Context, urlID int64, q *query.Logs) ([]model.Log, error) {
	query, args := querybuilder.NewSelect(querybuilder.NewCallExpr("unnest", "logs")).
		From("urls").
		Where(querybuilder.NewIdentifier("id").Equal(urlID)).
		Offset(q.Cursor).
		Build()

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
	query, args := querybuilder.NewInsert("youtube_videos", []string{"youtube_id"}).
		Values(v.YoutubeID).
		Returning("id", "created_at").
		Build()

	return s.db.QueryRowContext(ctx, query, args...).Scan(&v.ID, &v.CreatedAt)
}

// GetYoutubeVideoByYoutubeID gets a youtube video from a youtube id.
func (s *Store) GetYoutubeVideoByYoutubeID(ctx context.Context, youtubeID string) (*model.YoutubeVideo, error) {
	query, args := querybuilder.NewSelect("id", "youtube_id", "created_at").
		From("youtube_videos").
		Where(querybuilder.NewIdentifier("youtube_id").Equal(youtubeID)).
		Build()

	var v model.YoutubeVideo
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&v.ID, &v.YoutubeID, &v.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &v, nil
}
