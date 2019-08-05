package manager

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"

	"github.com/yansal/youtube-ar/downloader"
	"github.com/yansal/youtube-ar/event"
	"github.com/yansal/youtube-ar/model"
)

// Worker is the manager used for worker features.
type Worker struct {
	downloader Downloader
	storage    Storage
	store      StoreWorker
}

// Downloader is the downloader interface required by Worker.
type Downloader interface {
	Download(context.Context, string) <-chan downloader.Event
}

// Storage is the storage interface required by Worker.
type Storage interface {
	Upload(context.Context, string) (string, error)
}

// StoreWorker is the store interface required by Worker.
type StoreWorker interface {
	LockURL(context.Context, *model.URL) error
	UnlockURL(context.Context, *model.URL) error
	CreateLog(context.Context, int64, *model.Log) error
}

// NewWorker returns a new Worker.
func NewWorker(downloader Downloader, storage Storage, store StoreWorker) *Worker {
	return &Worker{downloader: downloader, storage: storage, store: store}
}

// ProcessURL processes e.
func (m *Worker) ProcessURL(ctx context.Context, e event.URL) error {
	// TODO: process in db transaction?

	url := &model.URL{ID: e.ID, Status: "processing"}
	if err := m.store.LockURL(ctx, url); err != nil {
		return err
	}

	// TODO: defer unlock, instead of having two code paths

	file, err := m.processURL(ctx, url)
	if err != nil {
		url.Error = sql.NullString{Valid: true, String: err.Error()}
		url.Status = "failure"
		if err := m.store.UnlockURL(ctx, url); err != nil {
			// TODO: log err
		}
		return err
	}

	url.File = sql.NullString{Valid: true, String: file}
	url.Status = "success"
	return m.store.UnlockURL(ctx, url)
}

func (m *Worker) processURL(ctx context.Context, url *model.URL) (string, error) {
	var (
		path string
		err  error
	)
	stream := m.downloader.Download(ctx, url.URL)
	for event := range stream {
		switch event.Type {
		case downloader.Log:
			if err := m.store.CreateLog(ctx, url.ID, &model.Log{Log: event.Log}); err != nil {
				// TODO: log err
			}
		case downloader.Failure:
			err = event.Err
		case downloader.Success:
			path = event.Path
		}
	}
	defer os.Remove(path)
	if err != nil {
		return "", err
	}

	uploaded, err := m.storage.Upload(ctx, path)
	if err != nil {
		return "", err
	}
	return filepath.Base(uploaded), nil
}
