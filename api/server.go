package main

import (
	"context"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"regexp"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/yansal/youtube-ar/api/broker"
	"github.com/yansal/youtube-ar/api/log"
	"github.com/yansal/youtube-ar/api/manager"
	"github.com/yansal/youtube-ar/api/resource"
	"github.com/yansal/youtube-ar/api/server"
	"github.com/yansal/youtube-ar/api/server/handler"
	"github.com/yansal/youtube-ar/api/server/middleware"
	"github.com/yansal/youtube-ar/api/service"
	"github.com/yansal/youtube-ar/api/store"
)

func runServer(ctx context.Context, args []string) error {
	log := log.New()
	redis, err := newRedis(log)
	if err != nil {
		return err
	}
	broker := broker.New(redis, log)
	db, err := newSQLDB(log)
	if err != nil {
		return err
	}
	store := store.New()
	manager := manager.NewServer(broker, store)

	serializer := resource.NewSerializer(
		"https://" + os.Getenv("S3_BUCKET") + ".s3." + os.Getenv("AWS_REGION") + ".amazonaws.com/",
	)

	mux := server.NewMux()
	mux.HandleFunc(http.MethodGet, regexp.MustCompile(`^/urls$`), handler.ListURLs(manager, db, serializer))
	mux.HandleFunc(http.MethodPost, regexp.MustCompile(`^/urls$`), handler.CreateURL(manager, db, serializer))
	mux.HandleFunc(http.MethodGet, regexp.MustCompile(`^/urls/(\d+)$`), handler.DetailURL(manager, db, serializer))

	mux.HandleFunc(http.MethodDelete, regexp.MustCompile(`^/urls/(\d+)$`), handler.DeleteURL(manager, db))
	mux.HandleFunc(http.MethodOptions, regexp.MustCompile(`^/urls/(\d+)$`), func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Methods", http.MethodDelete)
	})

	mux.HandleFunc(http.MethodGet, regexp.MustCompile(`^/urls/(\d+)/logs$`), handler.ListLogs(manager, db, serializer))

	retrier := service.NewRetrier(broker, manager, store)
	mux.HandleFunc(http.MethodPost, regexp.MustCompile(`^/urls/(\d+)/retry$`), handler.RetryDownloadURL(retrier, db, serializer))

	handler := middleware.Log(mux, log)
	handler = middleware.CORS(handler)
	server := http.Server{Handler: handler}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	l, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}

	cerr := make(chan error)
	go func() {
		cerr <- server.Serve(l)
	}()

	select {
	case <-ctx.Done():
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Log(ctx, err.Error())
		}
		return nil
	case err := <-cerr:
		return err
	}
}

func runPprofServer(ctx context.Context, port string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	server := http.Server{Handler: mux}

	l, err := net.Listen("tcp", "localhost:"+port)
	if err != nil {
		return err
	}

	cerr := make(chan error)
	go func() {
		cerr <- server.Serve(l)
	}()

	select {
	case <-ctx.Done():
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.New().Log(ctx, err.Error())
		}
		return nil
	case err := <-cerr:
		return err
	}
}

func runPrometheusServer(ctx context.Context, port string) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	server := http.Server{Handler: mux}

	l, err := net.Listen("tcp", "localhost:"+port)
	if err != nil {
		return err
	}

	cerr := make(chan error)
	go func() {
		cerr <- server.Serve(l)
	}()

	select {
	case <-ctx.Done():
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.New().Log(ctx, err.Error())
		}
		return nil
	case err := <-cerr:
		return err
	}
}
