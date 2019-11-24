package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/yansal/sql/nest"
	"github.com/yansal/youtube-ar/api/model"
	"github.com/yansal/youtube-ar/api/payload"
	"github.com/yansal/youtube-ar/api/query"
	"github.com/yansal/youtube-ar/api/resource"
	"github.com/yansal/youtube-ar/api/server"
)

// URLSerializer is the serializer interface required by url handlers.
type URLSerializer interface {
	NewURL(url *model.URL) *resource.URL
	NewURLs(urls []model.URL) *resource.URLs
}

// ListURLsManager is the manager interface required by ListURLs.
type ListURLsManager interface {
	ListURLs(context.Context, nest.Querier, *query.URLs) ([]model.URL, error)
}

// ListURLs is the GET /urls handler.
func ListURLs(m ListURLsManager, db nest.Querier, s URLSerializer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serveHTTP(w, r, listURLs(m, db, s))
	}
}

func listURLs(m ListURLsManager, db nest.Querier, s URLSerializer) handlerFunc {
	return func(r *http.Request) (*response, error) {
		q, err := query.ParseURLs(r.URL.Query())
		if err != nil {
			return nil, httpError{
				err:  err,
				code: http.StatusBadRequest,
			}
		}

		ctx := r.Context()
		urls, err := m.ListURLs(ctx, db, q)
		if err != nil {
			return nil, err
		}
		resource := s.NewURLs(urls)
		b, err := json.Marshal(resource)
		if err != nil {
			return nil, err
		}
		return &response{body: b, code: http.StatusOK}, nil
	}
}

// CreateURLManager is the manager interface required by CreateURL.
type CreateURLManager interface {
	CreateURL(context.Context, nest.Querier, payload.URL) (*model.URL, error)
}

// CreateURL is the POST /urls handler.
func CreateURL(m CreateURLManager, db nest.Querier, s URLSerializer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serveHTTP(w, r, createURL(m, db, s))
	}
}

func createURL(m CreateURLManager, db nest.Querier, s URLSerializer) handlerFunc {
	return func(r *http.Request) (*response, error) {
		var payload payload.URL
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			return nil, httpError{
				err:  err,
				code: http.StatusBadRequest,
			}
		}
		if err := payload.Validate(); err != nil {
			return nil, httpError{
				err:  err,
				code: http.StatusBadRequest,
			}
		}

		ctx := r.Context()
		url, err := m.CreateURL(ctx, db, payload)
		if err != nil {
			return nil, err
		}
		resource := s.NewURL(url)
		b, err := json.Marshal(resource)
		if err != nil {
			return nil, err
		}
		return &response{body: b, code: http.StatusCreated}, nil
	}
}

// DetailURLManager is the manager interface required by DetailURL.
type DetailURLManager interface {
	GetURL(context.Context, nest.Querier, int64) (*model.URL, error)
}

// DetailURL is the GET /urls handler.
func DetailURL(m DetailURLManager, db nest.Querier, s URLSerializer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serveHTTP(w, r, detailURL(m, db, s))
	}
}

func detailURL(m DetailURLManager, db nest.Querier, s URLSerializer) handlerFunc {
	return func(r *http.Request) (*response, error) {
		ctx := r.Context()
		match := server.ContextMatch(ctx)
		id, err := strconv.ParseInt(match[1], 0, 0)
		if err != nil {
			return nil, httpError{code: http.StatusNotFound}
		}

		url, err := m.GetURL(ctx, db, id)
		if err == sql.ErrNoRows {
			return nil, httpError{code: http.StatusNotFound}
		} else if err != nil {
			return nil, err
		}
		resource := s.NewURL(url)
		b, err := json.Marshal(resource)
		if err != nil {
			return nil, err
		}
		return &response{body: b, code: http.StatusOK}, nil
	}
}

// DeleteURLManager is the manager interface required by DeleteURL.
type DeleteURLManager interface {
	DeleteURL(context.Context, nest.Querier, int64) error
}

// DeleteURL is the GET /urls handler.
func DeleteURL(m DeleteURLManager, db nest.Querier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serveHTTP(w, r, deleteURL(m, db))
	}
}

func deleteURL(m DeleteURLManager, db nest.Querier) handlerFunc {
	return func(r *http.Request) (*response, error) {
		ctx := r.Context()
		match := server.ContextMatch(ctx)
		id, err := strconv.ParseInt(match[1], 0, 0)
		if err != nil {
			return nil, httpError{code: http.StatusNotFound}
		}

		if err := m.DeleteURL(ctx, db, id); err != nil {
			return nil, err
		}
		return &response{code: http.StatusNoContent}, nil
	}
}

// Retrier is the interface required by RetryDownloadURL.
type Retrier interface {
	RetryDownloadURL(context.Context, nest.Querier, int64) (*model.URL, error)
}

// RetryDownloadURL is the POST /urls/:id/retry handler.
func RetryDownloadURL(retrier Retrier, db nest.Querier, s URLSerializer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serveHTTP(w, r, retryURL(retrier, db, s))
	}
}

func retryURL(retrier Retrier, db nest.Querier, s URLSerializer) handlerFunc {
	return func(r *http.Request) (*response, error) {
		ctx := r.Context()
		match := server.ContextMatch(ctx)
		id, err := strconv.ParseInt(match[1], 0, 0)
		if err != nil {
			return nil, httpError{code: http.StatusNotFound}
		}

		url, err := retrier.RetryDownloadURL(ctx, db, id)
		if err != nil {
			return nil, err
		}

		resource := s.NewURL(url)
		b, err := json.Marshal(resource)
		if err != nil {
			return nil, err
		}
		return &response{body: b, code: http.StatusCreated}, nil
	}
}
