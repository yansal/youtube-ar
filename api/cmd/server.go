package cmd

import (
	"context"
	"net/http"
	"os"
	"regexp"

	"github.com/yansal/youtube-ar/api/broker"
	"github.com/yansal/youtube-ar/api/broker/redis"
	"github.com/yansal/youtube-ar/api/log"
	"github.com/yansal/youtube-ar/api/manager"
	"github.com/yansal/youtube-ar/api/server"
	"github.com/yansal/youtube-ar/api/server/handler"
	"github.com/yansal/youtube-ar/api/server/middleware"
	"github.com/yansal/youtube-ar/api/store"
	"github.com/yansal/youtube-ar/api/store/db"
)

// Server is the server cmd.
func Server(ctx context.Context, args []string) error {
	log := log.New()
	redis, err := redis.New(log)
	if err != nil {
		return err
	}
	broker := broker.New(redis, log)
	db, err := db.New(log)
	if err != nil {
		return err
	}
	store := store.New(db)
	manager := manager.NewServer(broker, store)

	mux := server.NewMux()
	mux.HandleFunc(http.MethodGet, regexp.MustCompile(`^/urls$`), handler.ListURLs(manager))
	mux.HandleFunc(http.MethodGet, regexp.MustCompile(`^/urls/(\d+)$`), handler.DetailURL(manager))
	mux.HandleFunc(http.MethodPost, regexp.MustCompile(`^/urls$`), handler.CreateURL(manager))
	mux.HandleFunc(http.MethodGet, regexp.MustCompile(`^/urls/(\d+)/logs$`), handler.ListLogs(manager))

	handler := middleware.Log(mux, log)
	handler = middleware.CORS(handler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return http.ListenAndServe(":"+port, handler)
}
