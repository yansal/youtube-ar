package manager

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/yansal/sql/scan"
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
	CreateURL(context.Context, scan.Queryer, *model.URL) error
	GetURL(context.Context, scan.Queryer, int64) (*model.URL, error)
	DeleteURL(context.Context, scan.Queryer, int64) error
	ListURLs(context.Context, scan.Queryer, *query.URLs) ([]model.URL, error)
	ListLogs(context.Context, scan.Queryer, int64, *query.Logs) ([]model.Log, error)
}

// NewServer returns a new Server.
func NewServer(broker BrokerServer, store StoreServer) *Server {
	return &Server{broker: broker, store: store}
}

// CreateURL creates an URL.
func (m *Server) CreateURL(ctx context.Context, db scan.Queryer, p payload.URL) (*model.URL, error) {
	url := &model.URL{URL: p.URL}
	if p.Retries != 0 {
		url.Retries = sql.NullInt64{Valid: true, Int64: p.Retries}
	}
	if err := m.store.CreateURL(ctx, db, url); err != nil {
		return nil, err
	}

	e := &event.URL{ID: url.ID, URL: url.URL}
	b, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}
	if err := m.broker.Send(ctx, "download-url", string(b)); err != nil {
		// TODO: log err
	}
	if err := m.broker.Send(ctx, "get-oembed", string(b)); err != nil {
		// TODO: log err
	}
	return url, nil
}

// GetURL gets an url.
func (m *Server) GetURL(ctx context.Context, db scan.Queryer, id int64) (*model.URL, error) {
	return m.store.GetURL(ctx, db, id)
}

// DeleteURL deletes an url.
func (m *Server) DeleteURL(ctx context.Context, db scan.Queryer, id int64) error {
	return m.store.DeleteURL(ctx, db, id)
}

// ListURLs lists urls.
func (m *Server) ListURLs(ctx context.Context, db scan.Queryer, q *query.URLs) ([]model.URL, error) {
	return m.store.ListURLs(ctx, db, q)
}

// ListLogs lists logs.
func (m *Server) ListLogs(ctx context.Context, db scan.Queryer, urlID int64, q *query.Logs) ([]model.Log, error) {
	return m.store.ListLogs(ctx, db, urlID, q)
}
