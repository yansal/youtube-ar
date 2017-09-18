package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sync/errgroup"
)

func main() {
	var (
		httpAddr   = flag.String("addr", "localhost:8080", "HTTP listening address")
		pgConnInfo = flag.String("conninfo", "dbname=youtube-ar sslmode=disable", "PostgreSQL connection string")
		s3Bucket   = flag.String("bucket", "", "S3 bucket")
	)
	flag.Parse()

	if *s3Bucket == "" {
		fmt.Fprintln(os.Stderr, "bucket flag must be set")
		flag.Usage()
		os.Exit(2)
	}

	g, ctx := errgroup.WithContext(context.Background())

	g.Go(func() error {
		s, err := newServer(*pgConnInfo)
		if err != nil {
			return err
		}
		http.Handle("/", s)

		srv := &http.Server{Addr: *httpAddr}
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
		w, err := newWorker(ctx, *pgConnInfo, *s3Bucket)
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
