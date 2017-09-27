//go:generate go run generate_embed.go
package main

import (
	"fmt"
	"html/template"
	"os/exec"
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

func isInPath(programs ...string) error {
	for _, program := range programs {
		if _, err := exec.LookPath(program); err != nil {
			return fmt.Errorf("couldn't find %q in PATH", program)
		}
	}
	return nil
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
