package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

type server struct {
	db      *sqlx.DB
	queries map[string]string
	tmpl    *template.Template
}

func newServer(pgConnInfo string) (*server, error) {
	db := sqlx.MustConnect("postgres", pgConnInfo)

	tmpl, err := loadTemplates()
	if err != nil {
		return nil, err
	}

	return &server{
		db:      db,
		queries: queries,
		tmpl:    tmpl,
	}, nil
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if url := r.FormValue("url"); url != "" {
		if _, err := s.db.Exec(s.queries["insert.sql"], url); err != nil {
			log.Print(err)
		}
		http.Redirect(w, r, "/", http.StatusFound)
	}

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
	case p == "/_status/":
		s.statusHandler(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (s *server) runningHandler(w http.ResponseWriter, r *http.Request) {
	var jobs []Job
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
	var jobs []Job
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
	var jobs []Job
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

	var job Job
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

func (s *server) statusHandler(w http.ResponseWriter, r *http.Request) {
	var jobs []Job
	if err := s.db.Select(&jobs, s.queries["select_running.sql"]); err != nil {
		log.Print(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(jobs) > 0 {
		fmt.Fprint(w, "running")
	} else {
		fmt.Fprint(w, "idle")
	}
}

func loadTemplates() (*template.Template, error) {
	t := template.New("").Funcs(template.FuncMap{
		"ago": func(t time.Time) string {
			return fmt.Sprintf("%s (%s)",
				ago(t),
				t.Format("2 Jan 2006 15:04:05 MST"))
		},
	})

	for k, v := range templates {
		_, err := t.New(k).Parse(v)
		if err != nil {
			return nil, err
		}
	}
	return t, nil
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
