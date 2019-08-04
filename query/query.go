package query

import (
	"net/url"
	"strconv"

	"github.com/yansal/youtube-ar/model"
)

// ParsePage parses v.
func ParsePage(v url.Values) (*model.Page, error) {
	page := model.Page{Limit: 10}
	limit := v.Get("limit")
	if limit != "" {
		var err error
		page.Limit, err = strconv.ParseInt(limit, 0, 0)
		if err != nil {
			return nil, err
		}
	}
	cursor := v.Get("cursor")
	if cursor != "" {
		var err error
		page.Cursor, err = strconv.ParseInt(cursor, 0, 0)
		if err != nil {
			return nil, err
		}
	}
	return &page, nil
}
