//go:generate go run generate_embed.go
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
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	raven "github.com/getsentry/raven-go"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/sync/errgroup"
	youtube "google.golang.org/api/youtube/v3"
)

func init() {
	log.SetFlags(log.Lshortfile)
	raven.SetDSN(os.Getenv("SENTRY_DSN"))
}

type cfg struct {
	httpAddr, s3Bucket string
	oauth2             *oauth2.Config
	s3Uploader         *s3manager.Uploader
	db                 *sqlx.DB
	pqListener         *pq.Listener
}

func loadConfig() cfg {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "dbname=youtube-ar sslmode=disable"
	}
	db := sqlx.MustConnect("postgres", databaseURL)
	pqListener := pq.NewListener(databaseURL, time.Second, time.Minute, func(ev pq.ListenerEventType, err error) {
		if err != nil {
			raven.CaptureError(err, nil)
		}
	})

	httpAddr := "localhost:8080"
	port := os.Getenv("PORT")
	if port != "" {
		httpAddr = ":" + port
	}

	s3Bucket := os.Getenv("S3_BUCKET")
	if s3Bucket == "" {
		log.Fatal("S3_BUCKET env must be set")
	}
	// TODO: check that s3bucket is present and writable, try to create it if not
	s3Uploader := s3manager.NewUploader(session.Must(session.NewSession()))

	oauth2, err := google.ConfigFromJSON([]byte(os.Getenv("GOOGLE_CLIENT_SECRET_JSON")), youtube.YoutubeReadonlyScope)
	if err != nil {
		log.Print(err)
	}

	return cfg{
		db:         db,
		httpAddr:   httpAddr,
		oauth2:     oauth2,
		pqListener: pqListener,
		s3Bucket:   s3Bucket,
		s3Uploader: s3Uploader,
	}
}

func main() {
	cfg := loadConfig()

	flag.Parse()
	if flag.Arg(0) == "youtubelikes" {
		youtubelikes(cfg)
		return
	}

	g, ctx := errgroup.WithContext(context.Background())

	g.Go(func() error {
		s, err := newServer(cfg)
		if err != nil {
			return err
		}
		http.Handle("/", s)

		srv := &http.Server{Addr: cfg.httpAddr}
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
		w, err := newWorker(ctx, cfg)
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
