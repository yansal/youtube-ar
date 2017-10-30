package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/getsentry/raven-go"
	"golang.org/x/oauth2"
	youtube "google.golang.org/api/youtube/v3"
)

func youtubelikes(cfg cfg) {
	var jsonTokens [][]byte
	if err := cfg.db.Select(&jsonTokens, queries["select_oauth2_token.sql"]); err != nil {
		raven.CaptureError(err, nil)
		return
	}
	for _, jsonToken := range jsonTokens {
		var token oauth2.Token
		if err := json.Unmarshal(jsonToken, &token); err != nil {
			raven.CaptureError(err, nil)
			continue
		}

		ids, err := youtubelikesIDs(cfg.oauth2.Client(context.Background(), &token))
		if err != nil {
			raven.CaptureError(err, nil)
		}
		log.Print(ids)
	}
}

func youtubelikesIDs(client *http.Client) ([]string, error) {
	svc, err := youtube.New(client)
	if err != nil {
		return nil, err
	}

	var (
		ids           []string
		nextPageToken string
	)
	for {
		videoListCall := svc.Videos.List("id").MyRating("like").MaxResults(50)
		if nextPageToken != "" {
			videoListCall = videoListCall.PageToken(nextPageToken)
		}
		resp, err := videoListCall.Do()
		if err != nil {
			return nil, err
		}
		for i := range resp.Items {
			ids = append(ids, resp.Items[i].Id)
		}
		nextPageToken = resp.NextPageToken
		if nextPageToken == "" {
			return ids, nil
		}
	}
}
