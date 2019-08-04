package worker

import (
	"context"

	"github.com/yansal/youtube-ar/broker"
	"golang.org/x/sync/errgroup"
)

// Worker is a worker implementation.
type Worker struct {
	broker   broker.Broker
	handlers map[string]broker.Handler
}

// New returns a new worker.
func New(b broker.Broker, h map[string]broker.Handler) *Worker {
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
