package downloader

import (
	"context"
	"os"
	"path/filepath"

	"github.com/yansal/youtube-ar/api/log"
	"github.com/yansal/youtube-ar/api/model"
	"github.com/yansal/youtube-ar/api/youtubedl"
)

// Downloader is a downloader implementation.
type Downloader struct {
	youtubedl YoutubeDL
	storage   Storage
	store     Store
	log       log.Logger
}

// YoutubeDL is the youtubedl interface required by Downloader.
type YoutubeDL interface {
	Download(ctx context.Context, url string) <-chan youtubedl.Event
}

// Storage is the storage interface required by Downloader.
type Storage interface {
	Save(ctx context.Context, path string) (string, error)
}

// Store is the store interface required by Downloader.
type Store interface {
	AppendLog(ctx context.Context, urlID int64, log *model.Log) error
}

// New returns a new Downloader.
func New(youtubedl YoutubeDL, storage Storage, store Store, log log.Logger) *Downloader {
	return &Downloader{youtubedl: youtubedl, storage: storage, store: store, log: log}
}

// DownloadURL downloads an url.
func (p *Downloader) DownloadURL(ctx context.Context, url *model.URL) (string, error) {
	var (
		path string
		err  error
	)
	stream := p.youtubedl.Download(ctx, url.URL)
	for event := range stream {
		switch event.Type {
		case youtubedl.Log:
			if err := p.store.AppendLog(ctx, url.ID, &model.Log{Log: event.Log}); err != nil {
				p.log.Log(ctx, err.Error())
			}
		case youtubedl.Failure:
			err = event.Err
		case youtubedl.Success:
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
