package processor

import (
	"context"
	"os"
	"path/filepath"

	"github.com/yansal/youtube-ar/api/downloader"
	"github.com/yansal/youtube-ar/api/model"
)

// Processor is a processor implementation.
type Processor struct {
	downloader Downloader
	storage    Storage
	store      Store
}

// Downloader is the downloader interface required by Processor.
type Downloader interface {
	Download(ctx context.Context, url string) <-chan downloader.Event
}

// Storage is the storage interface required by Processor.
type Storage interface {
	Save(ctx context.Context, path string) (string, error)
}

// Store is the store interface required by Processor.
type Store interface {
	AppendLog(ctx context.Context, urlID int64, log *model.Log) error
}

// New returns a new Processor.
func New(downloader Downloader, storage Storage, store Store) *Processor {
	return &Processor{downloader: downloader, storage: storage, store: store}
}

// Process processes an url.
func (p *Processor) Process(ctx context.Context, url *model.URL) (string, error) {
	var (
		path string
		err  error
	)
	stream := p.downloader.Download(ctx, url.URL)
	for event := range stream {
		switch event.Type {
		case downloader.Log:
			if err := p.store.AppendLog(ctx, url.ID, &model.Log{Log: event.Log}); err != nil {
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

	uploaded, err := p.storage.Save(ctx, path)
	if err != nil {
		return "", err
	}
	return filepath.Base(uploaded), nil
}
