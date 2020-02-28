package store

import (
	"context"
	"time"

	"github.com/yansal/sql/build"
	"github.com/yansal/sql/nest"
	"github.com/yansal/sql/scan"
	"github.com/yansal/youtube-ar/api/model"
	"github.com/yansal/youtube-ar/api/query"
)

// New returns a new store.
func New() *Store {
	return &Store{}
}

// Store is a store.
type Store struct{}

// CreateURL creates url.
func (*Store) CreateURL(ctx context.Context, db nest.Querier, url *model.URL) error {
	query, args := build.InsertInto("urls").
		Values(
			build.Value("url", build.Bind(url.URL)),
			build.Value("retries", build.Bind(url.Retries)),
		).
		Returning(build.Columns(url.Columns()...)...).
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
	query, args := build.Update("urls").
		Set(build.Value("status", build.Bind(url.Status))).
		Where(build.Infix(build.Ident("id")).Equal(build.Bind(url.ID)).
			And(build.Infix(build.Ident("status")).Equal(build.String("pending")))).
		Build()
	_, err := db.ExecContext(ctx, query, args...)
	return err
}

// UnlockURL unlocks url.
func (*Store) UnlockURL(ctx context.Context, db nest.Querier, url *model.URL) error {
	query, args := build.Update("urls").
		Set(
			build.Value("status", build.Bind(url.Status)),
			build.Value("file", build.Bind(url.File)),
			build.Value("error", build.Bind(url.Error)),
		).
		Where(build.Infix(build.Ident("id")).Equal(build.Bind(url.ID)).
			And(build.Infix(build.Ident("status")).Equal(build.String("processing"))),
		).
		Build()
	_, err := db.ExecContext(ctx, query, args...)
	return err
}

// SetOEmbed sets oembed.
func (*Store) SetOEmbed(ctx context.Context, db nest.Querier, url *model.URL) error {
	query, args := build.Update("urls").
		Set(build.Value("oembed", build.Bind(url.OEmbed))).
		Where(build.Infix(build.Ident("id")).Equal(build.Bind(url.ID))).
		Build()
	_, err := db.ExecContext(ctx, query, args...)
	return err
}

// AppendLog create log.
func (*Store) AppendLog(ctx context.Context, db nest.Querier, urlID int64, log *model.Log) error {
	query, args := build.Update("urls").
		Set(build.Value("logs", build.CallExpr("array_append", build.Ident("logs"), build.Bind(log.Log)))).
		Where(build.Infix(build.Ident("id")).Equal(build.Bind(urlID))).
		Build()
	_, err := db.ExecContext(ctx, query, args...)
	return err
}

// GetURL gets the url with id.
func (s *Store) GetURL(ctx context.Context, db nest.Querier, id int64) (*model.URL, error) {
	var url model.URL
	query, args := build.Select(build.Columns(url.Columns()...)...).
		From(build.Ident("urls")).
		Where(build.Infix(build.Ident("id")).Equal(build.Bind(id)).
			And(build.Ident("deleted_at")).IsNull()).
		Build()

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if err := scan.Struct(rows, &url); err != nil {
		return nil, err
	}
	return &url, nil
}

// DeleteURL deletes the url with id.
func (*Store) DeleteURL(ctx context.Context, db nest.Querier, id int64) error {
	query, args := build.Update("urls").
		Set(build.Value("deleted_at", build.Bind(time.Now()))).
		Where(build.Infix(build.Ident("id")).Equal(build.Bind(id))).
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
	var url model.URL
	cmd := build.Select(build.Columns(url.Columns()...)...)
	if q.Q != "" {
		cmd = cmd.From(
			build.Ident("urls"),
			build.FromExpr(
				build.CallExpr("websearch_to_tsquery", build.Bind(q.Q)),
			).As("tsquery"),
		)
	} else {
		cmd = cmd.From(build.Ident("urls"))
	}

	expr := build.Infix(build.Ident("deleted_at")).IsNull()
	if q.Status != nil {
		expr = expr.And(build.Ident("status")).In(build.Bind(q.Status))
	}
	if q.Cursor != 0 {
		expr = expr.And(build.Ident("id")).LessThan(build.Bind(q.Cursor))
	}
	if q.Q != "" {
		expr = expr.And(build.Ident("tsv")).Op("@@", build.Ident("tsquery"))
	}
	cmd = cmd.Where(expr)

	if q.Q != "" {
		cmd = cmd.OrderBy(
			build.OrderExpr(build.CallExpr("ts_rank", build.Ident("tsv"), build.Ident("tsquery")), build.Desc),
			build.OrderExpr(build.Ident("id"), build.Desc),
		)
	} else {
		cmd = cmd.OrderBy(
			build.OrderExpr(build.Ident("id"), build.Desc),
		)
	}

	return cmd.Limit(build.Bind(q.Limit)).
		Build()
}

// ListLogs list logs.
func (s *Store) ListLogs(ctx context.Context, db nest.Querier, urlID int64, q *query.Logs) ([]model.Log, error) {
	query, args := build.Select(
		build.ColumnExpr(build.CallExpr("unnest", build.Ident("logs"))).As("log"),
	).
		From(build.Ident("urls")).
		Where(build.Infix(build.Ident("id")).Equal(build.Bind(urlID))).
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
	query, args := build.InsertInto("youtube_videos").
		Values(build.Value("youtube_id", build.Bind(v.YoutubeID))).
		Returning(build.Columns(v.Columns()...)...).
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
	query, args := build.Select(build.Columns(
		"id",
		"youtube_id",
		"created_at",
	)...).
		From(build.Ident("youtube_videos")).
		Where(build.Infix(build.Ident("youtube_id")).Equal(build.Bind(youtubeID))).
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
