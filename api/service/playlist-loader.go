package service

import (
	"context"
	"database/sql"

	"github.com/yansal/sql/nest"
	"github.com/yansal/youtube-ar/api/model"
	"github.com/yansal/youtube-ar/api/payload"
	"github.com/yansal/youtube-ar/api/store"
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
	CreateURL(context.Context, nest.Querier, payload.URL) (*model.URL, error)
}

// PlaylistLoaderStore is the store interface required by PlaylistLoader.
type PlaylistLoaderStore interface {
	GetYoutubeVideoByYoutubeID(context.Context, nest.Querier, string) (*model.YoutubeVideo, error)
	CreateYoutubeVideo(context.Context, nest.Querier, *model.YoutubeVideo) error
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
func (s *PlaylistLoader) CreateURLsFromYoutube(ctx context.Context, db nest.Querier, playlistID string) error {
	videos, err := s.youtubeClient.GetVideosFromPlaylist(ctx, playlistID)
	if err != nil {
		return err
	}

	for i := range videos {
		youtubeID := videos[i].ID
		if _, err := s.getOrCreateYoutubeVideo(ctx, db, youtubeID); err != nil {
			// TODO: log err, don't return
			return err
		}
	}

	return nil
}

func (s *PlaylistLoader) getOrCreateYoutubeVideo(ctx context.Context, db nest.Querier, youtubeID string) (*model.YoutubeVideo, error) {
	v, err := s.store.GetYoutubeVideoByYoutubeID(ctx, db, youtubeID)
	if err == nil {
		return v, nil
	} else if err != sql.ErrNoRows {
		return nil, err
	}

	v = &model.YoutubeVideo{YoutubeID: youtubeID}
	if err := store.Transaction(ctx, db, func(ctx context.Context, tx nest.Querier) error {
		if err := s.store.CreateYoutubeVideo(ctx, tx, v); err != nil {
			return err
		}

		p := payload.URL{URL: "https://www.youtube.com/watch?v=" + youtubeID}
		if _, err := s.manager.CreateURL(ctx, tx, p); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return v, nil
}
