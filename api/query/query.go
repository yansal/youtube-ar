package query

import (
	"net/url"

	"github.com/yansal/query"
)

// ParseURLs parses v and returns a new URLs.
func ParseURLs(v url.Values) (*URLs, error) {
	q, err := query.Validate(v,
		query.IntParam("limit"),
		query.IntParam("cursor"),
		query.StringsParam("status", []string{"pending", "processing", "failure", "success"}),
	)
	if err != nil {
		return nil, err
	}
	var u URLs
	if cursor, ok := q["cursor"]; ok {
		u.Cursor = cursor.(int64)
	}
	if limit, ok := q["limit"]; !ok {
		u.Limit = DefaultLimit
	} else {
		u.Limit = limit.(int64)
	}
	if status, ok := q["status"]; ok {
		u.Status = status.([]string)
	}
	return &u, nil
}

// ParseLogs parses v and returns a new Logs.
func ParseLogs(v url.Values) (*Logs, error) {
	q, err := query.Validate(v, query.IntParam("cursor"))
	if err != nil {
		return nil, err
	}
	var l Logs
	if cursor, ok := q["cursor"]; ok {
		l.Cursor = cursor.(int64)
	}
	return &l, nil
}

// URLs is the query for urls.
type URLs struct {
	Cursor int64
	Limit  int64
	Status []string
}

// Logs is the query for logs.
type Logs struct {
	Cursor int64
}

// DefaultLimit is the default limit.
const DefaultLimit int64 = 10
