package service

import (
	"context"
	"encoding/json"

	"github.com/go-redis/redis"
	"github.com/yansal/youtube-ar/event"
	"github.com/yansal/youtube-ar/model"
	"github.com/yansal/youtube-ar/payload"
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

// RetryNext retries the next failed event.
func (s *Retrier) RetryNext(ctx context.Context) error {
	// TODO: use an atomic rpoplpush to ensure we don't lose any failed event?
	b, err := s.broker.PopNextFailed(ctx, "url-created")
	if err == redis.Nil {
		return nil
	} else if err != nil {
		return err
	}

	var e event.URL
	if err := json.Unmarshal([]byte(b), &e); err != nil {
		return err
	}
	last, err := s.store.GetURL(ctx, e.ID)
	if err != nil {
		return err
	}
	if last.Status != "failed" {
		// TODO: log that there is a problem, we should never retry an event with status != "failed"
	}
	if last.Retries.Int64 >= 5 {
		// TODO: log that we won't retry because we reached the maximum number of retries
		return nil
	}

	url := payload.URL{
		URL:     last.URL,
		Retries: last.Retries.Int64 + 1,
	}
	if _, err := s.manager.CreateURL(ctx, url); err != nil {
		return err
	}
	return nil
}
