package handler

import (
	"context"
	"encoding/json"

	"github.com/yansal/youtube-ar/api/broker"
	"github.com/yansal/youtube-ar/api/event"
)

// Manager is the manager interface required by worker handlers.
type Manager interface {
	DownloadURL(context.Context, event.URL) error
	GetOEmbed(context.Context, event.URL) error
}

// DownloadURL is the download-url handler.
func DownloadURL(m Manager) broker.Handler {
	return func(ctx context.Context, payload string) error {
		var e event.URL
		if err := json.Unmarshal([]byte(payload), &e); err != nil {
			return err
		}

		return m.DownloadURL(ctx, e)
	}
}

// GetOEmbed is the get-oembed handler.
func GetOEmbed(m Manager) broker.Handler {
	return func(ctx context.Context, payload string) error {
		var e event.URL
		if err := json.Unmarshal([]byte(payload), &e); err != nil {
			return err
		}

		return m.GetOEmbed(ctx, e)
	}
}
