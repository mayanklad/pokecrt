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
	generate := flags.Bool("generate", false, "Generate normalized catalog, assets, and coverage")
	check := flags.Bool("check", false, "Verify pinned inputs and, with mappings/out, generated output drift")
	sourcesPath := flags.String("sources", "", "Explicit source lock file path")
	cachePath := flags.String("cache", "", "Explicit download cache directory")
	mappingsPath := flags.String("mappings", "", "Explicit normalization mapping file path")
	outPath := flags.String("out", "", "Explicit project output root")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 || (!*fetch && !*generate && !*check) || *sourcesPath == "" || *cachePath == "" {
		fmt.Fprintln(stderr, "Usage: go run ./tools/dataset --sources <file> --cache <dir> [--mappings <file> --out <root>] --fetch|--generate|--check")
		return 2
	}
	if (*mappingsPath == "") != (*outPath == "") || (*generate && *mappingsPath == "") || (*mappingsPath != "" && !*generate && !*check) {
		fmt.Fprintln(stderr, "dataset: --mappings and --out must be provided together for --generate or generated --check")
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
	if *mappingsPath != "" {
		mappings, err := readMappings(*mappingsPath)
		if err != nil {
			fmt.Fprintf(stderr, "dataset: %v\n", err)
			return 1
		}
		bundle, err := buildBundle(lock, *cachePath, mappings)
		if err == nil && *generate {
			err = writeBundle(bundle, *outPath)
		}
		if err == nil && *check {
			err = checkBundle(bundle, *outPath)
		}
		if err != nil {
			fmt.Fprintf(stderr, "dataset: %v\n", err)
			return 1
		}
		if _, err := fmt.Fprintf(stdout, "Dataset: %s\nCatalog species: %d\nStandard regular sprites: %d\n", bundle.DatasetID, bundle.Coverage.CatalogSpecies, bundle.Coverage.StandardRegularSprites); err != nil {
			fmt.Fprintf(stderr, "dataset: write output: %v\n", err)
			return 1
		}
	}
	if _, err := fmt.Fprintf(stdout, "Verified %d pinned inputs.\n", count); err != nil {
		fmt.Fprintf(stderr, "dataset: write output: %v\n", err)
		return 1
	}
	return 0
}
