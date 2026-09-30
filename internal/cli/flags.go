package cli

import (
	"flag"
	"fmt"
	"io"
	"strings"
)

type rootOptions struct {
	help    bool
	version bool
}

func parseRootFlags(args []string) (rootOptions, error) {
	var options rootOptions

	flags := flag.NewFlagSet("pokecrt", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.BoolVar(&options.help, "help", false, "Show help")
	flags.BoolVar(&options.help, "h", false, "Show help")
	flags.BoolVar(&options.version, "version", false, "Show version")

	seen := make(map[string]bool)

	for _, argument := range args {
		if argument == "--" {
			break
		}
		if !strings.HasPrefix(argument, "-") {
			break
		}

		spelling, _, _ := strings.Cut(argument, "=")

		var name string
		switch spelling {
		case "--help", "-h":
			name = "help"
		case "--version":
			name = "version"
		default:
			return options, fmt.Errorf("unknown flag %q", spelling)
		}

		if seen[name] {
			return options, fmt.Errorf("flag --%s may appear only once", name)
		}
		seen[name] = true
	}

	if err := flags.Parse(args); err != nil {
		return options, err
	}
	if flags.NArg() != 0 {
		return options, fmt.Errorf(
			"unexpected positional argument %q",
			flags.Arg(0),
		)
	}

	return options, nil
}
