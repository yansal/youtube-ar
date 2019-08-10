package manager

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/yansal/youtube-ar/api/event"
	"github.com/yansal/youtube-ar/api/model"
)

// Worker is the manager used for worker features.
type Worker struct {
	processor Processor
	store     StoreWorker
}

// Processor is the processor interface required by Worker.
type Processor interface {
	Process(context.Context, *model.URL) (string, error)
}

// StoreWorker is the store interface required by Worker.
type StoreWorker interface {
	LockURL(context.Context, *model.URL) error
	UnlockURL(context.Context, *model.URL) error
}

// NewWorker returns a new Worker.
func NewWorker(processor Processor, store StoreWorker) *Worker {
	return &Worker{processor: processor, store: store}
}

// ProcessURL processes e.
func (m *Worker) ProcessURL(ctx context.Context, e event.URL) error {
	url := &model.URL{ID: e.ID, Status: "processing"}
	if err := m.store.LockURL(ctx, url); err != nil {
		return err
	}

	var (
		perr error
		file string
	)
	defer func() {
		r := recover()
		if r != nil {
			perr = fmt.Errorf("%s", r)
		}
		if perr != nil {
			url.Error = sql.NullString{Valid: true, String: perr.Error()}
			url.Status = "failure"
		} else {
			url.File = sql.NullString{Valid: true, String: file}
			url.Status = "success"
		}

		if err := m.store.UnlockURL(ctx, url); err != nil {
			// TODO: log err
		}

		if r != nil {
			panic(r)
		}
	}()

	file, perr = m.processor.Process(ctx, url)
	return perr
}
