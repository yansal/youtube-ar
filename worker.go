package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	torpkg "github.com/yansal/tor"
)

type worker struct {
	l          *pq.Listener
	db         *sqlx.DB
	queries    queryMap
	s3uploader *s3manager.Uploader
	s3bucket   string

	ctx     context.Context
	mu      sync.Mutex
	running map[int]chan struct{}
}

func newWorker(pgConnInfo, s3bucket string) (*worker, error) {
	db := sqlx.MustConnect("postgres", pgConnInfo)

	queries, err := loadQueries()
	if err != nil {
		return nil, err
	}

	// TODO: check that s3bucket is present and writable, try to create it if not
	s3uploader := s3manager.NewUploader(session.Must(session.NewSession()))

	l := pq.NewListener(pgConnInfo, time.Second, time.Minute, func(ev pq.ListenerEventType, err error) {
		if err != nil {
			log.Println(ev, err)
		}
	})

	if err := l.Listen("job"); err != nil {
		return nil, err
	}

	return &worker{
		l:          l,
		db:         db,
		queries:    queries,
		s3uploader: s3uploader,
		s3bucket:   s3bucket,
		running:    make(map[int]chan struct{}),
	}, nil
}

func (w *worker) loop() {
	for {
		payload := <-w.l.Notify
		var job Job
		if err := json.Unmarshal([]byte(payload.Extra), &job); err != nil {
			log.Print(err)
		}
		go w.run(job)
	}
}

func (w *worker) run(job Job) {
	done := make(chan struct{})
	w.mu.Lock()
	w.running[job.ID] = done
	w.mu.Unlock()
	defer close(done)

	tmpdir, err := ioutil.TempDir("", "youtube-ar")
	if err != nil {
		if _, dberr := w.db.Exec(w.queries["update_error.sql"], err.Error(), job.ID); dberr != nil {
			log.Print(dberr)
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
			if _, dberr := w.db.Exec(w.queries["update_error.sql"], err.Error(), job.ID); dberr != nil {
				log.Print(dberr)
			}
			return
		}
		defer tor.Shutdown()

		geoip, err := json.Marshal(tor.GeoIP)
		if err != nil {
			log.Print(err)
			return
		}
		if _, dberr := w.db.Exec(w.queries["update_geoip.sql"], geoip, job.ID); dberr != nil {
			log.Print(dberr)
			return
		}
	}

	output, err := youtubeDL(w.ctx, job.URL, tmpdir, tor)
	if err != nil {
		if tor != nil {
			if _, dberr := w.db.Exec(w.queries["update_output_error_torlog.sql"], output, err.Error(), tor.Log(), job.ID); dberr != nil {
				log.Print(dberr)
				return
			}
		} else {
			if _, dberr := w.db.Exec(w.queries["update_output_error.sql"], output, err.Error(), job.ID); dberr != nil {
				log.Print(dberr)
				return
			}
		}
		if _, ok := err.(geoError); !ok {
			return
		}

		// retry
		var id int
		job.Retries++
		if dberr := w.db.Get(&id, w.queries["insert_retries.sql"], job.URL, job.Retries); dberr != nil {
			log.Print(dberr)
		}
		return
	}

	if _, dberr := w.db.Exec(w.queries["update_output.sql"], output, time.Now(), job.ID); dberr != nil {
		log.Print(dberr)
		return
	}

	if err := w.uploadAllToS3(tmpdir); err != nil {
		if _, dberr := w.db.Exec(w.queries["update_error.sql"], err.Error(), job.ID); dberr != nil {
			log.Print(dberr)
		}
		return
	}
	if _, dberr := w.db.Exec(w.queries["update_uploaded_at.sql"], time.Now(), job.ID); dberr != nil {
		log.Print(dberr)
	}
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
	f, err := os.Open(fname)
	if err != nil {
		return err
	}

	defer f.Close()

	_, err = w.s3uploader.UploadWithContext(w.ctx, &s3manager.UploadInput{
		Bucket: aws.String(w.s3bucket),
		Key:    aws.String(filepath.Base(fname)),
		Body:   f,
	})
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
