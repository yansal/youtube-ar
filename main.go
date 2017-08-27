package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"go/build"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	torpkg "github.com/yansal/tor"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
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

	s, err := newServer(*pgConnInfo, *s3Bucket)
	if err != nil {
		log.Fatal(err)
	}
	http.Handle("/", s)

	// ctrl+c handler for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	s.ctx = ctx
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		s.mu.Lock()
		cancel()

		// wait for all jobs to finish
		for _, ch := range s.jobs {
			<-ch
		}

		os.Exit(0)
	}()

	log.Fatal(http.ListenAndServe(*httpAddr, nil))
}

func mustHaveInPath(programs ...string) {
	for _, program := range programs {
		if _, err := exec.LookPath(program); err != nil {
			log.Fatalf("couldn't find %q in PATH", program)
		}
	}
}

type server struct {
	db         *sqlx.DB
	queries    queryMap
	tmpl       *template.Template
	s3uploader *s3manager.Uploader
	s3bucket   string

	ctx  context.Context
	mu   sync.Mutex
	jobs map[int]chan struct{}
}

func newServer(pgConnInfo, s3bucket string) (*server, error) {
	db := sqlx.MustConnect("postgres", pgConnInfo)

	queries, err := loadQueries()
	if err != nil {
		return nil, err
	}

	tmpl, err := loadTemplates()
	if err != nil {
		return nil, err
	}

	// TODO: check that s3bucket is present and writable, try to create it if not
	s3uploader := s3manager.NewUploader(session.Must(session.NewSession()))

	return &server{
		db:         db,
		queries:    queries,
		tmpl:       tmpl,
		s3uploader: s3uploader,
		s3bucket:   s3bucket,
		jobs:       make(map[int]chan struct{}),
	}, nil
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path
	switch {
	case p == "/":
		s.runningHandler(w, r)
	case p == "/done/":
		s.doneHandler(w, r)
	case p == "/errors/":
		s.errorsHandler(w, r)
	case strings.HasPrefix(p, "/detail/"):
		s.detailHandler(w, r)
	default:
		http.NotFound(w, r)
	}
}

type queryMap map[string]string

func loadQueries() (queryMap, error) {
	pkg, err := build.Default.Import("github.com/yansal/youtube-ar/queries", "", build.FindOnly)
	if err != nil {
		return nil, fmt.Errorf("could not find queries directory: %v", err)
	}

	fnames, err := filepath.Glob(filepath.Join(pkg.Dir, "*.sql"))
	if err != nil {
		log.Fatal(err)
	}

	queries := queryMap{}
	for _, fname := range fnames {
		b, err := ioutil.ReadFile(fname)
		if err != nil {
			log.Fatal(err)
		}
		queries[filepath.Base(fname)] = string(b)
	}
	return queries, nil
}

func loadTemplates() (*template.Template, error) {
	pkg, err := build.Default.Import("github.com/yansal/youtube-ar/templates", "", build.FindOnly)
	if err != nil {
		return nil, fmt.Errorf("could not find queries directory: %v", err)
	}

	return template.New("").Funcs(template.FuncMap{
		"ago": func(t time.Time) string {
			return fmt.Sprintf("%s (%s)",
				ago(t),
				t.Format("2 Jan 2006 15:04:05 MST"))
		},
	}).ParseGlob(filepath.Join(pkg.Dir, "*.html"))
}

func ago(t time.Time) string {
	ago := time.Since(t)

	seconds := int(ago.Seconds())
	minutes := int(ago.Minutes())
	hours := int(ago.Hours())
	days := hours / 24
	weeks := days / 7
	years := days / 365

	if years > 1 {
		return fmt.Sprintf("%d years ago", years)
	} else if years == 1 {
		return "1 year ago"
	} else if weeks > 1 {
		return fmt.Sprintf("%d weeks ago", weeks)
	} else if weeks == 1 {
		return "1 week ago"
	} else if days > 1 {
		return fmt.Sprintf("%d days ago", days)
	} else if days == 1 {
		return "1 day ago"
	} else if hours > 1 {
		return fmt.Sprintf("%d hours ago", hours)
	} else if hours == 1 {
		return "1 hour ago"
	} else if minutes > 1 {
		return fmt.Sprintf("%d minutes ago", minutes)
	} else if minutes == 1 {
		return "1 minute ago"
	} else if seconds > 1 {
		return fmt.Sprintf("%d seconds ago", seconds)
	} else if seconds == 1 {
		return "1 second ago"
	} else {
		return "just now"
	}
}

type jobResource struct {
	ID           int
	URL          string
	StartedAt    time.Time  `db:"started_at"`
	DownloadedAt *time.Time `db:"downloaded_at"`
	UploadedAt   *time.Time `db:"uploaded_at"`
	Output       *string
	Error        *string
	TorLog       *string
	Retries      int
	IP           *string
	Country      *string
}

func (s *server) runningHandler(w http.ResponseWriter, r *http.Request) {
	if url := r.FormValue("url"); url != "" {
		var id int
		if err := s.db.Get(&id, s.queries["insert.sql"], url); err != nil {
			log.Print(err)
		} else {
			go s.run(Job{id: id, url: url})
		}
		http.Redirect(w, r, "/", http.StatusFound)
	}

	var jobs []jobResource
	if err := s.db.Select(&jobs, s.queries["select_running.sql"]); err != nil {
		log.Print(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.tmpl.ExecuteTemplate(w, "running.html", jobs); err != nil {
		log.Print(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *server) doneHandler(w http.ResponseWriter, r *http.Request) {
	var jobs []jobResource
	if err := s.db.Select(&jobs, s.queries["select_done.sql"]); err != nil {
		log.Print(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.tmpl.ExecuteTemplate(w, "done.html", jobs); err != nil {
		log.Print(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *server) errorsHandler(w http.ResponseWriter, r *http.Request) {
	var jobs []jobResource
	if err := s.db.Select(&jobs, s.queries["select_error.sql"]); err != nil {
		log.Print(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.tmpl.ExecuteTemplate(w, "errors.html", jobs); err != nil {
		log.Print(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *server) detailHandler(w http.ResponseWriter, r *http.Request) {
	split := strings.Split(r.URL.Path, "/") // ["", "detail", ":id", ...]
	// TODO: don't panic if len(split) < 3
	id, err := strconv.Atoi(split[2])
	if err != nil {
		http.Error(w, "missing :id parameter in path", http.StatusBadRequest)
		return
	}

	var job jobResource
	if err := s.db.Get(&job, s.queries["select_detail.sql"], id); err != nil {
		log.Print(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.tmpl.ExecuteTemplate(w, "detail.html", job); err != nil {
		log.Print(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

type Job struct {
	id      int
	url     string
	retries int
}

func (s *server) run(job Job) {
	done := make(chan struct{})
	s.mu.Lock()
	s.jobs[job.id] = done
	s.mu.Unlock()
	defer close(done)

	tmpdir, err := ioutil.TempDir("", "youtube-ar")
	if err != nil {
		if _, dberr := s.db.Exec(s.queries["update_error.sql"], err.Error(), job.id); dberr != nil {
			log.Print(dberr)
		}
		return
	}
	defer os.RemoveAll(tmpdir)

	var tor *torpkg.Tor
	if job.retries > 0 {
		ctx, cancel := context.WithTimeout(s.ctx, 10*time.Minute)
		defer cancel()
		tor, err = torpkg.New(ctx)
		if err != nil {
			if _, dberr := s.db.Exec(s.queries["update_error.sql"], err.Error(), job.id); dberr != nil {
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
		if _, dberr := s.db.Exec(s.queries["update_geoip.sql"], geoip, job.id); dberr != nil {
			log.Print(dberr)
			return
		}
	}

	output, err := youtubeDL(s.ctx, job.url, tmpdir, tor)
	if err != nil {
		if tor != nil {
			if _, dberr := s.db.Exec(s.queries["update_output_error_torlog.sql"], output, err.Error(), tor.Log(), job.id); dberr != nil {
				log.Print(dberr)
				return
			}
		} else {
			if _, dberr := s.db.Exec(s.queries["update_output_error.sql"], output, err.Error(), job.id); dberr != nil {
				log.Print(dberr)
				return
			}
		}
		if _, ok := err.(geoError); !ok {
			return
		}

		// retry
		var id int
		job.retries++
		if dberr := s.db.Get(&id, s.queries["insert_retries.sql"], job.url, job.retries); dberr != nil {
			log.Print(dberr)
			return
		}
		go s.run(Job{id: id, url: job.url, retries: job.retries})
		return
	}

	if _, dberr := s.db.Exec(s.queries["update_output.sql"], output, time.Now(), job.id); dberr != nil {
		log.Print(dberr)
		return
	}

	if err := s.uploadAllToS3(tmpdir); err != nil {
		if _, dberr := s.db.Exec(s.queries["update_error.sql"], err.Error(), job.id); dberr != nil {
			log.Print(dberr)
		}
		return
	}
	if _, dberr := s.db.Exec(s.queries["update_uploaded_at.sql"], time.Now(), job.id); dberr != nil {
		log.Print(dberr)
	}
}

func (s *server) uploadAllToS3(dir string) error {
	fis, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, fi := range fis {
		if err := s.uploadToS3(filepath.Join(dir, fi.Name())); err != nil {
			return err
		}
	}
	return nil
}

func (s *server) uploadToS3(fname string) error {
	f, err := os.Open(fname)
	if err != nil {
		return err
	}

	defer f.Close()

	_, err = s.s3uploader.UploadWithContext(s.ctx, &s3manager.UploadInput{
		Bucket: aws.String(s.s3bucket),
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
