package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/yansal/youtube-ar/api/model"
	"github.com/yansal/youtube-ar/api/payload"
	"github.com/yansal/youtube-ar/api/query"
	"github.com/yansal/youtube-ar/api/resource"
	"github.com/yansal/youtube-ar/api/server"
)

// ListURLsManager is the manager interface required by ListURLs.
type ListURLsManager interface {
	ListURLs(context.Context, *query.URLs) ([]model.URL, error)
}

// ListURLs is the GET /urls handler.
func ListURLs(m ListURLsManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serveHTTP(w, r, listURLs(m))
	}
}

func listURLs(m ListURLsManager) handlerFunc {
	return func(r *http.Request) (*response, error) {
		q, err := query.ParseURLs(r.URL.Query())
		if err != nil {
			return nil, httpError{
				err:  err,
				code: http.StatusBadRequest,
			}
		}

		ctx := r.Context()
		urls, err := m.ListURLs(ctx, q)
		if err != nil {
			return nil, err
		}
		resource := resource.NewURLs(urls)
		b, err := json.Marshal(resource)
		if err != nil {
			return nil, err
		}
		return &response{body: b, code: http.StatusOK}, nil
	}
}

// DetailURLManager is the manager interface required by DetailURL.
type DetailURLManager interface {
	GetURL(context.Context, int64) (*model.URL, error)
}

// DetailURL is the GET /urls handler.
func DetailURL(m DetailURLManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serveHTTP(w, r, detailURL(m))
	}
}

func detailURL(m DetailURLManager) handlerFunc {
	return func(r *http.Request) (*response, error) {
		ctx := r.Context()
		match := server.ContextMatch(ctx)
		id, err := strconv.ParseInt(match[1], 0, 0)
		if err != nil {
			return nil, httpError{code: http.StatusNotFound}
		}

		url, err := m.GetURL(ctx, id)
		if err != nil {
			return nil, err
		}
		resource := resource.NewURL(url)
		b, err := json.Marshal(resource)
		if err != nil {
			return nil, err
		}
		return &response{body: b, code: http.StatusOK}, nil
	}
}

// DeleteURLManager is the manager interface required by DeleteURL.
type DeleteURLManager interface {
	DeleteURL(context.Context, int64) error
}

// DeleteURL is the GET /urls handler.
func DeleteURL(m DeleteURLManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serveHTTP(w, r, deleteURL(m))
	}
}

func deleteURL(m DeleteURLManager) handlerFunc {
	return func(r *http.Request) (*response, error) {
		ctx := r.Context()
		match := server.ContextMatch(ctx)
		id, err := strconv.ParseInt(match[1], 0, 0)
		if err != nil {
			return nil, httpError{code: http.StatusNotFound}
		}

		if err := m.DeleteURL(ctx, id); err != nil {
			return nil, err
		}
		return &response{code: http.StatusNoContent}, nil
	}
}

// CreateURLManager is the manager interface required by CreateURL.
type CreateURLManager interface {
	CreateURL(context.Context, payload.URL) (*model.URL, error)
}

// CreateURL is the POST /urls handler.
func CreateURL(m CreateURLManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serveHTTP(w, r, createURL(m))
	}
}

func createURL(m CreateURLManager) handlerFunc {
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
		url, err := m.CreateURL(ctx, payload)
		if err != nil {
			return nil, err
		}
		resource := resource.NewURL(url)
		b, err := json.Marshal(resource)
		if err != nil {
			return nil, err
		}
		return &response{body: b, code: http.StatusCreated}, nil
	}
}
