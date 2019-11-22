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
	ID        int64          `scan:"id"`
	URL       string         `scan:"url"`
	CreatedAt time.Time      `scan:"created_at"`
	UpdatedAt time.Time      `scan:"updated_at"`
	Status    string         `scan:"status"`
	Error     sql.NullString `scan:"error"`
	File      sql.NullString `scan:"file"`
	Retries   sql.NullInt64  `scan:"retries"`
	Logs      pq.StringArray `scan:"logs"`
	OEmbed    []byte         `scan:"oembed"` // json-encoded
}

// ShouldRetry reports whether u failed because of a rate limiter or a geo limitation.
func (u URL) ShouldRetry() bool {
	if u.Error.String == "signal: killed" {
		return true
	}

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
	Log string `scan:"log"`
}

// YoutubeVideo is the youtube video model.
type YoutubeVideo struct {
	ID        int64     `scan:"id"`
	YoutubeID string    `scan:"youtube_id"`
	CreatedAt time.Time `scan:"created_at"`
}

// Page is the page model.
type Page struct {
	Limit  int64
	Cursor int64
}
