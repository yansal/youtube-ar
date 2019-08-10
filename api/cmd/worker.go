package cmd

import (
	"context"

	"github.com/yansal/youtube-ar/api/broker"
	"github.com/yansal/youtube-ar/api/broker/redis"
	"github.com/yansal/youtube-ar/api/downloader"
	"github.com/yansal/youtube-ar/api/log"
	"github.com/yansal/youtube-ar/api/manager"
	"github.com/yansal/youtube-ar/api/processor"
	"github.com/yansal/youtube-ar/api/storage"
	"github.com/yansal/youtube-ar/api/store"
	"github.com/yansal/youtube-ar/api/store/db"
	"github.com/yansal/youtube-ar/api/worker"
	"github.com/yansal/youtube-ar/api/worker/handler"
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