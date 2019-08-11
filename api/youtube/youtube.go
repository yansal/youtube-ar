package youtube

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"os"

	"github.com/yansal/youtube-ar/api/log"
	loghttp "github.com/yansal/youtube-ar/api/log/http"
)

// Video is the video resource.
type Video struct {
	ID string
}

// New returns a new Client.
func New(log log.Logger) *Client {
	return &Client{
		apiKey: os.Getenv("YOUTUBE_API_KEY"),
		client: loghttp.Wrap(new(http.Client), log),
	}
}

// A Client is a youtube client.
type Client struct {
	apiKey string
	client *http.Client
}

// GetVideosFromPlaylist gets videos from a youtube playlist.
func (c *Client) GetVideosFromPlaylist(ctx context.Context, playlist string) ([]Video, error) {
	g := getPlaylistItemsRequest{playlistID: playlist}
	var videos []Video
	for {
		items, err := c.getPlaylistItems(ctx, g)
		if err != nil {
			return nil, err
		}
		for i := range items.Items {
			videos = append(videos, Video{ID: items.Items[i].Snippet.ResourceID.VideoID})
		}
		if items.NextPageToken == "" {
			break
		}
		g.pageToken = items.NextPageToken
	}

	return videos, nil
}

type getPlaylistItemsRequest struct {
	pageToken  string
	playlistID string
}

func (c *Client) getPlaylistItems(ctx context.Context, g getPlaylistItemsRequest) (*playlistItems, error) {
	u, err := url.Parse("https://www.googleapis.com/youtube/v3/playlistItems?part=snippet&maxResults=50")
	if err != nil {
		return nil, err
	}

	q := u.Query()
	if g.playlistID != "" {
		q.Set("playlistId", g.playlistID)
	}
	if g.pageToken != "" {
		q.Set("pageToken", g.pageToken)
	}
	if c.apiKey != "" {
		q.Set("key", c.apiKey)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var items playlistItems
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, err
	}
	return &items, nil
}

type playlistItems struct {
	NextPageToken string `json:"nextPageToken"`
	Items         []struct {
		Snippet struct {
			ResourceID struct {
				VideoID string `json:"videoId"`
			} `json:"resourceId"`
		} `json:"snippet"`
	} `json:"items"`
}
