package resource

import (
	"os"
	"time"

	"github.com/yansal/youtube-ar/api/model"
)

// Serializer is a resource serializer.
type Serializer struct {
	mediaURL string
}

// NewSerializer returns a new serializer.
func NewSerializer() *Serializer {
	return &Serializer{mediaURL: "https://" + os.Getenv("S3_BUCKET") + ".s3." + os.Getenv("AWS_REGION") + ".amazonaws.com/"}
}

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
func (s *Serializer) NewURL(url *model.URL) *URL {
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
		resource.File = s.mediaURL + url.File.String
	}
	return &resource
}

// URLs is the urls resource.
type URLs struct {
	URLs       []URL `json:"urls"`
	NextCursor int64 `json:"next_cursor"`
}

// NewURLs returns a new URL list.
func (s *Serializer) NewURLs(urls []model.URL) *URLs {
	var resource URLs
	for i := range urls {
		url := s.NewURL(&urls[i])
		resource.URLs = append(resource.URLs, *url)
	}
	len := len(urls)
	if len > 0 {
		resource.NextCursor = urls[len-1].ID
	}
	return &resource
}

// Log is the log resource.
type Log struct {
	Log string `json:"log,omitempty"`
}

// NewLog returns a new Log.
func (s *Serializer) NewLog(log *model.Log) *Log {
	resource := Log{Log: log.Log}
	return &resource
}

// Logs is the logs resource.
type Logs struct {
	Logs       []Log `json:"logs"`
	NextCursor int64 `json:"next_cursor"`
}

// NewLogs returns a new Log list.
func (s *Serializer) NewLogs(logs []model.Log, cursor int64) *Logs {
	var resource Logs
	for i := range logs {
		log := s.NewLog(&logs[i])
		resource.Logs = append(resource.Logs, *log)
	}
	resource.NextCursor = cursor + int64(len(logs))
	return &resource
}
