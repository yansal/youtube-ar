package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/yansal/youtube-ar/model"
	"github.com/yansal/youtube-ar/payload"
	"github.com/yansal/youtube-ar/query"
	"github.com/yansal/youtube-ar/resource"
)

// ListURLsManager is the manager interface required by ListURLs.
type ListURLsManager interface {
	ListURLs(context.Context, *model.Page) ([]model.URL, error)
}

// ListURLs is the GET /urls handler.
func ListURLs(m ListURLsManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serveHTTP(w, r, listURLs(m))
	}
}

func listURLs(m ListURLsManager) handlerFunc {
	return func(r *http.Request) (*response, error) {
		page, err := query.ParsePage(r.URL.Query())
		if err != nil {
			return nil, httpError{
				err:  err,
				code: http.StatusBadRequest,
			}
		}

		ctx := r.Context()
		urls, err := m.ListURLs(ctx, page)
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
