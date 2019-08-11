package service

import (
	"context"
	"encoding/json"

	"github.com/go-redis/redis"
	"github.com/yansal/youtube-ar/api/event"
	"github.com/yansal/youtube-ar/api/model"
	"github.com/yansal/youtube-ar/api/payload"
)

// Retrier is a retrier.
type Retrier struct {
	broker  RetrierBroker
	manager RetrierManager
	store   RetrierStore
}

// RetrierBroker is the broker interface required by Retrier.
type RetrierBroker interface {
	PopNextFailed(context.Context, string) (string, error)
	RemFailed(context.Context, string, string) error
}

// RetrierManager is the manager interface required by Retrier.
type RetrierManager interface {
	CreateURL(context.Context, payload.URL) (*model.URL, error)
}

// RetrierStore is the store interface required by Retrier.
type RetrierStore interface {
	GetURL(context.Context, int64) (*model.URL, error)
}

// NewRetrier returns a new Retrier.
func NewRetrier(broker RetrierBroker, manager RetrierManager, store RetrierStore) *Retrier {
	return &Retrier{broker: broker, manager: manager, store: store}
}

// RetryNext retries the next failed url.
func (r *Retrier) RetryNext(ctx context.Context) error {
	// TODO: use an atomic rpoplpush to ensure we don't lose any failed event?
	b, err := r.broker.PopNextFailed(ctx, "url-created")
	if err == redis.Nil {
		return nil
	} else if err != nil {
		return err
	}

	var e event.URL
	if err := json.Unmarshal([]byte(b), &e); err != nil {
		return err
	}
	failed, err := r.store.GetURL(ctx, e.ID)
	if err != nil {
		return err
	}

	if !failed.ShouldRetry() {
		return nil
	}

	if failed.Retries.Int64 >= 5 {
		return nil
	}

	_, err = r.retry(ctx, failed)
	return err
}

// RetryURL retries the url with the given id.
func (r *Retrier) RetryURL(ctx context.Context, id int64) (*model.URL, error) {
	e := &event.URL{ID: id}
	b, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	// TODO: use an atomic rpoplpush to ensure we don't lose any failed event?
	if err := r.broker.RemFailed(ctx, "url-created", string(b)); err != nil && err != redis.Nil {
		return nil, err
	}

	failed, err := r.store.GetURL(ctx, e.ID)
	if err != nil {
		return nil, err
	}

	return r.retry(ctx, failed)
}

func (r *Retrier) retry(ctx context.Context, failed *model.URL) (*model.URL, error) {
	url := payload.URL{
		URL:     failed.URL,
		Retries: failed.Retries.Int64 + 1,
	}
	return r.manager.CreateURL(ctx, url)
}
