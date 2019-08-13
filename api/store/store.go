package store

import (
	"context"
	"time"

	"github.com/yansal/youtube-ar/api/model"
	"github.com/yansal/youtube-ar/api/query"
	"github.com/yansal/youtube-ar/api/store/querybuilder"
	"github.com/yansal/youtube-ar/api/store/sql"
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
	return s.db.QueryStruct(ctx, url, query, args...)
}

// LockURL locks url.
func (s *Store) LockURL(ctx context.Context, url *model.URL) error {
	query, args := querybuilder.NewUpdate("urls").
		Set(map[string]interface{}{"status": url.Status}).
		Where(querybuilder.NewBoolExpr(
			querybuilder.NewIdentifier("id").Equal(url.ID)).
			And(querybuilder.NewIdentifier("status").Equal("pending")),
		).
		Build()
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

// UnlockURL unlocks url.
func (s *Store) UnlockURL(ctx context.Context, url *model.URL) error {
	query, args := querybuilder.NewUpdate("urls").
		Set(map[string]interface{}{"status": url.Status, "file": url.File, "error": url.Error}).
		Where(querybuilder.NewBoolExpr(
			querybuilder.NewIdentifier("id").Equal(url.ID)).
			And(querybuilder.NewIdentifier("status").Equal("processing")),
		).
		Build()
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

// SetOEmbed sets oembed.
func (s *Store) SetOEmbed(ctx context.Context, url *model.URL) error {
	query, args := querybuilder.NewUpdate("urls").
		Set(map[string]interface{}{"oembed": url.OEmbed}).
		Where(querybuilder.NewIdentifier("id").Equal(url.ID)).
		Build()
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

// AppendLog create log.
func (s *Store) AppendLog(ctx context.Context, urlID int64, log *model.Log) error {
	query, args := querybuilder.NewUpdate("urls").
		Set(map[string]interface{}{"logs": querybuilder.NewCallExpr("array_append", "logs", querybuilder.NewBindValue(log.Log))}).
		Where(querybuilder.NewIdentifier("id").Equal(urlID)).
		Build()
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

// GetURL gets the url with id.
func (s *Store) GetURL(ctx context.Context, id int64) (*model.URL, error) {
	query, args := querybuilder.NewSelect("id", "url", "created_at", "updated_at", "status", "error", "file", "retries", "logs", "oembed").
		From("urls").
		Where(querybuilder.NewBoolExpr(
			querybuilder.NewIdentifier("id").Equal(id)).And(
			querybuilder.NewIdentifier("deleted_at").IsNull())).
		Build()

	var url model.URL
	if err := s.db.QueryStruct(ctx, &url, query, args...); err != nil {
		return nil, err
	}
	return &url, nil
}

// DeleteURL deletes the url with id.
func (s *Store) DeleteURL(ctx context.Context, id int64) error {
	query, args := querybuilder.NewUpdate("urls").
		Set(map[string]interface{}{"deleted_at": time.Now()}).
		Where(querybuilder.NewIdentifier("id").Equal(id)).
		Build()

	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

// ListURLs lists urls.
func (s *Store) ListURLs(ctx context.Context, q *query.URLs) ([]model.URL, error) {
	query, args := buildListURLs(q)
	var urls []model.URL
	if err := s.db.QueryStructSlice(ctx, &urls, query, args...); err != nil {
		return nil, err
	}
	return urls, nil
}

func buildListURLs(q *query.URLs) (string, []interface{}) {
	stmt := querybuilder.NewSelect(
		"id", "url", "created_at", "updated_at", "status", "error", "file", "retries", "oembed",
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
	query, args := querybuilder.NewSelect("unnest(logs) as log").
		From("urls").
		Where(querybuilder.NewIdentifier("id").Equal(urlID)).
		Offset(q.Cursor).
		Build()

	var logs []model.Log
	if err := s.db.QueryStructSlice(ctx, &logs, query, args...); err != nil {
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

	return s.db.QueryStruct(ctx, v, query, args...)
}

// GetYoutubeVideoByYoutubeID gets a youtube video from a youtube id.
func (s *Store) GetYoutubeVideoByYoutubeID(ctx context.Context, youtubeID string) (*model.YoutubeVideo, error) {
	query, args := querybuilder.NewSelect("id", "youtube_id", "created_at").
		From("youtube_videos").
		Where(querybuilder.NewIdentifier("youtube_id").Equal(youtubeID)).
		Build()

	var v model.YoutubeVideo
	if err := s.db.QueryStruct(ctx, &v, query, args...); err != nil {
		return nil, err
	}
	return &v, nil
}
