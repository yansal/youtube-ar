package manager

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/yansal/sql/scan"
	"github.com/yansal/youtube-ar/api/event"
	"github.com/yansal/youtube-ar/api/model"
)

// Worker is the manager used for worker features.
type Worker struct {
	downloader Downloader
	oembed     OEmbed
	store      StoreWorker
}

// Downloader is the downloader interface required by Worker.
type Downloader interface {
	DownloadURL(context.Context, scan.Queryer, *model.URL) (string, error)
}

// OEmbed is the oembed interface required by Worker.
type OEmbed interface {
	Get(context.Context, string) ([]byte, error)
}

// StoreWorker is the store interface required by Worker.
type StoreWorker interface {
	LockURL(context.Context, scan.Queryer, *model.URL) error
	UnlockURL(context.Context, scan.Queryer, *model.URL) error
	SetOEmbed(context.Context, scan.Queryer, *model.URL) error
}

// NewWorker returns a new Worker.
func NewWorker(downloader Downloader, oembed OEmbed, store StoreWorker) *Worker {
	return &Worker{downloader: downloader, oembed: oembed, store: store}
}

// DownloadURL downloads e.
func (m *Worker) DownloadURL(ctx context.Context, db scan.Queryer, e event.URL) error {
	url := &model.URL{ID: e.ID, URL: e.URL, Status: "processing"}
	if err := m.store.LockURL(ctx, db, url); err != nil {
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

		if ctx.Err() != nil {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(context.Background(), time.Second)
			defer cancel()
		}
		if err := m.store.UnlockURL(ctx, db, url); err != nil {
			// TODO: log err
		}

		if r != nil {
			panic(r)
		}
	}()

	file, perr = m.downloader.DownloadURL(ctx, db, url)
	return perr
}

// GetOEmbed gets oembed.
func (m *Worker) GetOEmbed(ctx context.Context, db scan.Queryer, e event.URL) error {
	data, err := m.oembed.Get(ctx, e.URL)
	if err != nil {
		return err
	}
	url := &model.URL{ID: e.ID, OEmbed: data}
	return m.store.SetOEmbed(ctx, db, url)
}
