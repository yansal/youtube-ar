package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"

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

	if err := g.Wait(); err != sentinel && err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

type cmd func(ctx context.Context, args []string) error
