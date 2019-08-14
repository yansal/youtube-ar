package cmd

import (
	"context"
	"net/http"
	"os"

	"github.com/yansal/youtube-ar/api/broker"
	"github.com/yansal/youtube-ar/api/downloader"
	"github.com/yansal/youtube-ar/api/log"
	loghttp "github.com/yansal/youtube-ar/api/log/http"
	"github.com/yansal/youtube-ar/api/manager"
	"github.com/yansal/youtube-ar/api/oembed"
	"github.com/yansal/youtube-ar/api/storage"
	"github.com/yansal/youtube-ar/api/store"
	"github.com/yansal/youtube-ar/api/worker"
	"github.com/yansal/youtube-ar/api/worker/handler"
	"github.com/yansal/youtube-ar/api/youtubedl"
)

// Worker is the worker cmd.
func Worker(ctx context.Context, args []string) error {
	log := log.New()
	redis, err := newRedis(log)
	if err != nil {
		return err
	}
	b := broker.New(redis, log)

	storage, err := storage.New(os.Getenv("S3_BUCKET"))
	if err != nil {
		return err
	}
	db, err := newSQLDB(log)
	if err != nil {
		return err
	}
	store := store.New(db)
	downloader := downloader.New(youtubedl.New(), storage, store, log)
	httpclient := loghttp.Wrap(new(http.Client), log)
	m := manager.NewWorker(downloader, oembed.NewClient(httpclient), store)

	w := worker.New(b, map[string]broker.Handler{
		"download-url": handler.DownloadURL(m),
		"get-oembed":   handler.GetOEmbed(m),
	})
	return w.Listen(ctx)
}
