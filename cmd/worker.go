package cmd

import (
	"context"

	"github.com/yansal/youtube-ar/broker"
	"github.com/yansal/youtube-ar/broker/redis"
	"github.com/yansal/youtube-ar/downloader"
	"github.com/yansal/youtube-ar/log"
	"github.com/yansal/youtube-ar/manager"
	"github.com/yansal/youtube-ar/processor"
	"github.com/yansal/youtube-ar/storage"
	"github.com/yansal/youtube-ar/store"
	"github.com/yansal/youtube-ar/store/db"
	"github.com/yansal/youtube-ar/worker"
	"github.com/yansal/youtube-ar/worker/handler"
)

// Worker is the worker cmd.
func Worker(ctx context.Context, args []string) error {
	log := log.New()
	redis, err := redis.New(log)
	if err != nil {
		return err
	}
	b := broker.New(redis, log)

	storage, err := storage.New()
	if err != nil {
		return err
	}
	db, err := db.New(log)
	if err != nil {
		return err
	}
	store := store.New(db)
	processor := processor.New(downloader.New(), storage, store)
	m := manager.NewWorker(processor, store)

	w := worker.New(b, map[string]broker.Handler{
		"url-created": handler.URLCreated(m),
	})
	return w.Listen(ctx)
}
