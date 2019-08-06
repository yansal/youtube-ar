package manager

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/yansal/youtube-ar/downloader"
	"github.com/yansal/youtube-ar/event"
	"github.com/yansal/youtube-ar/model"
)

func assertf(t *testing.T, ok bool, msg string, args ...interface{}) {
	t.Helper()
	if !ok {
		t.Errorf(msg, args...)
	}
}

type downloaderMock struct {
	downloadFunc func(context.Context, string) <-chan downloader.Event
}

func (d downloaderMock) Download(ctx context.Context, url string) <-chan downloader.Event {
	return d.downloadFunc(ctx, url)
}

type storageMock struct {
	uploadFunc func(context.Context, string) (string, error)
}

func (s storageMock) Upload(ctx context.Context, url string) (string, error) {
	return s.uploadFunc(ctx, url)
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

func (s storeMock) CreateLog(ctx context.Context, urlID int64, log *model.Log) error {
	return nil
}

func TestProcessURLFailure(t *testing.T) {
	serr := "err"
	m := Worker{
		downloader: downloaderMock{
			downloadFunc: func(context.Context, string) <-chan downloader.Event {
				stream := make(chan downloader.Event)
				go func() {
					stream <- downloader.Event{
						Type: downloader.Failure,
						Err:  errors.New(serr),
					}
					close(stream)
				}()
				return stream
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

	err := m.ProcessURL(context.Background(), event.URL{})
	assertf(t, err.Error() == serr,
		`expected err to be %q, got %+v`, serr, err,
	)
}

func TestProcessURLSuccess(t *testing.T) {
	file := "file.go"
	m := Worker{
		downloader: downloaderMock{
			downloadFunc: func(context.Context, string) <-chan downloader.Event {
				stream := make(chan downloader.Event)
				go func() {
					stream <- downloader.Event{
						Type: downloader.Success,
						Path: file,
					}
					close(stream)
				}()
				return stream
			},
		},
		storage: storageMock{
			uploadFunc: func(ctx context.Context, in string) (string, error) {
				assertf(t, in == file,
					`expected file to be %q, got %q`, file, in,
				)
				return in, nil
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

	err := m.ProcessURL(context.Background(), event.URL{})
	assertf(t, err == nil, `expected err to be nil, got %+v`, err)
}

func TestProcessURLPanic(t *testing.T) {
	var (
		unlocked bool
		serr     = "panic"
	)
	m := Worker{
		downloader: downloaderMock{
			downloadFunc: func(context.Context, string) <-chan downloader.Event {
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
	_ = m.ProcessURL(context.Background(), event.URL{})
	t.Error("expected panic")
}
