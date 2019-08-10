package payload

import "net/url"

// URL is the url payload.
type URL struct {
	URL string `json:"url"`

	Retries int64 `json:"-"`
}

// Validate returns an error if u is invalid.
func (u *URL) Validate() error {
	_, err := url.Parse(u.URL)
	return err
}
