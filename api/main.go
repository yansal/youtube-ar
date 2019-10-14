package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/yansal/youtube-ar/api/log"
	"golang.org/x/sync/errgroup"
)

func main() {
	cmds := map[string]cmd{
		"create-url":                createURL,
		"create-urls-from-playlist": createURLsFromPlaylist,
		"download-url":              downloadURL,
		"get-oembed":                getOembed,
		"list-logs":                 listLogs,
		"list-urls":                 listURLs,
		"retry-next-download-url":   retryNextDownloadURL,
		"should-retry":              shouldRetry,
		"server":                    runServer,
		"worker":                    runWorker,
	}

	var names []string
	for k := range cmds {
		names = append(names, k)
	}
	sort.Strings(names)

	cmd := runServer
	args := os.Args[1:]
	if len(os.Args) > 1 {
		var ok bool
		cmd, ok = cmds[os.Args[1]]
		if !ok {
			fmt.Printf("unknown cmd %s\n", os.Args[1])
			fmt.Printf("usage: %s [%s]\n", os.Args[0], strings.Join(names, "|"))
			os.Exit(2)
		}
		args = os.Args[2:]
	}

	sentinel := errors.New("sentinel")
	g, ctx := errgroup.WithContext(context.Background())
	g.Go(func() error {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		select {
		case <-ctx.Done():
		case <-c:
		}
		return sentinel
	})
	g.Go(func() error {
		if err := cmd(ctx, args); err != nil {
			return err
		}
		return sentinel
	})

	if pprofPort := os.Getenv("PPROF_PORT"); pprofPort != "" {
		g.Go(func() error {
			if err := runPprofServer(ctx, pprofPort); err != nil {
				return err
			}
			return sentinel
		})
	}

	if err := g.Wait(); err != sentinel && err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

type cmd func(ctx context.Context, args []string) error

func runPprofServer(ctx context.Context, port string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	server := http.Server{Handler: mux}

	l, err := net.Listen("tcp", "localhost:"+port)
	if err != nil {
		return err
	}

	cerr := make(chan error)
	go func() {
		cerr <- server.Serve(l)
	}()

	select {
	case <-ctx.Done():
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.New().Log(ctx, err.Error())
		}
		return nil
	case err := <-cerr:
		return err
	}
}
