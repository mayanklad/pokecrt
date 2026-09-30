package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/mayanklad/pokecrt/internal/catalog"
	"github.com/mayanklad/pokecrt/internal/render"
	"github.com/mayanklad/pokecrt/internal/sprite"
)

const printHelp = `Print offline Pokémon artwork

Usage:
  pokecrt print [--name <species>] [--output compact|sprite]

Options:
  --name       Exact species name or generated alias; omission selects randomly
  --output     compact (default) or sprite
  --help, -h   Show this help

This increment supports standard regular artwork. More selectors arrive in D07.
`

type printOptions struct {
	name, output string
	help         bool
}

func parsePrintFlags(args []string) (printOptions, error) {
	options := printOptions{output: "compact"}
	seen := map[string]bool{}
	for i := 0; i < len(args); i++ {
		argument := args[i]
		spelling, value, equals := strings.Cut(argument, "=")
		var name string
		switch spelling {
		case "--name":
			name = "name"
		case "--output":
			name = "output"
		case "--help", "-h":
			name = "help"
		default:
			return options, fmt.Errorf("unknown flag or positional argument %q", argument)
		}
		if seen[name] {
			return options, fmt.Errorf("flag --%s may appear only once", name)
		}
		seen[name] = true
		if name == "help" {
			options.help = true
			if equals {
				parsed, err := strconv.ParseBool(value)
				if err != nil {
					return options, fmt.Errorf("invalid --help value %q", value)
				}
				options.help = parsed
			}
			continue
		}
		if !equals {
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				return options, fmt.Errorf("flag --%s requires a value", name)
			}
			i++
			value = args[i]
		}
		value = strings.TrimSpace(value)
		if value == "" {
			return options, fmt.Errorf("flag --%s cannot be empty", name)
		}
		if name == "name" {
			options.name = value
		} else {
			options.output = strings.ToLower(value)
		}
	}
	if options.output != "compact" && options.output != "sprite" {
		return options, fmt.Errorf("unsupported print output %q", options.output)
	}
	return options, nil
}

func runPrint(args []string, stdout, stderr io.Writer, selectIndex catalog.IndexSelector) int {
	options, err := parsePrintFlags(args)
	if err != nil {
		return invocationError(stderr, err)
	}
	if options.help {
		return writeOutput(stdout, stderr, []byte(printHelp))
	}
	var species catalog.Species
	var key catalog.VariantKey
	if options.name != "" {
		var ok bool
		species, ok = catalog.ByName(options.name)
		if !ok {
			return invocationError(stderr, fmt.Errorf("unknown species %q", options.name))
		}
		key, ok = catalog.StandardKey(species)
		if !ok {
			return operationalError(stderr, fmt.Errorf("standard artwork unavailable for %s", species.Name))
		}
	} else {
		species, key, err = catalog.ChooseStandard(selectIndex, func(key catalog.VariantKey) bool { _, ok := sprite.Lookup(key); return ok })
		if err != nil {
			return operationalError(stderr, err)
		}
	}
	pixels, err := sprite.Decode(key)
	if err != nil {
		return operationalError(stderr, err)
	}
	artwork, err := render.Render(pixels, os.Getenv("NO_COLOR") == "")
	if err != nil {
		return operationalError(stderr, err)
	}
	var output bytes.Buffer
	output.Write(artwork)
	if options.output == "compact" {
		fmt.Fprintf(&output, "\n#%03d %s\n", species.ID, species.Name)
	}
	return writeOutput(stdout, stderr, output.Bytes())
}

func operationalError(stderr io.Writer, err error) int {
	fmt.Fprintf(stderr, "pokecrt: %v\n", err)
	return 1
}
