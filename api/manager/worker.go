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
	downloader Downloader
	store      StoreWorker
}

// Downloader is the downloader interface required by Worker.
type Downloader interface {
	DownloadURL(context.Context, *model.URL) (string, error)
}

// StoreWorker is the store interface required by Worker.
type StoreWorker interface {
	LockURL(context.Context, *model.URL) error
	UnlockURL(context.Context, *model.URL) error
}

// NewWorker returns a new Worker.
func NewWorker(downloader Downloader, store StoreWorker) *Worker {
	return &Worker{downloader: downloader, store: store}
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

	file, perr = m.downloader.DownloadURL(ctx, url)
	return perr
}
