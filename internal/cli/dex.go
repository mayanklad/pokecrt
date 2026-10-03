package cli

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/mayanklad/pokecrt/internal/catalog"
	"github.com/mayanklad/pokecrt/internal/render"
	"github.com/mayanklad/pokecrt/internal/sprite"
	"github.com/mayanklad/pokecrt/internal/storage"
	"github.com/mayanklad/pokecrt/internal/trainer"
)

const dexHelp = `Browse the active trainer's Pokédex

Usage:
  pokecrt dex
  pokecrt dex list [filters]
  pokecrt dex show [entry and appearance selectors]

Commands:
  list        List discovered Pokémon and anonymous National slots
  show        View one entry and an exact collected appearance

Options:
  --help, -h  Show this help

The summary shows eligible species/variant completion, encounter count,
shiny collections and generation progress for the active trainer.
Run 'pokecrt dex list --help' or 'pokecrt dex show --help' for command options.
Browsing requires an active trainer and never records encounters or rewards.
`

const dexListHelp = `List the active trainer's Pokédex

Usage:
  pokecrt dex list [filters]

Filters:
  --seen         Include discovered species only
  --unseen       Include undiscovered slots only; cannot combine with --seen
  --gen          Introduction generations, comma-separated
  --type         Require ALL listed types on one encountered form
  --type-any     Require ANY listed types on that same encountered form
  --color        Species Pokédex colors, comma-separated
  --stage        Evolution stages, comma-separated
  --legendary    Require source legendary status
  --mythical     Require source mythical status
  --baby         Require source baby status

Options:
  --help, -h     Show this help

Results are in National number order. Undiscovered identities remain anonymous.
Metadata filters include discovered species only and cannot accompany --unseen.
Generation filtering may include anonymous undiscovered slots. Categories combine
with AND; values in gen/color/stage combine with OR. Boolean flags accept =true
or =false; false flags impose no restriction.
Browsing requires an active trainer and never records encounters or rewards.
`

const dexShowHelp = `View one entry in the active trainer's Pokédex

Usage:
  pokecrt dex show (--name NAME | --number N) [appearance selectors]

Entry selectors (exactly one required):
  --name         Exact species name or generated alias
  --number       National Pokédex number

Appearance selectors:
  --form         Exact form slug (default: standard)
  --gender       Distinct visual gender: male or female (default: form default)
  --shiny        Select shiny instead of regular artwork

Options:
  --help, -h     Show this help

Undiscovered entries reveal only their number and a locked notice. Discovered
entries reveal species facts, encountered forms and observed genders; evolution
nodes remain anonymous until discovered. Artwork requires the exact collected
appearance. Uncollected or unavailable artwork has no fallback. Boolean flags
accept =true or =false.
Browsing requires an active trainer and never records encounters or rewards.
`

type dexOptions struct {
	action    string
	help      bool
	filter    trainer.DexFilter
	selection catalog.Selection
	number    int
}

func parseDexFlags(args []string) (dexOptions, error) {
	o := dexOptions{action: "summary"}
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		o.action, args = args[0], args[1:]
		if o.action != "list" && o.action != "show" {
			return o, fmt.Errorf("unknown dex subcommand %q", o.action)
		}
	}
	fs := flag.NewFlagSet("dex", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.BoolVar(&o.help, "help", false, "Show help")
	trackFlag(fs, "help")
	fs.Var(fs.Lookup("help").Value, "h", "Show help")
	v := selectorValues{}
	if o.action == "list" {
		fs.BoolVar(&o.filter.Seen, "seen", false, "")
		fs.BoolVar(&o.filter.Unseen, "unseen", false, "")
		fs.StringVar(&v.gen, "gen", "", "")
		fs.StringVar(&v.types, "type", "", "")
		fs.StringVar(&v.typeAny, "type-any", "", "")
		fs.StringVar(&v.color, "color", "", "")
		fs.StringVar(&v.stage, "stage", "", "")
		fs.BoolVar(&v.selection.Legendary, "legendary", false, "")
		fs.BoolVar(&v.selection.Mythical, "mythical", false, "")
		fs.BoolVar(&v.selection.Baby, "baby", false, "")
		for _, n := range []string{"seen", "unseen", "gen", "type", "type-any", "color", "stage", "legendary", "mythical", "baby"} {
			trackFlag(fs, n)
		}
	} else if o.action == "show" {
		fs.StringVar(&v.selection.Name, "name", "", "")
		fs.IntVar(&o.number, "number", 0, "")
		fs.StringVar(&v.selection.Form, "form", "standard", "")
		fs.StringVar(&v.selection.Gender, "gender", "", "")
		fs.BoolVar(&v.selection.Shiny, "shiny", false, "")
		for _, n := range []string{"name", "number", "form", "gender", "shiny"} {
			trackFlag(fs, n)
		}
	}
	if err := parseFlags(fs, args); err != nil {
		return o, err
	}
	var err error
	o.selection, err = v.validate()
	if err != nil {
		return o, err
	}
	if o.action == "list" {
		o.filter.Selection = o.selection
		if o.filter.Seen && o.filter.Unseen {
			return o, fmt.Errorf("--seen and --unseen cannot be combined")
		}
		if o.filter.Unseen && o.filter.Metadata() {
			return o, fmt.Errorf("--unseen cannot be combined with metadata filters")
		}
	}
	if o.action == "show" && !o.help {
		named := o.selection.Name != ""
		numbered := fs.Lookup("number").Value.(*onceValue).seen
		if named == numbered {
			return o, fmt.Errorf("dex show requires exactly one of --name or --number")
		}
		if numbered {
			if _, ok := catalog.ByNumber(o.number); !ok {
				return o, fmt.Errorf("unknown National number %d", o.number)
			}
		} else {
			s, _ := catalog.ByName(o.selection.Name)
			o.number = s.ID
		}
	}
	return o, nil
}
func runDex(args []string, stdout, stderr io.Writer) int {
	o, err := parseDexFlags(args)
	if err != nil {
		return invocationError(stderr, err)
	}
	if o.help {
		help := dexHelp
		if o.action == "list" {
			help = dexListHelp
		}
		if o.action == "show" {
			help = dexShowHelp
		}
		return writeOutput(stdout, stderr, []byte(help))
	}
	path, err := storage.ResolvePath()
	if err != nil {
		if errors.Is(err, storage.ErrInvalidPath) {
			return invocationError(stderr, err)
		}
		return operationalError(stderr, err)
	}
	ctx := context.Background()
	repo, err := storage.ReadOnly(ctx, path)
	if errors.Is(err, storage.ErrNoState) {
		return operationalError(stderr, fmt.Errorf("no trainer profiles; create one with 'pokecrt trainer create <name>'"))
	}
	if err != nil {
		return operationalError(stderr, err)
	}
	defer repo.Close()
	p, err := repo.ActiveProfile(ctx)
	if errors.Is(err, trainer.ErrNoActive) {
		return operationalError(stderr, fmt.Errorf("no active trainer; select one with 'pokecrt trainer use <name>'"))
	}
	if err != nil {
		return operationalError(stderr, err)
	}
	records, err := repo.DexRecords(ctx, p.ID)
	if err != nil {
		return operationalError(stderr, err)
	}
	d := trainer.NewDex(catalog.All(), records, func(k catalog.VariantKey) bool { _, ok := sprite.Lookup(k); return ok })
	var b bytes.Buffer
	switch o.action {
	case "summary":
		s := d.Summary()
		fmt.Fprintf(&b, "Pokédex: %s\nSpecies: %d / %d\nVariants: %d / %d\nEncounters: %d\nShiny collections: %d\n\nGeneration progress:\n", p.Name, s.Completion.Species, s.Completion.SpeciesTotal, s.Completion.Variants, s.Completion.VariantsTotal, s.Encounters, s.ShinyCollections)
		for _, g := range s.Generations {
			fmt.Fprintf(&b, "  Generation %d: %d / %d\n", g.Generation, g.Seen, g.Total)
		}
	case "list":
		rows := d.List(o.filter)
		if len(rows) == 0 {
			b.WriteString("No Pokémon match the specified filters.\n")
		} else {
			for _, r := range rows {
				status := "UNDISCOVERED"
				if r.Seen {
					status = "Discovered"
				}
				if !r.Eligible {
					status += " · Encounter unavailable"
				}
				fmt.Fprintf(&b, "#%03d %s - %s\n", r.Number, r.Name, status)
			}
		}
	case "show":
		e, err := d.Entry(o.number, o.selection)
		if err != nil {
			return operationalError(stderr, err)
		}
		if !e.Seen {
			fmt.Fprintf(&b, "#%03d ????? - UNDISCOVERED\n\n%s\n", e.Number, e.Notice)
			break
		}
		fmt.Fprintf(&b, "#%03d %s\n", e.Number, e.Name)
		if e.ArtworkKey != nil {
			pixels, err := sprite.Decode(*e.ArtworkKey)
			if err != nil {
				return operationalError(stderr, err)
			}
			art, err := render.Render(pixels, os.Getenv("NO_COLOR") == "")
			if err != nil {
				return operationalError(stderr, err)
			}
			b.WriteByte('\n')
			b.Write(art)
		}
		if e.SelectedName != "" {
			fmt.Fprintf(&b, "\nAppearance: %s\nTypes: %s\n", e.SelectedName, strings.Join(e.SelectedTypes, " / "))
			formatDexDiscovery(&b, "Variant", e.Selected)
		}
		if e.Notice != "" {
			fmt.Fprintf(&b, "\n%s\n", e.Notice)
		}
		fmt.Fprintf(&b, "\nGeneration: %d\nColor: %s\nEvolution stage: %d\nBaby: %s\nLegendary: %s\nMythical: %s\n", e.Generation, e.Color, e.Stage, dexYes(e.Baby), dexYes(e.Legendary), dexYes(e.Mythical))
		formatDexDiscovery(&b, "Species", e.Discovery)
		b.WriteString("\nKnown forms:\n")
		for _, f := range e.Forms {
			fmt.Fprintf(&b, "  %s: %d encounters · %s\n", f.Name, f.Count, strings.Join(f.Types, " / "))
			fmt.Fprintf(&b, "    Undiscovered gender slots: %d\n", f.UnknownGenders)
			for _, g := range f.Genders {
				fmt.Fprintf(&b, "    %s: Regular %s; Shiny %s\n", g.Name, dexCollected(g.Regular), dexCollected(g.Shiny))
			}
		}
		fmt.Fprintf(&b, "Undiscovered collectible forms: %d\n\nEvolution family:\n", e.UnknownForms)
		for _, n := range e.Evolution {
			fmt.Fprintf(&b, "  #%03d %s", n.Number, n.Name)
			if len(n.Children) > 0 {
				b.WriteString(" → ")
				for i, c := range n.Children {
					if i > 0 {
						b.WriteString(", ")
					}
					name := "?????"
					for _, node := range e.Evolution {
						if node.Number == c {
							name = node.Name
						}
					}
					fmt.Fprintf(&b, "#%03d %s", c, name)
				}
			}
			b.WriteByte('\n')
		}
	}
	return writeOutput(stdout, stderr, b.Bytes())
}
func dexYes(v bool) string {
	if v {
		return "Yes"
	}
	return "No"
}
func dexCollected(v bool) string {
	if v {
		return "Collected"
	}
	return "Uncollected"
}
func formatDexDiscovery(b *bytes.Buffer, label string, d trainer.Discovery) {
	fmt.Fprintf(b, "%s encounters: %d\nFirst seen: %s\nLast seen: %s\n", label, d.Count, time.UnixMilli(d.FirstMS).UTC().Format(time.RFC3339Nano), time.UnixMilli(d.LastMS).UTC().Format(time.RFC3339Nano))
}
