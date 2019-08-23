package downloader

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/yansal/youtube-ar/api/log"
	"github.com/yansal/youtube-ar/api/model"
	"github.com/yansal/youtube-ar/api/tor"
	"github.com/yansal/youtube-ar/api/youtubedl"
)

// Downloader is a downloader implementation.
type Downloader struct {
	tor       Tor
	youtubedl YoutubeDL
	storage   Storage
	store     Store
	log       log.Logger
}

// Tor is the tor interface required by Downloader.
type Tor interface {
	Start(ctx context.Context) <-chan tor.Event
}

// YoutubeDL is the youtubedl interface required by Downloader.
type YoutubeDL interface {
	Download(ctx context.Context, url string, proxyurl string) <-chan youtubedl.Event
}

// Storage is the storage interface required by Downloader.
type Storage interface {
	Save(ctx context.Context, path string, reader io.ReadSeeker) error
}

// Store is the store interface required by Downloader.
type Store interface {
	AppendLog(ctx context.Context, urlID int64, log *model.Log) error
}

// New returns a new Downloader.
func New(tor Tor, youtubedl YoutubeDL, storage Storage, store Store, log log.Logger) *Downloader {
	return &Downloader{tor: tor, youtubedl: youtubedl, storage: storage, store: store, log: log}
}

// DownloadURL downloads an url.
func (p *Downloader) DownloadURL(ctx context.Context, url *model.URL) (string, error) {
	torready := make(chan tor.Event)
	torctx, shutdowntor := context.WithCancel(ctx)
	defer shutdowntor()
	go func() {
		var ready bool
		stream := p.tor.Start(torctx)
		for event := range stream {
			switch event.Type {
			case tor.Log:
				if err := p.store.AppendLog(ctx, url.ID, &model.Log{Log: event.Log}); err != nil {
					p.log.Log(ctx, err.Error())
				}
			case tor.Failure:
				if !ready {
					torready <- event
					ready = true
					continue
				}
				// TODO: log tor failure?
			case tor.Ready:
				if !ready {
					torready <- event
					ready = true
				}
			}
		}
	}()

	var proxyurl string
	switch event := <-torready; event.Type {
	case tor.Failure:
		return "", event.Err
	case tor.Ready:
		proxyurl = event.ProxyURL
	}

	// TODO: fetch and save tor output geoip

	var (
		path string
		err  error
	)
	stream := p.youtubedl.Download(ctx, url.URL, proxyurl)
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
	defer os.RemoveAll(filepath.Dir(path))
	if err != nil {
		return "", err
	}

	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	filename := filepath.Base(path)
	if err := p.storage.Save(ctx, filename, f); err != nil {
		return "", err
	}
	return filename, nil
}
