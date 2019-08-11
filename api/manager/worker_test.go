package manager

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/yansal/youtube-ar/api/event"
	"github.com/yansal/youtube-ar/api/model"
)

func assertf(t *testing.T, ok bool, msg string, args ...interface{}) {
	t.Helper()
	if !ok {
		t.Errorf(msg, args...)
	}
}

type dowloaderMock struct {
	downloadURLFunc func(context.Context, *model.URL) (string, error)
}

func (p dowloaderMock) DownloadURL(ctx context.Context, url *model.URL) (string, error) {
	return p.downloadURLFunc(ctx, url)
}

type storeMock struct {
	unlockURLFunc func(context.Context, *model.URL) error
}

func (s storeMock) LockURL(ctx context.Context, url *model.URL) error {
	return nil
}

func (s storeMock) UnlockURL(ctx context.Context, url *model.URL) error {
	return s.unlockURLFunc(ctx, url)
}

func TestDownloadURLFailure(t *testing.T) {
	serr := "err"
	m := Worker{
		downloader: dowloaderMock{
			downloadURLFunc: func(ctx context.Context, url *model.URL) (string, error) {
				return "", errors.New(serr)
			},
		},
		store: storeMock{
			unlockURLFunc: func(ctx context.Context, url *model.URL) error {
				assertf(t, url.Status == "failure",
					`expected status to be "failure", got %q`, url.Status,
				)
				assertf(t, url.Error == sql.NullString{Valid: true, String: serr},
					`expected error to be valid and equal to %q, got %+v`, serr, url.Error,
				)
				assertf(t, !url.File.Valid,
					`expected file to not be valid, got %+v`, url.File,
				)
				return nil
			},
		},
	}

	err := m.DownloadURL(context.Background(), event.URL{})
	assertf(t, err.Error() == serr,
		`expected err to be %q, got %+v`, serr, err,
	)
}

func TestDownloadURLSuccess(t *testing.T) {
	file := "file.go"
	m := Worker{
		downloader: dowloaderMock{
			downloadURLFunc: func(ctx context.Context, url *model.URL) (string, error) {
				return file, nil
			},
		},
		store: storeMock{
			unlockURLFunc: func(ctx context.Context, url *model.URL) error {
				assertf(t, url.Status == "success",
					`expected status to be "success", got %q`, url.Status,
				)
				assertf(t, !url.Error.Valid,
					`expected error to not be valid, got %+v`, url.Error,
				)
				assertf(t, url.File == sql.NullString{Valid: true, String: file},
					`expected file to be valid and equal to %q, got %+v`, file, url.File,
				)
				return nil
			},
		},
	}

	err := m.DownloadURL(context.Background(), event.URL{})
	assertf(t, err == nil, `expected err to be nil, got %+v`, err)
}

func TestDownloadURLPanic(t *testing.T) {
	var (
		unlocked bool
		serr     = "panic"
	)
	m := Worker{
		downloader: dowloaderMock{
			downloadURLFunc: func(ctx context.Context, url *model.URL) (string, error) {
				panic(serr)
			},
		},
		store: storeMock{
			unlockURLFunc: func(ctx context.Context, url *model.URL) error {
				unlocked = true
				assertf(t, url.Status == "failure",
					`expected status to be "failure", got %q`, url.Status,
				)
				assertf(t, url.Error == sql.NullString{Valid: true, String: serr},
					`expected error to be valid and equal to %q, got %+v`, serr, url.Error,
				)
				assertf(t, !url.File.Valid,
					`expected file to not be valid, got %+v`, url.File,
				)
				return nil
			},
		},
	}

	defer func() {
		if r := recover(); r != nil {
			assertf(t, r == serr,
				`expected err to be %q, got %+v`, serr, r,
			)
			assertf(t, unlocked, `expected the unlock method to be called`)
		}
	}()
	_ = m.DownloadURL(context.Background(), event.URL{})
	t.Error("expected panic")
}
