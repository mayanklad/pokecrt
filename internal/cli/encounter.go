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

	"github.com/mayanklad/pokecrt/internal/render"
	"github.com/mayanklad/pokecrt/internal/sprite"
	"github.com/mayanklad/pokecrt/internal/storage"
	"github.com/mayanklad/pokecrt/internal/trainer"
)

const encounterHelp = `Encounter a Pokémon with the active trainer

Usage:
  pokecrt encounter [--output full|compact|no-title|achievements|sprite]

Options:
  --output    full (default), compact, no-title, achievements or sprite
  --help, -h  Show this help

Modes:
  full          Artwork, heading, selected-form types, progress and new achievements
  compact       Artwork and selected appearance heading
  no-title      Artwork, progress and new achievements
  achievements  Artwork and newly earned achievements; artwork alone if none
  sprite        Artwork alone

Requires an active trainer. Species, supported forms and visual genders are
selected uniformly in separate steps. An exact appearance with shiny artwork
has a 1/4096 shiny chance. Filters and appearance selectors are not accepted.
NO_COLOR disables artwork color. Every mode records the same encounter, XP,
discoveries and unlocks. Encounters are saved before displaying output. If output fails, the encounter
remains saved and is not repeated. Closed output pipes exit quietly.
`

type encounterOptions struct {
	output string
	help   bool
}

func parseEncounterFlags(args []string) (encounterOptions, error) {
	o := encounterOptions{output: "full"}
	fs := flag.NewFlagSet("encounter", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&o.output, "output", "full", "Output mode")
	trackFlag(fs, "output")
	fs.BoolVar(&o.help, "help", false, "Show help")
	trackFlag(fs, "help")
	fs.Var(fs.Lookup("help").Value, "h", "Show help")
	if err := parseFlags(fs, args); err != nil {
		return o, err
	}
	o.output = strings.ToLower(o.output)
	switch o.output {
	case "full", "compact", "no-title", "achievements", "sprite":
		return o, nil
	}
	return o, fmt.Errorf("unsupported encounter output %q", o.output)
}

type encounterRepository interface {
	trainer.EncounterRepository
	ListProfiles(context.Context) ([]trainer.Profile, error)
	Close() error
}
type encounterOpener func(context.Context, string) (encounterRepository, error)
type encounterFactory func(encounterRepository) (trainer.EncounterService, error)

func runEncounter(args []string, stdout, stderr io.Writer) int {
	return executeEncounter(args, stdout, stderr, storage.ResolvePath,
		func(ctx context.Context, path string) (encounterRepository, error) { return storage.Open(ctx, path) },
		func(repo encounterRepository) (trainer.EncounterService, error) {
			return trainer.NewBundledEncounterService(repo, func(choice trainer.Choice) ([]byte, error) {
				pixels, err := sprite.Decode(choice.Key())
				if err != nil {
					return nil, err
				}
				return render.Render(pixels, os.Getenv("NO_COLOR") == "")
			})
		})
}

func executeEncounter(args []string, stdout, stderr io.Writer, resolve func() (string, error), open encounterOpener, factory encounterFactory) int {
	o, err := parseEncounterFlags(args)
	if err != nil {
		return invocationError(stderr, err)
	}
	if o.help {
		return writeOutput(stdout, stderr, []byte(encounterHelp))
	}
	path, err := resolve()
	if err != nil {
		if errors.Is(err, storage.ErrInvalidPath) {
			return invocationError(stderr, err)
		}
		return operationalError(stderr, err)
	}
	ctx := context.Background()
	repo, err := open(ctx, path)
	if errors.Is(err, storage.ErrNoState) {
		fmt.Fprint(stderr, "pokecrt: encounter requires an active trainer.\n", noProfiles)
		return 1
	}
	if err != nil {
		return operationalError(stderr, err)
	}
	defer repo.Close()
	service, err := factory(repo)
	if err != nil {
		return operationalError(stderr, err)
	}
	result, err := service.Encounter(ctx)
	if errors.Is(err, trainer.ErrNoActive) {
		profiles, listErr := repo.ListProfiles(ctx)
		if listErr != nil {
			return operationalError(stderr, listErr)
		}
		fmt.Fprint(stderr, "pokecrt: encounter requires an active trainer.\n", formatProfiles(profiles))
		if len(profiles) > 0 {
			fmt.Fprint(stderr, "Select a trainer:\n  pokecrt trainer use <name>\n")
		}
		return 1
	}
	if err != nil {
		return operationalError(stderr, err)
	}
	return writeOutput(stdout, stderr, formatEncounter(result.Record, result.Artwork(), o.output))
}

// formatEncounter only presents committed values. It cannot record an encounter,
// recalculate awards, evaluate achievements or query trainer state.
func formatEncounter(record trainer.Record, artwork []byte, mode string) []byte {
	var output bytes.Buffer
	output.Write(artwork)
	key, snapshot := record.Choice.Key(), record.Choice.Snapshot()
	if mode == "full" || mode == "compact" {
		label := snapshot.SpeciesName
		if key.FormID != "standard" {
			label += " · " + snapshot.FormName
		}
		if key.Gender != "default" {
			label += " · " + titleWord(key.Gender)
		}
		if key.Palette == "shiny" {
			label += " · Shiny"
		}
		fmt.Fprintf(&output, "\n#%03d %s\n", key.SpeciesID, label)
		if mode == "full" {
			types := []string{snapshot.Type1}
			if snapshot.Type2 != "" {
				types = append(types, snapshot.Type2)
			}
			fmt.Fprintln(&output, typeLabel(types))
		}
	}
	if mode == "full" || mode == "no-title" {
		notice := "REPEAT ENCOUNTER"
		if record.FirstSpecies {
			notice = "NEW SPECIES DISCOVERED"
		} else if record.FirstVariant {
			notice = "NEW VARIANT DISCOVERED"
		}
		fmt.Fprintf(&output, "\n%s\n", notice)
		if key.Palette == "shiny" {
			fmt.Fprintln(&output, "SHINY ENCOUNTER")
			if record.FirstVariant && !record.RegularCollected {
				fmt.Fprintln(&output, "Shiny variant collected; regular variant not yet collected.")
			}
		}
		c := record.Completion
		fmt.Fprintf(&output, "Pokédex: %d / %d\nVariants: %d / %d\nXP gained: %d\n", c.Species, c.SpeciesTotal, c.Variants, c.VariantsTotal, record.XPAwarded)
		if record.After.Level != record.Before.Level {
			fmt.Fprintf(&output, "LEVEL UP - %d → %d\n", record.Before.Level, record.After.Level)
		}
	}
	if mode == "full" || mode == "no-title" || mode == "achievements" {
		for _, unlock := range record.NewUnlocks() {
			fmt.Fprintf(&output, "\n🏆 %s\n%s\n", unlock.Name, unlock.Description)
		}
	}
	return output.Bytes()
}
