package handler

import (
	"context"
	"encoding/json"

	"github.com/yansal/youtube-ar/broker"
	"github.com/yansal/youtube-ar/event"
)

// Manager is the manager interface required by URLCreated.
type Manager interface {
	ProcessURL(context.Context, event.URL) error
}

// URLCreated is the url-created handler.
func URLCreated(m Manager) broker.Handler {
	return func(ctx context.Context, payload string) error {
		var e event.URL
		if err := json.Unmarshal([]byte(payload), &e); err != nil {
			return err
		}

		return m.ProcessURL(ctx, e)
	}
}
