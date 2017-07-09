package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
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

var (
	httpAddr   = flag.String("addr", "localhost:8080", "HTTP listening address")
	pgConnInfo = flag.String("conninfo", "sslmode=disable", "PostgreSQL connection string")
	s3Bucket   = flag.String("bucket", "", "S3 bucket")
)

var (
	db       *sqlx.DB
	queryMap map[string]string
	tmpl     *template.Template
	uploader *s3manager.Uploader
)

func mustHaveInPath(programs ...string) {
	for _, program := range programs {
		if _, err := exec.LookPath(program); err != nil {
			log.Fatalf(`couldn't find %q in PATH`, program)
		}
	}
}

func mustHaveS3() {
	// TODO: check that s3bucket is present and writable, try to create it if not
	uploader = s3manager.NewUploader(session.Must(session.NewSession()))
}

func mustHaveTemplates() {
	tmpl = template.Must(
		template.New("").Funcs(template.FuncMap{
			"ago": func(t time.Time) string {
				return fmt.Sprintf("%s (%s)",
					ago(t),
					t.Format("2 Jan 2006 15:04:05 MST"))
			},
		}).ParseGlob("templates/*.html"))
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

func mustHaveQueries() {
	fnames, err := filepath.Glob("queries/*.sql")
	if err != nil {
		log.Fatal(err)
	}
	if len(fnames) == 0 {
		log.Fatal("couldn't find queries directory")
	}
	queryMap = make(map[string]string)
	for _, fname := range fnames {
		b, err := ioutil.ReadFile(fname)
		if err != nil {
			log.Fatal(err)
		}
		queryMap[filepath.Base(fname)] = string(b)
	}
}

func mustHaveDB() {
	db = sqlx.MustConnect("postgres", *pgConnInfo)
	db.MustExec(queryMap["create.sql"])
}

var (
	ctx, cancel = context.WithCancel(context.Background())
	mutex       sync.Mutex
	jobs        = make(map[int]chan struct{})
)

func main() {
	flag.Parse()

	mustHaveInPath("tor", "youtube-dl")
	mustHaveS3()
	mustHaveTemplates()
	mustHaveQueries()
	mustHaveDB()

	// Cancel main context before exiting.
	// All jobs will finish their running commands and uploads, and the db will be updated
	exiting := make(chan os.Signal, 1)
	signal.Notify(exiting, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-exiting
		cancel()

		mutex.Lock()
		for _, ch := range jobs {
			<-ch
		}

		os.Exit(0)
	}()

	http.HandleFunc("/", runningHandler)
	http.Handle("/done", http.RedirectHandler("/done/", http.StatusFound))
	http.HandleFunc("/done/", doneHandler)
	http.Handle("/errors", http.RedirectHandler("/errors/", http.StatusFound))
	http.HandleFunc("/errors/", errorsHandler)
	http.HandleFunc("/detail/", detailHandler)

	log.Fatal(http.ListenAndServe(*httpAddr, nil))
}

type JobResource struct {
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

func runningHandler(w http.ResponseWriter, r *http.Request) {
	if url := r.FormValue("url"); url != "" {
		var id int
		if err := db.Get(&id, queryMap["insert.sql"], url); err != nil {
			log.Print(err)
		} else {
			go run(ctx, Job{id: id, url: url})
		}
		http.Redirect(w, r, "/", http.StatusFound)
	}

	var jobs []JobResource
	if err := db.Select(&jobs, queryMap["select_running.sql"]); err != nil {
		log.Print(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "running.html", jobs); err != nil {
		log.Print(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func doneHandler(w http.ResponseWriter, r *http.Request) {
	var jobs []JobResource
	if err := db.Select(&jobs, queryMap["select_done.sql"]); err != nil {
		log.Print(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "done.html", jobs); err != nil {
		log.Print(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func errorsHandler(w http.ResponseWriter, r *http.Request) {
	var jobs []JobResource
	if err := db.Select(&jobs, queryMap["select_error.sql"]); err != nil {
		log.Print(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "errors.html", jobs); err != nil {
		log.Print(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func detailHandler(w http.ResponseWriter, r *http.Request) {
	split := strings.Split(r.URL.Path, "/") // ["", "detail", ":id", ...]
	id, err := strconv.Atoi(split[2])
	if err != nil {
		http.Error(w, "missing :id parameter in path", http.StatusBadRequest)
		return
	}

	var job JobResource
	if err := db.Get(&job, queryMap["select_detail.sql"], id); err != nil {
		log.Print(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "detail.html", job); err != nil {
		log.Print(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

type Job struct {
	id      int
	url     string
	retries int
}

func run(ctx context.Context, job Job) {
	ch := make(chan struct{})
	mutex.Lock()
	jobs[job.id] = ch
	mutex.Unlock()
	defer close(ch)

	tmpdir, err := ioutil.TempDir("", "youtube-ar")
	if err != nil {
		if _, dberr := db.Exec(queryMap["update_error.sql"], err.Error(), job.id); dberr != nil {
			log.Print(dberr)
		}
		return
	}
	defer os.RemoveAll(tmpdir)

	var tor *torpkg.Tor
	if job.retries > 0 {
		ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
		defer cancel()
		tor, err = torpkg.New(ctx)
		if err != nil {
			if _, dberr := db.Exec(queryMap["update_error.sql"], err.Error(), job.id); dberr != nil {
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
		if _, dberr := db.Exec(queryMap["update_geoip.sql"], geoip, job.id); dberr != nil {
			log.Print(dberr)
			return
		}
	}

	output, err := youtubeDL(ctx, job.url, tmpdir, tor)
	if err != nil {
		if tor != nil {
			if _, dberr := db.Exec(queryMap["update_output_error_torlog.sql"], output, err.Error(), tor.Log(), job.id); dberr != nil {
				log.Print(dberr)
				return
			}
		} else {
			if _, dberr := db.Exec(queryMap["update_output_error.sql"], output, err.Error(), job.id); dberr != nil {
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
		if dberr := db.Get(&id, queryMap["insert_retries.sql"], job.url, job.retries); dberr != nil {
			log.Print(dberr)
			return
		}
		go run(ctx, Job{id: id, url: job.url, retries: job.retries})
		return
	}

	if _, dberr := db.Exec(queryMap["update_output.sql"], output, time.Now(), job.id); dberr != nil {
		log.Print(dberr)
		return
	}

	if err := uploadAllToS3(ctx, tmpdir); err != nil {
		if _, dberr := db.Exec(queryMap["update_error.sql"], err.Error(), job.id); dberr != nil {
			log.Print(dberr)
		}
		return
	}
	if _, dberr := db.Exec(queryMap["update_uploaded_at.sql"], time.Now(), job.id); dberr != nil {
		log.Print(dberr)
	}
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

type geoError struct{ error }

func (e geoError) Error() string {
	return e.error.Error()
}

func uploadAllToS3(ctx context.Context, dir string) error {
	fis, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, fi := range fis {
		if err := uploadToS3(ctx, filepath.Join(dir, fi.Name())); err != nil {
			return err
		}
	}
	return nil
}

func uploadToS3(ctx context.Context, fname string) error {
	f, err := os.Open(fname)
	if err != nil {
		return err
	}

	defer f.Close()

	_, err = uploader.UploadWithContext(ctx, &s3manager.UploadInput{
		Bucket: aws.String(*s3Bucket),
		Key:    aws.String(filepath.Base(fname)),
		Body:   f,
	})
	return err
}
