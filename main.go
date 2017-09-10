package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

func main() {
	mustHaveInPath("tor", "youtube-dl")

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

	s, err := newServer(*pgConnInfo)
	if err != nil {
		log.Fatal(err)
	}
	http.Handle("/", s)

	w, err := newWorker(*pgConnInfo, *s3Bucket)
	if err != nil {
		log.Fatal(err)
	}

	// ctrl+c handler for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	w.ctx = ctx
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		w.mu.Lock()
		cancel()

		// wait for all jobs to finish
		for _, ch := range w.running {
			<-ch
		}

		os.Exit(0)
	}()

	go w.loop()
	log.Fatal(http.ListenAndServe(*httpAddr, nil))
}

func mustHaveInPath(programs ...string) {
	for _, program := range programs {
		if _, err := exec.LookPath(program); err != nil {
			log.Fatalf("couldn't find %q in PATH", program)
		}
	}
}
