package main

import (
	"fmt"
	"go/build"
	"html/template"
	"io/ioutil"
	"log"
	"path/filepath"
	"time"
)

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
		return nil, fmt.Errorf("could not find templates directory: %v", err)
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
