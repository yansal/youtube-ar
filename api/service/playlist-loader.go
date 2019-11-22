package service

import (
	"context"
	"database/sql"

	"github.com/yansal/sql/scan"
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
	CreateURL(context.Context, scan.Queryer, payload.URL) (*model.URL, error)
}

// PlaylistLoaderStore is the store interface required by PlaylistLoader.
type PlaylistLoaderStore interface {
	GetYoutubeVideoByYoutubeID(context.Context, scan.Queryer, string) (*model.YoutubeVideo, error)
	CreateYoutubeVideo(context.Context, scan.Queryer, *model.YoutubeVideo) error
}

// PlaylistLoaderYoutubeClient is the youtube client interface required by PlaylistLoader.
type PlaylistLoaderYoutubeClient interface {
	GetVideosFromPlaylist(context.Context, string) ([]youtube.Video, error)
}

// NewPlaylistLoader returns a new PlaylistLoader.
func NewPlaylistLoader(manager PlaylistLoaderManager, store PlaylistLoaderStore, youtubeClient PlaylistLoaderYoutubeClient) *PlaylistLoader {
	return &PlaylistLoader{manager: manager, store: store, youtubeClient: youtubeClient}
}

type Queryer interface {
	scan.Queryer
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

// CreateURLsFromYoutube creates urls from youtube playlistID.
func (s *PlaylistLoader) CreateURLsFromYoutube(ctx context.Context, db Queryer, playlistID string) error {
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

func (s *PlaylistLoader) getOrCreateYoutubeVideo(ctx context.Context, db Queryer, youtubeID string) (*model.YoutubeVideo, error) {
	v, err := s.store.GetYoutubeVideoByYoutubeID(ctx, db, youtubeID)
	if err == nil {
		return v, nil
	} else if err != sql.ErrNoRows {
		return nil, err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	v = &model.YoutubeVideo{YoutubeID: youtubeID}
	if err := s.store.CreateYoutubeVideo(ctx, tx, v); err != nil {
		return nil, err
	}

	p := payload.URL{URL: "https://www.youtube.com/watch?v=" + youtubeID}
	if _, err := s.manager.CreateURL(ctx, tx, p); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return v, nil
}
