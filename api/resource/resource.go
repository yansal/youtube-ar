package resource

import (
	"time"

	"github.com/yansal/youtube-ar/api/model"
)

// URL is the url resource.
type URL struct {
	ID        int64     `json:"id,omitempty"`
	URL       string    `json:"url,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
	Status    string    `json:"status,omitempty"`
	Error     string    `json:"error,omitempty"`
	File      string    `json:"file,omitempty"`
}

// NewURL returns a new URL.
func NewURL(url *model.URL) *URL {
	resource := URL{
		ID:        url.ID,
		URL:       url.URL,
		CreatedAt: url.CreatedAt,
		UpdatedAt: url.UpdatedAt,
		Status:    url.Status,
	}
	if url.Error.Valid {
		resource.Error = url.Error.String
	}
	if url.File.Valid {
		resource.File = url.File.String
	}
	return &resource
}

// NewURLs returns a new URL list.
func NewURLs(urls []model.URL) []URL {
	var resources []URL
	for i := range urls {
		resource := NewURL(&urls[i])
		resources = append(resources, *resource)
	}
	return resources
}

// Log is the log resource.
type Log struct {
	Log string `json:"log,omitempty"`
}

// NewLog returns a new Log.
func NewLog(log *model.Log) *Log {
	resource := Log{Log: log.Log}
	return &resource
}

// Logs is the logs resource.
type Logs struct {
	Logs       []Log `json:"logs"`
	NextCursor int64 `json:"next_cursor"`
}

// NewLogs returns a new Log list.
func NewLogs(logs []model.Log, cursor int64) *Logs {
	var resource Logs
	for i := range logs {
		log := NewLog(&logs[i])
		resource.Logs = append(resource.Logs, *log)
	}
	resource.NextCursor = cursor + int64(len(logs))
	return &resource
}
