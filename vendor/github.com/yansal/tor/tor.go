package tor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
)

type Tor struct {
	ListenAddr string
	GeoIP      GeoIP
	datadir    string
	process    *os.Process
	log        *log
}

type GeoIP struct {
	IP          string  `json:"ip"`
	CountryCode string  `json:"country_code"`
	CountryName string  `json:"country_name"`
	RegionCode  string  `json:"region_code"`
	RegionName  string  `json:"region_name"`
	City        string  `json:"city"`
	ZipCode     string  `json:"zip_code"`
	TimeZone    string  `json:"time_zone"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	MetroCode   int     `json:"metro_code"`
}

type log struct {
	mu    sync.Mutex
	buf   bytes.Buffer
	grep  []byte
	found chan<- struct{}
}

func (l *log) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.grep != nil && bytes.Contains(p, l.grep) {
		l.found <- struct{}{}
	}

	return l.buf.Write(p)
}

func (l *log) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.String()
}

func New(ctx context.Context) (*Tor, error) {
	tmpdir, err := ioutil.TempDir("", "tordatadir")
	if err != nil {
		return nil, err
	}
	port := randomPort()
	cmd := exec.CommandContext(ctx, "tor", "-f", "-")
	torrc := fmt.Sprintf(`Log notice
DataDirectory %s
SocksPort %d`, tmpdir, port)
	cmd.Stdin = strings.NewReader(torrc)

	ready := make(chan struct{})
	log := &log{
		grep:  []byte("[notice] Bootstrapped 100%: Done"),
		found: ready,
	}
	cmd.Stdout = log
	cmd.Stderr = log

	tor := &Tor{
		ListenAddr: fmt.Sprintf("localhost:%d", port),
		datadir:    tmpdir,
		log:        log,
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	tor.process = cmd.Process

	select {
	case <-ready:
	case <-ctx.Done():
		tor.Shutdown()
		return nil, fmt.Errorf("timeout waiting for tor:\n%v", tor.Log())
	}

	geoip, err := geoip(tor.ListenAddr)
	if err != nil {
		tor.Shutdown()
		return nil, err
	}
	tor.GeoIP = geoip

	return tor, nil
}

func (tor *Tor) Shutdown() error {
	defer os.RemoveAll(tor.datadir)

	if err := tor.process.Signal(os.Interrupt); err != nil {
		return err
	}
	_, err := tor.process.Wait()
	return err
}

func (tor *Tor) Log() string {
	return tor.log.String()
}

// randomPort generates a random int between 10000 and 19999
// TODO: ask to OS for a random port
func randomPort() int { return rand.Intn(10000) + 10000 }

func geoip(torAddr string) (GeoIP, error) {
	// Ask tor exit IP to check.torproject.org
	httpClient := http.Client{Transport: &http.Transport{
		Proxy: func(*http.Request) (*url.URL, error) {
			return url.Parse("socks5://" + torAddr)
		},
	}}
	resp, err := httpClient.Get("https://check.torproject.org/api/ip")
	if err != nil {
		return GeoIP{}, err
	}
	defer resp.Body.Close()
	var ip struct {
		IP string
	}
	if err := json.NewDecoder(resp.Body).Decode(&ip); err != nil {
		return GeoIP{}, err
	}

	// Ask geoip data to freegeoip
	// TODO: use a local geoip database
	resp, err = http.Get("https://freegeoip.net/json/" + ip.IP)
	if err != nil {
		return GeoIP{}, err
	}
	defer resp.Body.Close()

	var geoip GeoIP
	if err := json.NewDecoder(resp.Body).Decode(&geoip); err != nil {
		return GeoIP{}, err
	}
	return geoip, nil
}
