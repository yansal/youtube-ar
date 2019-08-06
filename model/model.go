package model

import (
	"database/sql"
	"regexp"
	"strings"
	"time"

	"github.com/lib/pq"
)

// URL is the url model.
type URL struct {
	ID        int64
	URL       string
	CreatedAt time.Time
	UpdatedAt time.Time
	Status    string
	Error     sql.NullString
	File      sql.NullString
	Retries   sql.NullInt64
	Logs      pq.StringArray
}

// ShouldRetry reports whether failed because of a rate limiter or a geo limitation.
func (u URL) ShouldRetry() bool {
	if u.Error.String != "exit status 1" {
		return false
	}
	log := strings.Join(u.Logs, "\n")
	for _, re := range shouldRetryRegexps {
		if re.MatchString(log) {
			return true
		}
	}
	return false
}

var shouldRetryRegexps = []*regexp.Regexp{
	regexp.MustCompile(`ERROR: Unable to download webpage: HTTP Error 429: Too Many Requests`),
	regexp.MustCompile(`ERROR: The uploader has not made this video available in your country\.`),
	regexp.MustCompile(`ERROR: .*: YouTube said: This video contains content from .*, who has blocked it on copyright grounds\.`),
}

// Log is the log model.
type Log struct {
	Log string
}

// YoutubeVideo is the youtube video model.
type YoutubeVideo struct {
	ID        int64
	YoutubeID string
	CreatedAt time.Time
}

// Page is the page model.
type Page struct {
	Limit  int64
	Cursor int64
}
