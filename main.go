//go:generate go run generate_embed.go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
)

func main() {
	log.SetFlags(log.Lshortfile)

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "dbname=youtube-ar sslmode=disable"
	}

	httpAddr := "localhost:8080"
	port := os.Getenv("PORT")
	if port != "" {
		httpAddr = ":" + port
	}

	s3Bucket := os.Getenv("S3_BUCKET")
	if s3Bucket == "" {
		fmt.Fprintln(os.Stderr, "S3_BUCKET env must be set")
		os.Exit(2)
	}

	g, ctx := errgroup.WithContext(context.Background())

	g.Go(func() error {
		s, err := newServer(databaseURL)
		if err != nil {
			return err
		}
		http.Handle("/", s)

		srv := &http.Server{Addr: httpAddr}
		cerr := make(chan error)
		go func() {
			cerr <- srv.ListenAndServe()
		}()

		select {
		case <-ctx.Done():
			return srv.Shutdown(context.Background())
		case err := <-cerr:
			return err
		}
	})

	g.Go(func() error {
		w, err := newWorker(ctx, databaseURL, s3Bucket)
		if err != nil {
			return err
		}
		go w.loop()

		<-ctx.Done()
		return w.shutdown()
	})

	g.Go(func() error {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		select {
		case <-ctx.Done():
			return nil
		case s := <-sig:
			return fmt.Errorf("Got signal %v", s)
		}
	})

	log.Print(g.Wait())
}

type Job struct {
	ID           int        `json:"id"`
	URL          string     `json:"url"`
	Retries      int        `json:"retries"`
	StartedAt    time.Time  `db:"started_at"`
	DownloadedAt *time.Time `db:"downloaded_at"`
	UploadedAt   *time.Time `db:"uploaded_at"`
	Output       *string
	Error        *string
	TorLog       *string
	IP           *string
	Country      *string
	Feed         *string
}
