package oembed

import (
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	urlpkg "net/url"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/net/html"
)

// NewClient returns a new client.
func NewClient(httpclient *http.Client) *Client {
	return &Client{
		client: httpclient,
	}
}

// Client is an oembed client.
type Client struct {
	client *http.Client

	loadProvidersOnce sync.Once
	providers         []provider
}

type provider struct {
	pattern  string
	endpoint endpoint
}

// Get gets oembed data.
func (c *Client) Get(ctx context.Context, url string) ([]byte, error) {
	oembedURL, err := c.find(ctx, url)
	if err != nil {
		return nil, err
	}
	if oembedURL == "" {
		return nil, errors.New("couldn't find oembed URL")
	}

	req, err := http.NewRequest(http.MethodGet, oembedURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.client.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}

func (c *Client) find(ctx context.Context, url string) (string, error) {
	c.loadProvidersOnce.Do(c.loadProviders)

	// first, lookup in known providers
	for _, provider := range c.providers {
		if ok, err := filepath.Match(provider.pattern, url); err != nil {
			return "", err
		} else if !ok {
			continue
		}
		if provider.endpoint.Discovery {
			break // fallback to discover
		}

		oembedURL := *provider.endpoint.URL
		oembedURL.RawQuery = urlpkg.Values{"url": []string{url}}.Encode()
		return oembedURL.String(), nil
	}

	// second, discover
	return c.discover(ctx, url)
}

func (c *Client) discover(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := c.client.Do(req.WithContext(ctx))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return "", err
	}

	var f func(*html.Node) string
	f = func(n *html.Node) string {
		if href := findHref(n); href != "" {
			return href
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if href := f(c); href != "" {
				return href
			}
		}
		return ""
	}
	return f(doc), nil
}

func findHref(n *html.Node) string {
	if n.Type != html.ElementNode || n.Data != "link" {
		return ""
	}

	var ok bool
	for i := range n.Attr {
		if n.Attr[i].Key == "type" && strings.HasSuffix(n.Attr[i].Val, "+oembed") {
			ok = true
			break
		}
	}
	if !ok {
		return ""
	}
	for i := range n.Attr {
		if n.Attr[i].Key == "href" {
			return n.Attr[i].Val
		}
	}
	return ""
}
