package manager

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/yansal/youtube-ar/api/event"
	"github.com/yansal/youtube-ar/api/model"
	"github.com/yansal/youtube-ar/api/payload"
	"github.com/yansal/youtube-ar/api/query"
)

// Server is the manager used for server features.
type Server struct {
	broker BrokerServer
	store  StoreServer
}

// BrokerServer is the broker interface required by Server.
type BrokerServer interface {
	Send(context.Context, string, string) error
}

// StoreServer is the store interface required by Server.
type StoreServer interface {
	CreateURL(context.Context, *model.URL) error
	GetURL(context.Context, int64) (*model.URL, error)
	ListURLs(context.Context, *query.URLs) ([]model.URL, error)
	ListLogs(context.Context, int64, *query.Logs) ([]model.Log, error)
}

// NewServer returns a new Server.
func NewServer(broker BrokerServer, store StoreServer) *Server {
	return &Server{broker: broker, store: store}
}

// CreateURL creates an URL.
func (m *Server) CreateURL(ctx context.Context, p payload.URL) (*model.URL, error) {
	url := &model.URL{URL: p.URL}
	if p.Retries != 0 {
		url.Retries = sql.NullInt64{Valid: true, Int64: p.Retries}
	}
	if err := m.store.CreateURL(ctx, url); err != nil {
		return nil, err
	}

	e := &event.URL{ID: url.ID}
	b, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}
	return url, m.broker.Send(ctx, "url-created", string(b))
}

// GetURL gets an url.
func (m *Server) GetURL(ctx context.Context, id int64) (*model.URL, error) {
	return m.store.GetURL(ctx, id)
}

// ListURLs lists urls.
func (m *Server) ListURLs(ctx context.Context, q *query.URLs) ([]model.URL, error) {
	return m.store.ListURLs(ctx, q)
}

// ListLogs lists logs.
func (m *Server) ListLogs(ctx context.Context, urlID int64, q *query.Logs) ([]model.Log, error) {
	return m.store.ListLogs(ctx, urlID, q)
}
