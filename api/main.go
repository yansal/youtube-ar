package main

import (
	"context"
	"fmt"
	"os"

	"github.com/yansal/youtube-ar/api/cmd"
)

func usage() string {
	// TODO: generate automatically from commands in package cmd
	return `usage: youtube-ar [create-url|create-urls-from-playlist|download-url|list-logs|list-urls|retry-next|server|worker]`
}

func main() {
	ctx := context.Background()
	if len(os.Args) == 1 {
		if err := cmd.Server(ctx, os.Args[1:]); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	cmds := map[string]cmd.Cmd{
		"create-url":                cmd.CreateURL,
		"create-urls-from-playlist": cmd.CreateURLsFromPlaylist,
		"download-url":              cmd.DownloadURL,
		"list-logs":                 cmd.ListLogs,
		"list-urls":                 cmd.ListURLs,
		"retry-next":                cmd.RetryNext,
		"server":                    cmd.Server,
		"worker":                    cmd.Worker,
	}

	cmd, ok := cmds[os.Args[1]]
	if !ok {
		fmt.Printf("error: unknown cmd %s\n", os.Args[1])
		fmt.Println(usage())
		os.Exit(2)
	}

	if err := cmd(ctx, os.Args[2:]); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
