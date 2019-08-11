package youtubedl

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"sync"
)

// New returns a new YoutubeDL.
func New() *YoutubeDL {
	return &YoutubeDL{}
}

// YoutubeDL is a downloader.
type YoutubeDL struct{}

// Download downloads url and returns a stream of Event.
func (p *YoutubeDL) Download(ctx context.Context, url string) <-chan Event {
	stream := make(chan Event)
	go func() {
		defer close(stream)

		dir, err := ioutil.TempDir("", "youtube-ar-")
		if err != nil {
			stream <- Event{Type: Failure, Err: err}
			return
		}

		cmd := exec.CommandContext(ctx, "youtube-dl", "--newline", "--verbose", url)
		cmd.Dir = dir

		// stream stderr and stdout
		stderr, err := cmd.StderrPipe()
		if err != nil {
			stream <- Event{Type: Failure, Err: err}
			return
		}
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			stream <- Event{Type: Failure, Err: err}
			return
		}
		var wg sync.WaitGroup
		wg.Add(2)
		slurp := func(r io.Reader) {
			defer wg.Done()
			s := bufio.NewScanner(r)
			for s.Scan() {
				stream <- Event{Type: Log, Log: s.Text()}
			}
		}
		go slurp(stderr)
		go slurp(stdout)

		if err := cmd.Start(); err != nil {
			stream <- Event{Type: Failure, Err: err}
			return
		}

		wg.Wait()
		if err := cmd.Wait(); err != nil {
			stream <- Event{Type: Failure, Err: err}
			return
		}

		fis, err := ioutil.ReadDir(dir)
		if err != nil {
			stream <- Event{Type: Failure, Err: err}
			return
		}
		if len(fis) != 1 {
			err := fmt.Errorf("expected 1 file in %s, got %d", dir, len(fis))
			stream <- Event{Type: Failure, Err: err}
			return
		}
		stream <- Event{Type: Success, Path: filepath.Join(dir, fis[0].Name())}
	}()
	return stream
}

// Event is a downloader event.
type Event struct {
	Type EventType
	Log  string
	Err  error
	Path string
}

// EventType is an event type.
type EventType int

// EventType values.
const (
	Log EventType = iota
	Failure
	Success
)
