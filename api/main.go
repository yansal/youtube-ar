package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/yansal/youtube-ar/api/cmd"
)

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

	cmd, ok := cmds[os.Args[1]]
	if !ok {
		fmt.Printf("unknown cmd %s\n", os.Args[1])
		fmt.Printf("usage: %s [%s]\n", os.Args[0], strings.Join(names, "|"))
		os.Exit(2)
	}

	if err := cmd(ctx, os.Args[2:]); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
