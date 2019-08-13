package cmd

import (
	"context"
	"errors"
	"flag"
	"fmt"

	"github.com/yansal/youtube-ar/api/broker"
	"github.com/yansal/youtube-ar/api/broker/redis"
	"github.com/yansal/youtube-ar/api/log"
	"github.com/yansal/youtube-ar/api/manager"
	"github.com/yansal/youtube-ar/api/oembed"
	"github.com/yansal/youtube-ar/api/payload"
	"github.com/yansal/youtube-ar/api/query"
	"github.com/yansal/youtube-ar/api/service"
	"github.com/yansal/youtube-ar/api/store"
	"github.com/yansal/youtube-ar/api/store/sql"
	"github.com/yansal/youtube-ar/api/youtube"
	"github.com/yansal/youtube-ar/api/youtubedl"
)

// Cmd is the cmd functional type.
type Cmd func(ctx context.Context, args []string) error

// CreateURL is the create-url cmd.
func CreateURL(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("create-url", flag.ExitOnError)
	var url string
	fs.StringVar(&url, "url", "", "url to create")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if url == "" {
		return errors.New("url is required")
	}

	log := log.New()
	redis, err := redis.New(log)
	if err != nil {
		return err
	}
	broker := broker.New(redis, log)
	db, err := sql.New(log)
	if err != nil {
		return err
	}
	store := store.New(db)
	m := manager.NewServer(broker, store)

	p := payload.URL{URL: url}
	if err := p.Validate(); err != nil {
		return err
	}

	if _, err := m.CreateURL(ctx, p); err != nil {
		return err
	}
	return nil
}

// CreateURLsFromPlaylist is the create-urls-from-playlist cmd.
func CreateURLsFromPlaylist(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("create-urls-from-playlist", flag.ExitOnError)
	var playlist string
	fs.StringVar(&playlist, "playlist", "", "youtube playlist")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if playlist == "" {
		return errors.New("playlist is required")
	}

	log := log.New()
	redis, err := redis.New(log)
	if err != nil {
		return err
	}
	broker := broker.New(redis, log)
	db, err := sql.New(log)
	if err != nil {
		return err
	}
	store := store.New(db)
	manager := manager.NewServer(broker, store)
	playlistLoader := service.NewPlaylistLoader(manager, store, youtube.New(log))

	return playlistLoader.CreateURLsFromYoutube(ctx, playlist)
}

// GetOembed is the get-oembed cmd.
func GetOembed(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("get-oembed", flag.ExitOnError)
	var url string
	fs.StringVar(&url, "url", "", "url")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if url == "" {
		return errors.New("url is required")
	}

	log := log.New()
	oe := oembed.NewClient(log)

	data, err := oe.Get(ctx, url)
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", data)
	return nil
}

// DownloadURL is the download-url cmd.
func DownloadURL(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("download-url", flag.ExitOnError)
	var url string
	fs.StringVar(&url, "url", "", "url to download")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if url == "" {
		return errors.New("url is required")
	}

	d := youtubedl.New()
	stream := d.Download(ctx, url)
	for event := range stream {
		switch event.Type {
		case youtubedl.Log:
			fmt.Println(event.Log)
		case youtubedl.Failure:
			return event.Err
		case youtubedl.Success:
			fmt.Printf("downloaded url to %s\n", event.Path)
		}
	}
	return nil
}

// ListLogs is the list-logs cmd.
func ListLogs(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("list-logs", flag.ExitOnError)
	var urlID, cursor, limit int64
	fs.Int64Var(&urlID, "url-id", 0, "url-id")
	fs.Int64Var(&cursor, "cursor", 0, "cursor")
	fs.Int64Var(&limit, "limit", 10, "limit")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if urlID == 0 {
		return errors.New("url-id is required")
	}

	log := log.New()
	db, err := sql.New(log)
	if err != nil {
		return err
	}
	store := store.New(db)
	m := manager.NewServer(nil, store)

	logs, err := m.ListLogs(ctx, urlID, &query.Logs{Cursor: cursor})
	if err != nil {
		return err
	}
	for i := range logs {
		fmt.Printf("%+v\n", logs[i])
	}
	return nil
}

// ListURLs is the list-urls cmd.
func ListURLs(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("list-urls", flag.ExitOnError)
	var cursor, limit int64
	fs.Int64Var(&cursor, "cursor", 0, "cursor")
	fs.Int64Var(&limit, "limit", 10, "limit")
	// TODO: add status flag
	if err := fs.Parse(args); err != nil {
		return err
	}

	log := log.New()
	db, err := sql.New(log)
	if err != nil {
		return err
	}
	store := store.New(db)
	m := manager.NewServer(nil, store)

	urls, err := m.ListURLs(ctx, &query.URLs{Cursor: cursor, Limit: limit})
	if err != nil {
		return err
	}
	for i := range urls {
		fmt.Printf("%+v\n", urls[i])
	}
	return nil
}

// RetryNextDownloadURL is the retry-next-download-url cmd.
func RetryNextDownloadURL(ctx context.Context, args []string) error {
	log := log.New()
	redis, err := redis.New(log)
	if err != nil {
		return err
	}
	broker := broker.New(redis, log)
	db, err := sql.New(log)
	if err != nil {
		return err
	}
	store := store.New(db)
	manager := manager.NewServer(broker, store)

	retrier := service.NewRetrier(broker, manager, store)
	return retrier.RetryNextDownloadURL(ctx)
}
