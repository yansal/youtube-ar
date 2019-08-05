package worker

import (
	"context"

	"github.com/yansal/youtube-ar/broker"
	"golang.org/x/sync/errgroup"
)

// Worker is a worker implementation.
type Worker struct {
	broker   Broker
	handlers map[string]broker.Handler
}

// Broker is the broker interface required by Worker.
type Broker interface {
	Receive(ctx context.Context, queue string, handler broker.Handler) error
}

// New returns a new Worker.
func New(b Broker, h map[string]broker.Handler) *Worker {
	return &Worker{broker: b, handlers: h}
}

// Listen starts a goroutine for each handler.
func (w *Worker) Listen(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)
	for queue, handler := range w.handlers {
		queue := queue
		handler := handler
		g.Go(func() error {
			for {
				if err := w.broker.Receive(ctx, queue, handler); err != nil {
					return err
				}
			}
		})
	}
	return g.Wait()
}
