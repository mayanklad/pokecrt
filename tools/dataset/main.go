package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	os.Exit(run(ctx, os.Args[1:], os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("dataset", flag.ContinueOnError)
	flags.SetOutput(stderr)
	fetch := flags.Bool("fetch", false, "Download and verify pinned source inputs")
	check := flags.Bool("check", false, "Verify cached source inputs without downloading")
	sourcesPath := flags.String("sources", "", "Explicit source lock file path")
	cachePath := flags.String("cache", "", "Explicit download cache directory")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 || (!*fetch && !*check) || *sourcesPath == "" || *cachePath == "" {
		fmt.Fprintln(stderr, "Usage: go run ./tools/dataset --sources <file> --cache <dir> --fetch|--check")
		return 2
	}
	lock, err := readLock(*sourcesPath)
	if err != nil {
		fmt.Fprintf(stderr, "dataset: %v\n", err)
		return 1
	}
	client := &http.Client{Timeout: 30 * time.Second}
	count, err := syncInputs(ctx, lock, *cachePath, *fetch, client)
	if err != nil {
		fmt.Fprintf(stderr, "dataset: %v\n", err)
		return 1
	}
	if _, err := fmt.Fprintf(stdout, "Verified %d pinned inputs.\n", count); err != nil {
		fmt.Fprintf(stderr, "dataset: write output: %v\n", err)
		return 1
	}
	return 0
}
