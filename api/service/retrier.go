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
	failed, err := s.store.GetURL(ctx, e.ID)
	if err != nil {
		return err
	}

	return s.retry(ctx, failed)
}

func (s *Retrier) retry(ctx context.Context, failed *model.URL) error {
	if !failed.ShouldRetry() {
		return nil
	}

	if failed.Retries.Int64 >= 5 {
		return nil
	}

	url := payload.URL{
		URL:     failed.URL,
		Retries: failed.Retries.Int64 + 1,
	}
	if _, err := s.manager.CreateURL(ctx, url); err != nil {
		return err
	}
	return nil
}
