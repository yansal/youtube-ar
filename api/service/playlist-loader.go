package service

import (
	"context"
	"database/sql"

	"github.com/yansal/youtube-ar/api/model"
	"github.com/yansal/youtube-ar/api/payload"
	"github.com/yansal/youtube-ar/api/youtube"
)

// PlaylistLoader is a playlist loader.
type PlaylistLoader struct {
	manager       PlaylistLoaderManager
	store         PlaylistLoaderStore
	youtubeClient PlaylistLoaderYoutubeClient
}

// PlaylistLoaderManager is the manager interface required by PlaylistLoader.
type PlaylistLoaderManager interface {
	CreateURL(context.Context, payload.URL) (*model.URL, error)
}

// PlaylistLoaderStore is the store interface required by PlaylistLoader.
type PlaylistLoaderStore interface {
	CreateYoutubeVideo(context.Context, *model.YoutubeVideo) error
}

// PlaylistLoaderYoutubeClient is the youtube client interface required by PlaylistLoader.
type PlaylistLoaderYoutubeClient interface {
	GetVideosFromPlaylist(context.Context, string) ([]youtube.Video, error)
}

// NewPlaylistLoader returns a new PlaylistLoader.
func NewPlaylistLoader(manager PlaylistLoaderManager, store PlaylistLoaderStore, youtubeClient PlaylistLoaderYoutubeClient) *PlaylistLoader {
	return &PlaylistLoader{manager: manager, store: store, youtubeClient: youtubeClient}
}

// CreateURLsFromYoutube creates urls from youtube playlistID.
func (s *PlaylistLoader) CreateURLsFromYoutube(ctx context.Context, playlistID string) error {
	videos, err := s.youtubeClient.GetVideosFromPlaylist(ctx, playlistID)
	if err != nil {
		return err
	}

	for i := range videos {
		youtubeID := videos[i].ID
		v := &model.YoutubeVideo{YoutubeID: youtubeID}
		if err := s.store.CreateYoutubeVideo(ctx, v); err == sql.ErrNoRows {
			continue
		} else if err != nil {
			return err
		}

		p := payload.URL{URL: "https://www.youtube.com/watch?v=" + youtubeID}
		if _, err := s.manager.CreateURL(ctx, p); err != nil {
			return err
		}
	}

	return nil
}
