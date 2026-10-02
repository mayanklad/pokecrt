package cli

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/mayanklad/pokecrt/internal/catalog"
	"github.com/mayanklad/pokecrt/internal/render"
	"github.com/mayanklad/pokecrt/internal/sprite"
)

const listHelp = `List the public Pokémon catalog

Usage:
  pokecrt list [selectors] [--details]

Selectors:
  --name         Exact species name or generated alias
  --gen          Introduction generations, comma-separated
  --type         Require ALL listed selected-form types
  --type-any     Require ANY listed selected-form types
  --color        Species Pokédex colors, comma-separated
  --stage        Evolution stages, comma-separated
  --form         Exact form slug (default: standard)
  --gender       Distinct visual gender: male or female
  --shiny        Require actual shiny artwork (default: regular)
  --legendary    Require source legendary status
  --mythical     Require source mythical status
  --baby         Require source baby status

Options:
  --details      Metadata, evolution family, form/artwork inventory and one sprite
  --help, -h     Show this help

Selectors follow print's AND/OR rules. Results are in National number order.
Regular metadata entries remain visible when artwork is unavailable. Valid empty
queries succeed. No trainer, discovery state, random selection or pager is used.
`

type listOptions struct {
	selection     catalog.Selection
	details, help bool
}

func parseListFlags(args []string) (listOptions, error) {
	var o listOptions
	values := selectorValues{}
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	values.register(fs)
	fs.BoolVar(&o.details, "details", false, "Show detailed entries")
	trackFlag(fs, "details")
	fs.BoolVar(&o.help, "help", false, "Show help")
	trackFlag(fs, "help")
	fs.Var(fs.Lookup("help").Value, "h", "Show help")
	if err := parseFlags(fs, args); err != nil {
		return o, err
	}
	var err error
	o.selection, err = values.validate()
	return o, err
}

func runList(args []string, stdout, stderr io.Writer) int {
	o, err := parseListFlags(args)
	if err != nil {
		return invocationError(stderr, err)
	}
	if o.help {
		return writeOutput(stdout, stderr, []byte(listHelp))
	}
	matches, err := catalog.Query(o.selection, func(k catalog.VariantKey) bool { _, ok := sprite.Lookup(k); return ok })
	if err != nil {
		return invocationError(stderr, err)
	}
	if len(matches) == 0 {
		return writeOutput(stdout, stderr, []byte("No Pokémon match the specified filters.\n"))
	}
	var output bytes.Buffer
	if !o.details {
		table := tabwriter.NewWriter(&output, 0, 0, 2, ' ', 0)
		fmt.Fprintln(table, "NUMBER\tNAME\tGEN\tTYPE\tARTWORK")
		for _, m := range matches {
			availability := "Unavailable"
			if m.Available {
				availability = "Available"
			}
			fmt.Fprintf(table, "#%03d\t%s\t%d\t%s\t%s\n", m.Species.ID, variantLabel(m, o.selection.Gender != ""), m.Species.Generation, typeLabel(m.Form.Types), availability)
		}
		if err := table.Flush(); err != nil {
			return operationalError(stderr, err)
		}
	} else {
		for i, m := range matches {
			if i > 0 {
				output.WriteString("\n---\n\n")
			}
			if err := writeCatalogDetails(&output, m, o.selection.Gender != ""); err != nil {
				return operationalError(stderr, err)
			}
		}
	}
	return writeOutput(stdout, stderr, output.Bytes())
}

func titleWord(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
func typeLabel(types []string) string {
	result := make([]string, len(types))
	for i, v := range types {
		result[i] = titleWord(v)
	}
	return strings.Join(result, " / ")
}
func yesNo(v bool) string {
	if v {
		return "Yes"
	}
	return "No"
}

func writeCatalogDetails(output *bytes.Buffer, m catalog.Match, explicitGender bool) error {
	fmt.Fprintf(output, "#%03d %s\n\n", m.Species.ID, variantLabel(m, explicitGender))
	if m.Available {
		pixels, err := sprite.Decode(m.Key)
		if err != nil {
			return err
		}
		artwork, err := render.Render(pixels, os.Getenv("NO_COLOR") == "")
		if err != nil {
			return err
		}
		output.Write(artwork)
	} else {
		output.WriteString("Artwork unavailable for this selection.\n")
	}
	s := m.Species
	fmt.Fprintf(output, "\nGeneration: %d\nTypes: %s\nColor: %s\nEvolution stage: %d\nBaby: %s\nLegendary: %s\nMythical: %s\n", s.Generation, typeLabel(m.Form.Types), titleWord(s.Color), s.Stage, yesNo(s.Baby), yesNo(s.Legendary), yesNo(s.Mythical))
	output.WriteString("\nEvolution family:\n")
	if err := writeEvolutionFamily(output, s); err != nil {
		return err
	}
	output.WriteString("\nForms and artwork:\n")
	for _, f := range s.Forms {
		slots := []string{}
		for _, gender := range f.Genders {
			palettes := []string{}
			for _, palette := range []string{"regular", "shiny"} {
				if _, ok := sprite.Lookup(catalog.VariantKey{SpeciesID: s.ID, FormID: f.ID, Gender: gender, Palette: palette}); ok {
					palettes = append(palettes, titleWord(palette))
				}
			}
			available := strings.Join(palettes, ", ")
			if len(palettes) == 0 {
				available = "Unavailable"
			}
			if gender != "default" {
				label := titleWord(gender)
				if gender == f.DefaultGender {
					label += " (default)"
				}
				available = label + ": " + available
			}
			slots = append(slots, available)
		}
		fmt.Fprintf(output, "  %s [%s]: %s\n", f.Name, f.ID, strings.Join(slots, "; "))
	}
	return nil
}

// Public species relationships include the complete connected family, branches
// and later-generation relatives, independently of selected form or artwork.
func writeEvolutionFamily(output *bytes.Buffer, start catalog.Species) error {
	family := map[int]catalog.Species{start.ID: start}
	queue := []int{start.ID}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		s := family[id]
		related := append([]int{}, s.EvolvesTo...)
		if s.EvolvesFrom > 0 {
			related = append(related, s.EvolvesFrom)
		}
		for _, next := range related {
			if _, seen := family[next]; seen {
				continue
			}
			species, ok := catalog.ByNumber(next)
			if !ok {
				return fmt.Errorf("evolution relationship references missing species #%03d", next)
			}
			family[next] = species
			queue = append(queue, next)
		}
	}
	if len(family) == 1 {
		output.WriteString("  No evolution relationships.\n")
		return nil
	}
	ids := make([]int, 0, len(family))
	for id := range family {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		s := family[id]
		children := []string{}
		for _, child := range s.EvolvesTo {
			next := family[child]
			children = append(children, fmt.Sprintf("#%03d %s", next.ID, next.Name))
		}
		next := strings.Join(children, ", ")
		if len(children) == 0 {
			next = "No further evolution"
		}
		fmt.Fprintf(output, "  #%03d %s → %s\n", s.ID, s.Name, next)
	}
	return nil
}
