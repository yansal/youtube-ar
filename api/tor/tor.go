package tor

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"sync"
)

// Tor is a tor process starter.
type Tor struct{}

// New returns a new Tor.
func New() *Tor {
	return &Tor{}
}

// Start starts a new tor process.
func (*Tor) Start(ctx context.Context) <-chan Event {
	stream := make(chan Event)
	go func() {
		defer close(stream)
		dir, err := ioutil.TempDir("", "youtube-ar-tor-")
		if err != nil {
			stream <- Event{Type: Failure, Err: err}
			return
		}
		defer os.RemoveAll(dir)

		port := getRandomPort()
		cmd := exec.CommandContext(ctx, "tor", "-f", "-")
		cmd.Stdin = strings.NewReader(fmt.Sprintf(torrcformat, dir, port))

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
				if strings.Contains(s.Text(), `Bootstrapped 100%`) {
					stream <- Event{Type: Ready, ProxyURL: "socks5://localhost:" + port}
				}
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
	}()
	return stream
}

func getRandomPort() string {
	// TODO: get a random port from the OS, just like net.Listen does
	return "9051"
}

const torrcformat = `DataDirectory %s
SocksPort %s`

// Event is a tor process event.
type Event struct {
	Type     EventType
	Log      string
	Err      error
	ProxyURL string
}

// EventType is an event type.
type EventType int

// EventType values.
const (
	Log EventType = iota
	Failure
	Ready
)
