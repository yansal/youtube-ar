package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/yansal/youtube-ar/model"
	"github.com/yansal/youtube-ar/query"
	"github.com/yansal/youtube-ar/resource"
	"github.com/yansal/youtube-ar/server"
)

// ListLogsManager is the manager interface required by ListLogs.
type ListLogsManager interface {
	ListLogs(context.Context, int64, *model.Page) ([]model.Log, error)
}

// ListLogs is the GET /urls/:id/logs handler.
func ListLogs(m ListLogsManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serveHTTP(w, r, listLogs(m))
	}
}

func listLogs(m ListLogsManager) handlerFunc {
	return func(r *http.Request) (*response, error) {
		ctx := r.Context()
		match := server.ContextMatch(ctx)
		id, err := strconv.ParseInt(match[1], 0, 0)
		if err != nil {
			return nil, httpError{code: http.StatusNotFound}
		}

		page, err := query.ParsePage(r.URL.Query())
		if err != nil {
			return nil, httpError{
				err:  err,
				code: http.StatusBadRequest,
			}
		}

		logs, err := m.ListLogs(ctx, id, page)
		if err != nil {
			return nil, err
		}
		resource := resource.NewLogs(logs)
		b, err := json.Marshal(resource)
		if err != nil {
			return nil, err
		}
		return &response{body: b, code: http.StatusOK}, nil
	}
}
