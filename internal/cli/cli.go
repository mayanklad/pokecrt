package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"syscall"

	"github.com/mayanklad/pokecrt/internal/catalog"
)

const rootHelp = `PokéCRT - offline Pokémon terminal artwork

Usage:
  pokecrt
  pokecrt --help
  pokecrt --version
  pokecrt print [selectors] [--output compact|sprite]
  pokecrt list [selectors] [--details]
  pokecrt encounter [--output full|compact|no-title|achievements|sprite]
  pokecrt dex [list|show]
  pokecrt trainer [--name <trainer-name>]
  pokecrt trainer create <trainer-name>
  pokecrt trainer list
  pokecrt trainer use <trainer-name>
  pokecrt trainer achievements

Options:
  --help, -h   Show this help
  --version    Show application version and dataset identity

Commands:
  print       Print named or uniformly random matching artwork
  list        List the public catalog or detailed entries
  encounter   Record one unrestricted encounter for the active trainer
  dex         Browse the active trainer’s discoveries
  trainer     Manage profiles, view statistics and achievement progress

Run 'pokecrt <command> --help' for command options.
`

// Run executes one invocation and returns its process exit status.
// Public commands bypass trainer path resolution and storage entirely.
func Run(args []string, stdout, stderr io.Writer, version, datasetID string) int {
	if len(args) > 0 && args[0] == "print" {
		return runPrint(args[1:], stdout, stderr, catalog.CryptoIndex)
	}
	if len(args) > 0 && args[0] == "list" {
		return runList(args[1:], stdout, stderr)
	}
	if len(args) > 0 && args[0] == "encounter" {
		return runEncounter(args[1:], stdout, stderr)
	}
	if len(args) > 0 && args[0] == "dex" {
		return runDex(args[1:], stdout, stderr)
	}
	if len(args) > 0 && args[0] == "trainer" {
		return runTrainer(args[1:], stdout, stderr)
	}
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		return invocationError(stderr, fmt.Errorf("unknown command %q", args[0]))
	}

	options, err := parseRootFlags(args)
	if err != nil {
		return invocationError(stderr, err)
	}

	if options.help && options.version {
		return invocationError(stderr, errors.New(
			"--help and --version cannot be combined",
		))
	}

	output := rootHelp
	if options.version {
		output = fmt.Sprintf("pokecrt %s\ndataset: %s\n", version, datasetID)
	}

	return writeOutput(stdout, stderr, []byte(output))
}

func writeOutput(stdout, stderr io.Writer, output []byte) int {
	n, err := stdout.Write(output)
	if err == nil && n != len(output) {
		err = io.ErrShortWrite
	}
	if err == nil || errors.Is(err, syscall.EPIPE) {
		return 0
	}

	fmt.Fprintf(stderr, "pokecrt: write output: %v\n", err)
	return 1
}

func invocationError(stderr io.Writer, err error) int {
	fmt.Fprintf(stderr, "pokecrt: %v\nRun 'pokecrt --help' for usage.\n", err)
	return 2
}
