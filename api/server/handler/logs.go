package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/yansal/youtube-ar/api/model"
	"github.com/yansal/youtube-ar/api/query"
	"github.com/yansal/youtube-ar/api/resource"
	"github.com/yansal/youtube-ar/api/server"
	storesql "github.com/yansal/youtube-ar/api/store/sql"
)

// LogSerializer is the serializer interface required by log handlers.
type LogSerializer interface {
	NewLogs(logs []model.Log, cursor int64) *resource.Logs
}

// ListLogsManager is the manager interface required by ListLogs.
type ListLogsManager interface {
	ListLogs(context.Context, storesql.QueryStructSlicer, int64, *query.Logs) ([]model.Log, error)
}

// ListLogs is the GET /urls/:id/logs handler.
func ListLogs(m ListLogsManager, db storesql.QueryStructSlicer, s LogSerializer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serveHTTP(w, r, listLogs(m, db, s))
	}
}

func listLogs(m ListLogsManager, db storesql.QueryStructSlicer, s LogSerializer) handlerFunc {
	return func(r *http.Request) (*response, error) {
		ctx := r.Context()
		match := server.ContextMatch(ctx)
		id, err := strconv.ParseInt(match[1], 0, 0)
		if err != nil {
			return nil, httpError{code: http.StatusNotFound}
		}

		q, err := query.ParseLogs(r.URL.Query())
		if err != nil {
			return nil, httpError{
				err:  err,
				code: http.StatusBadRequest,
			}
		}

		logs, err := m.ListLogs(ctx, db, id, q)
		if err != nil {
			return nil, err
		}
		resource := s.NewLogs(logs, q.Cursor)
		b, err := json.Marshal(resource)
		if err != nil {
			return nil, err
		}
		return &response{body: b, code: http.StatusOK}, nil
	}
}
