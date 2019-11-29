package store

import (
	"context"
	"time"

	"github.com/yansal/sql/build"
	"github.com/yansal/sql/nest"
	"github.com/yansal/sql/scan"
	"github.com/yansal/youtube-ar/api/model"
	"github.com/yansal/youtube-ar/api/query"
	"github.com/yansal/youtube-ar/api/store/querybuilder"
)

// New returns a new store.
func New() *Store {
	return &Store{}
}

// Store is a store.
type Store struct{}

// CreateURL creates url.
func (*Store) CreateURL(ctx context.Context, db nest.Querier, url *model.URL) error {
	query, args := querybuilder.Insert("urls", []string{"url", "retries"}).
		Values(url.URL, url.Retries).
		Returning("id", "created_at", "updated_at", "status").
		Build()

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	return scan.Struct(rows, url)
}

// LockURL locks url.
func (*Store) LockURL(ctx context.Context, db nest.Querier, url *model.URL) error {
	query, args := querybuilder.Update("urls").
		Set(map[string]interface{}{"status": url.Status}).
		Where(querybuilder.Expr(
			querybuilder.Expr("id").Equal(url.ID),
		).And(
			querybuilder.Expr("status").Equal("pending"),
		)).Build()
	_, err := db.ExecContext(ctx, query, args...)
	return err
}

// UnlockURL unlocks url.
func (*Store) UnlockURL(ctx context.Context, db nest.Querier, url *model.URL) error {
	query, args := querybuilder.Update("urls").
		Set(map[string]interface{}{"status": url.Status, "file": url.File, "error": url.Error}).
		Where(querybuilder.Expr(
			querybuilder.Expr("id").Equal(url.ID),
		).And(
			querybuilder.Expr("status").Equal("processing"),
		)).Build()
	_, err := db.ExecContext(ctx, query, args...)
	return err
}

// SetOEmbed sets oembed.
func (*Store) SetOEmbed(ctx context.Context, db nest.Querier, url *model.URL) error {
	query, args := querybuilder.Update("urls").
		Set(map[string]interface{}{"oembed": url.OEmbed}).
		Where(querybuilder.Expr("id").Equal(url.ID)).
		Build()
	_, err := db.ExecContext(ctx, query, args...)
	return err
}

// AppendLog create log.
func (*Store) AppendLog(ctx context.Context, db nest.Querier, urlID int64, log *model.Log) error {
	query, args := querybuilder.Update("urls").
		Set(map[string]interface{}{"logs": querybuilder.Call("array_append", "logs", querybuilder.Bind(log.Log))}).
		Where(querybuilder.Expr("id").Equal(urlID)).
		Build()
	_, err := db.ExecContext(ctx, query, args...)
	return err
}

// GetURL gets the url with id.
func (s *Store) GetURL(ctx context.Context, db nest.Querier, id int64) (*model.URL, error) {
	query, args := build.Select(
		"id",
		"url",
		"created_at",
		"updated_at",
		"status",
		"error",
		"file",
		"retries",
		"logs",
		"oembed",
	).
		From("urls").
		Where(build.Expr("id").Equal(build.Bind(id)).And("deleted_at").IsNull()).
		Build()

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var url model.URL
	if err := scan.Struct(rows, &url); err != nil {
		return nil, err
	}
	return &url, nil
}

// DeleteURL deletes the url with id.
func (*Store) DeleteURL(ctx context.Context, db nest.Querier, id int64) error {
	query, args := querybuilder.Update("urls").
		Set(map[string]interface{}{"deleted_at": time.Now()}).
		Where(querybuilder.Expr("id").Equal(id)).
		Build()

	_, err := db.ExecContext(ctx, query, args...)
	return err
}

// ListURLs lists urls.
func (*Store) ListURLs(ctx context.Context, db nest.Querier, q *query.URLs) ([]model.URL, error) {
	query, args := buildListURLs(q)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var urls []model.URL
	if err := scan.StructSlice(rows, &urls); err != nil {
		return nil, err
	}
	return urls, nil
}

func buildListURLs(q *query.URLs) (string, []interface{}) {
	stmt := build.Select(
		"id", "url", "created_at", "updated_at", "status", "error", "file", "retries", "oembed",
	)
	if q.Q != "" {
		stmt = stmt.From(
			"urls",
			build.FromExpr(
				build.CallExpr("websearch_to_tsquery", build.Bind(q.Q)),
			).As("tsquery"),
		)
	} else {
		stmt = stmt.From("urls")
	}

	expr := build.Expr("deleted_at").IsNull()
	if q.Status != nil {
		expr = expr.And("status").In(build.Bind(q.Status))
	}
	if q.Cursor != 0 {
		expr = expr.And("id").LessThan(build.Bind(q.Cursor))
	}
	if q.Q != "" {
		expr = expr.And("tsv").Op("@@", "tsquery")
	}
	stmt = stmt.Where(expr)

	if q.Q != "" {
		stmt = stmt.OrderBy(
			build.OrderExpr("ts_rank(tsv, tsquery)", build.Desc),
			build.OrderExpr("id", build.Desc),
		)
	} else {
		stmt = stmt.OrderBy(
			build.OrderExpr("id", build.Desc),
		)
	}

	return stmt.Limit(build.Bind(q.Limit)).
		Build()
}

// ListLogs list logs.
func (s *Store) ListLogs(ctx context.Context, db nest.Querier, urlID int64, q *query.Logs) ([]model.Log, error) {
	query, args := build.Select("unnest(logs) as log").
		From("urls").
		Where(build.Expr("id").Equal(build.Bind(urlID))).
		Offset(build.Bind(q.Cursor)).
		Build()

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []model.Log
	if err := scan.StructSlice(rows, &logs); err != nil {
		return nil, err
	}
	return logs, nil
}

// CreateYoutubeVideo creates v.
func (*Store) CreateYoutubeVideo(ctx context.Context, db nest.Querier, v *model.YoutubeVideo) error {
	query, args := querybuilder.Insert("youtube_videos", []string{"youtube_id"}).
		Values(v.YoutubeID).
		Returning("id", "created_at").
		Build()

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	return scan.Struct(rows, v)
}

// GetYoutubeVideoByYoutubeID gets a youtube video from a youtube id.
func (*Store) GetYoutubeVideoByYoutubeID(ctx context.Context, db nest.Querier, youtubeID string) (*model.YoutubeVideo, error) {
	query, args := build.Select("id", "youtube_id", "created_at").
		From("youtube_videos").
		Where(build.Expr("youtube_id").Equal(build.Bind(youtubeID))).
		Build()

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var v model.YoutubeVideo
	if err := scan.Struct(rows, &v); err != nil {
		return nil, err
	}
	return &v, nil
}
