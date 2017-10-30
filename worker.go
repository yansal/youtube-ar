package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	raven "github.com/getsentry/raven-go"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	torpkg "github.com/yansal/tor"
)

type worker struct {
	l          *pq.Listener
	db         *sqlx.DB
	s3uploader *s3manager.Uploader
	s3bucket   string

	ctx     context.Context
	mu      sync.Mutex
	running map[int]chan struct{}
}

func newWorker(ctx context.Context, cfg cfg) (*worker, error) {
	if err := isInPath("tor", "youtube-dl"); err != nil {
		return nil, err
	}

	if err := cfg.pqListener.Listen("jobs"); err != nil {
		return nil, err
	}

	return &worker{
		l:          cfg.pqListener,
		db:         cfg.db,
		s3uploader: cfg.s3Uploader,
		s3bucket:   cfg.s3Bucket,
		running:    make(map[int]chan struct{}),
		ctx:        ctx,
	}, nil
}

func isInPath(programs ...string) error {
	for _, program := range programs {
		if _, err := exec.LookPath(program); err != nil {
			return fmt.Errorf("couldn't find %q in PATH", program)
		}
	}
	return nil
}

func (w *worker) loop() {
	for range w.l.Notify {
		go w.getWork()
	}
}

func (w *worker) getWork() {
	for {
		tx, err := w.db.Beginx()
		if err != nil {
			raven.CaptureError(err, nil)
			return
		}
		var job Job
		if err := tx.Get(&job, queries["select_running_for_update.sql"]); err == sql.ErrNoRows {
			rollback(tx)
			return
		} else if err != nil {
			raven.CaptureError(err, nil)
			rollback(tx)
			continue
		}
		w.doWork(tx, job)
	}
}

func rollback(tx *sqlx.Tx) {
	if dberr := tx.Rollback(); dberr != nil {
		raven.CaptureError(dberr, nil)
	}
}

func (w *worker) doWork(tx *sqlx.Tx, job Job) {
	done := make(chan struct{})
	w.mu.Lock()
	w.running[job.ID] = done
	w.mu.Unlock()
	defer func() {
		w.mu.Lock()
		delete(w.running, job.ID)
		w.mu.Unlock()
	}()
	defer close(done)

	defer func() {
		defer tx.Rollback()
		if err := tx.Commit(); err != nil {
			raven.CaptureError(err, nil)
		}
	}()

	tmpdir, err := ioutil.TempDir("", "youtube-ar")
	if err != nil {
		if _, dberr := tx.Exec(queries["update_error.sql"], err.Error(), job.ID); dberr != nil {
			raven.CaptureError(dberr, nil)
		}
		return
	}
	defer os.RemoveAll(tmpdir)

	var tor *torpkg.Tor
	if job.Retries > 0 {
		ctx, cancel := context.WithTimeout(w.ctx, 10*time.Minute)
		defer cancel()
		tor, err = torpkg.New(ctx)
		if err != nil {
			if _, dberr := tx.Exec(queries["update_error.sql"], err.Error(), job.ID); dberr != nil {
				raven.CaptureError(dberr, nil)
			}
			return
		}
		defer tor.Shutdown()

		geoip, err := json.Marshal(tor.GeoIP)
		if err != nil {
			raven.CaptureError(err, nil)
			return
		}
		if _, dberr := tx.Exec(queries["update_geoip.sql"], geoip, job.ID); dberr != nil {
			raven.CaptureError(dberr, nil)
			return
		}
	}

	output, err := youtubeDL(w.ctx, job.URL, tmpdir, tor)
	if err != nil {
		if tor != nil {
			if _, dberr := tx.Exec(queries["update_output_error_torlog.sql"], output, err.Error(), tor.Log(), job.ID); dberr != nil {
				raven.CaptureError(dberr, nil)
				return
			}
		} else {
			if _, dberr := tx.Exec(queries["update_output_error.sql"], output, err.Error(), job.ID); dberr != nil {
				raven.CaptureError(dberr, nil)
				return
			}
		}
		if _, ok := err.(geoError); !ok {
			return
		}

		// retry
		var id int
		job.Retries++
		if dberr := w.db.Get(&id, queries["insert_retries.sql"], job.URL, job.Retries); dberr != nil {
			raven.CaptureError(dberr, nil)
		}
		return
	}

	if _, dberr := tx.Exec(queries["update_output.sql"], output, time.Now(), job.ID); dberr != nil {
		raven.CaptureError(dberr, nil)
		return
	}

	if err := w.uploadAllToS3(tmpdir); err != nil {
		if _, dberr := tx.Exec(queries["update_error.sql"], err.Error(), job.ID); dberr != nil {
			raven.CaptureError(dberr, nil)
		}
		return
	}
	if _, dberr := tx.Exec(queries["update_uploaded_at.sql"], time.Now(), job.ID); dberr != nil {
		raven.CaptureError(dberr, nil)
	}
}

func (w *worker) shutdown() error {
	if err := w.l.UnlistenAll(); err != nil {
		raven.CaptureError(err, nil)
	}

	w.mu.Lock()
	for _, ch := range w.running {
		<-ch
	}
	return nil
}

func (w *worker) uploadAllToS3(dir string) error {
	fis, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, fi := range fis {
		if err := w.uploadToS3(filepath.Join(dir, fi.Name())); err != nil {
			return err
		}
	}
	return nil
}

func (w *worker) uploadToS3(fname string) error {
	input := &s3manager.UploadInput{
		Bucket: aws.String(w.s3bucket),
		Key:    aws.String(filepath.Base(fname)),
	}

	switch {
	case strings.HasSuffix(fname, ".mp3"):
		input.ContentType = aws.String("audio/mpeg")
	case strings.HasSuffix(fname, ".mp4"):
		input.ContentType = aws.String("video/mp4")
	case strings.HasSuffix(fname, ".webm"):
		input.ContentType = aws.String("video/webm")
	}

	f, err := os.Open(fname)
	if err != nil {
		return err
	}
	defer f.Close()
	input.Body = f

	_, err = w.s3uploader.UploadWithContext(w.ctx, input)
	return err
}

func youtubeDL(ctx context.Context, url string, tmpdir string, tor *torpkg.Tor) (string, error) {
	args := []string{"-v"}
	if tor != nil {
		args = append(args, "--proxy", "socks5://"+tor.ListenAddr)
	}
	args = append(args, url)
	cmd := exec.CommandContext(ctx, "youtube-dl", args...)
	cmd.Dir = tmpdir
	bytes, err := cmd.CombinedOutput()
	output := strings.Replace(string(bytes), "\r", "\r\n", -1)
	if err != nil {
		if looksLikeGeoError(output) {
			err = geoError{error: err}
		}
	}
	return output, err
}

type geoError struct{ error }

func (e geoError) Error() string {
	return e.error.Error()
}

var geoErrorRegexs = []*regexp.Regexp{
	regexp.MustCompile(`ERROR: The uploader has not made this video available in your country\.`),
	regexp.MustCompile(`ERROR: .*: YouTube said: This video contains content from .*, who has blocked it on copyright grounds\.`),
}

func looksLikeGeoError(output string) bool {
	for _, regex := range geoErrorRegexs {
		if regex.MatchString(output) {
			return true
		}
	}
	return false
}
