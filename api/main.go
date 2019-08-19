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

	"github.com/yansal/youtube-ar/api/cmd"
	"golang.org/x/sync/errgroup"
)

func main() {
	cmds := map[string]cmd.Cmd{
		"create-url":                cmd.CreateURL,
		"create-urls-from-playlist": cmd.CreateURLsFromPlaylist,
		"download-url":              cmd.DownloadURL,
		"get-oembed":                cmd.GetOembed,
		"list-logs":                 cmd.ListLogs,
		"list-urls":                 cmd.ListURLs,
		"retry-next-download-url":   cmd.RetryNextDownloadURL,
		"server":                    cmd.Server,
		"worker":                    cmd.Worker,
	}

	var names []string
	for k := range cmds {
		names = append(names, k)
	}
	sort.Strings(names)

	cmd := cmd.Server
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
