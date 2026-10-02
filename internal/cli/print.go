package cli

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mayanklad/pokecrt/internal/catalog"
	"github.com/mayanklad/pokecrt/internal/render"
	"github.com/mayanklad/pokecrt/internal/sprite"
)

const printHelp = `Print offline Pokémon artwork

Usage:
  pokecrt print [selectors] [--output compact|sprite]

Selectors:
  --name         Exact species name or generated alias
  --gen          Introduction generations, comma-separated
  --type         Require ALL listed selected-form types
  --type-any     Require ANY listed selected-form types
  --color        Species Pokédex colors, comma-separated
  --stage        Evolution stages, comma-separated
  --form         Exact form slug (default: standard)
  --gender       Distinct visual gender: male or female
  --shiny        Select shiny artwork (default: regular)
  --legendary    Require source legendary status
  --mythical     Require source mythical status
  --baby         Require source baby status

Options:
  --output       compact (default) or sprite
  --help, -h     Show this help

Categories combine with AND. Values in gen/color/stage combine with OR.
Boolean flags accept =true or =false; false status flags impose no restriction.
Omission selects a matching species uniformly, using the selected form's default
gender. Missing artwork is never replaced with another appearance.
`

type printOptions struct {
	selection catalog.Selection
	output    string
	help      bool
}

func parsePrintFlags(args []string) (printOptions, error) {
	o := printOptions{output: "compact"}
	values := selectorValues{}
	fs := flag.NewFlagSet("print", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	values.register(fs)
	fs.StringVar(&o.output, "output", "compact", "Output mode")
	trackFlag(fs, "output")
	fs.BoolVar(&o.help, "help", false, "Show help")
	trackFlag(fs, "help")
	fs.Var(fs.Lookup("help").Value, "h", "Show help")
	if err := parseFlags(fs, args); err != nil {
		return o, err
	}
	o.output = strings.ToLower(o.output)
	if o.output != "compact" && o.output != "sprite" {
		return o, fmt.Errorf("unsupported print output %q", o.output)
	}
	var err error
	o.selection, err = values.validate()
	return o, err
}

func runPrint(args []string, stdout, stderr io.Writer, selectIndex catalog.IndexSelector) int {
	options, err := parsePrintFlags(args)
	if err != nil {
		return invocationError(stderr, err)
	}
	if options.help {
		return writeOutput(stdout, stderr, []byte(printHelp))
	}
	matches, err := catalog.Query(options.selection, func(key catalog.VariantKey) bool { _, ok := sprite.Lookup(key); return ok })
	if err != nil {
		return invocationError(stderr, err)
	}
	if options.selection.Name != "" {
		if len(matches) == 0 && options.selection.Gender != "" {
			species, _ := catalog.ByName(options.selection.Name)
			for _, form := range species.Forms {
				if form.ID == options.selection.Form && len(form.Genders) < 2 {
					return operationalError(stderr, fmt.Errorf("No Pokémon match the specified filters. The selected form has no distinct visual-gender artwork."))
				}
			}
		}
		if len(matches) == 0 && options.selection.Shiny {
			regular := options.selection
			regular.Shiny = false
			identities, _ := catalog.Query(regular, nil)
			if len(identities) > 0 {
				return operationalError(stderr, fmt.Errorf("No Pokémon match the specified filters. selected shiny artwork unavailable for %s; no fallback applied", identities[0].Species.Name))
			}
		}
		if len(matches) == 1 && !matches[0].Available {
			return operationalError(stderr, fmt.Errorf("No Pokémon match the specified filters. selected artwork unavailable for %s; no fallback applied", matches[0].Species.Name))
		}
	}
	chosen, err := catalog.Choose(matches, selectIndex)
	if err != nil {
		return operationalError(stderr, err)
	}
	pixels, err := sprite.Decode(chosen.Key)
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
		fmt.Fprintf(&output, "\n#%03d %s", chosen.Species.ID, chosen.Species.Name)
		if chosen.Form.ID != "standard" {
			fmt.Fprintf(&output, " · %s", chosen.Form.Name)
		}
		if options.selection.Gender != "" {
			gender := "Male"
			if chosen.Key.Gender == "female" {
				gender = "Female"
			}
			fmt.Fprintf(&output, " · %s", gender)
		}
		if chosen.Key.Palette == "shiny" {
			output.WriteString(" · Shiny")
		}
		output.WriteByte('\n')
	}
	return writeOutput(stdout, stderr, output.Bytes())
}

func operationalError(stderr io.Writer, err error) int {
	fmt.Fprintf(stderr, "pokecrt: %v\n", err)
	return 1
}
