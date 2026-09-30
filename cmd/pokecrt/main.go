package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/mayanklad/pokecrt/internal/catalog"
	"github.com/mayanklad/pokecrt/internal/cli"
)

var version = "dev"

func main() {
	// Return quietly when a downstream reader closes its pipe.
	signal.Ignore(syscall.SIGPIPE)

	os.Exit(cli.Run(
		os.Args[1:],
		os.Stdout,
		os.Stderr,
		version,
		catalog.DatasetID,
	))
}
