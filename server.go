package main

import (
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"

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
