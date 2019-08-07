package query

import (
	"net/url"

	"github.com/yansal/query"
)

// ParseURLs parses v and returns a new URLs.
func ParseURLs(v url.Values) (*URLs, error) {
	q, err := query.Validate(v,
		query.WithIntParam("limit"),
		query.WithIntParam("cursor"),
		query.WithStringsParam("status", []string{"pending", "processing", "failure", "success"}),
	)
	if err != nil {
		return nil, err
	}
	var u URLs
	if cursor, ok := q.Params["cursor"]; ok {
		u.Page.Cursor = cursor.(int64)
	}
	if limit, ok := q.Params["limit"]; !ok {
		u.Page.Limit = DefaultLimit
	} else {
		u.Page.Limit = limit.(int64)
	}
	if status, ok := q.Params["status"]; ok {
		u.Status = status.([]string)
	}
	return &u, nil
}

// ParseLogs parses v and returns a new Logs.
func ParseLogs(v url.Values) (*Logs, error) {
	q, err := query.Validate(v,
		query.WithIntParam("limit"),
		query.WithIntParam("cursor"),
		query.WithStringsParam("status", []string{"pending", "processing", "failure", "success"}),
	)
	if err != nil {
		return nil, err
	}
	var l Logs
	if cursor, ok := q.Params["cursor"]; ok {
		l.Page.Cursor = cursor.(int64)
	}
	if limit, ok := q.Params["limit"]; !ok {
		l.Page.Limit = DefaultLimit
	} else {
		l.Page.Limit = limit.(int64)
	}
	return &l, nil
}

// URLs is the query for urls.
type URLs struct {
	Page   Page
	Status []string
}

// Logs is the query for logs.
type Logs struct {
	Page Page
}

// Page is a paginated query.
type Page struct {
	Cursor int64
	Limit  int64
}

// DefaultLimit is the default limit.
const DefaultLimit int64 = 10
