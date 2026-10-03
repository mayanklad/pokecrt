package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/mayanklad/pokecrt/internal/catalog"
	"github.com/mayanklad/pokecrt/internal/sprite"
	"github.com/mayanklad/pokecrt/internal/storage"
	"github.com/mayanklad/pokecrt/internal/trainer"
)

const trainerHelp = `Manage local trainer profiles

Usage:
  pokecrt trainer [--name <trainer-name>]
  pokecrt trainer create <trainer-name>
  pokecrt trainer list
  pokecrt trainer use <trainer-name>
  pokecrt trainer achievements

Options:
  --name       View another profile without changing the active trainer
  --help, -h   Show this help

Names use 1–32 Unicode code points after trimming. Internal spaces are allowed;
quote them in the shell. NFC-normalized, case-folded names identify one profile.
Only the first creation activates automatically. Listing and viewing create no
storage. Profile commands never record encounters or discoveries. Viewing shows XP,
collection statistics and generation progress; achievements lists earned and
locked goals for the active trainer without granting rewards.
`

const noProfiles = "No trainer profiles found.\nCreate your first trainer:\n  pokecrt trainer create <name>\n"

type trainerOptions struct {
	action string
	help   bool
	name   trainer.Name
	named  bool
}

func parseTrainerFlags(args []string) (trainerOptions, error) {
	o := trainerOptions{action: "view"}
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		o.action, args = args[0], args[1:]
		if o.action != "create" && o.action != "list" && o.action != "use" && o.action != "achievements" {
			return o, fmt.Errorf("unknown trainer subcommand %q", o.action)
		}
	}
	fs := flag.NewFlagSet("trainer "+o.action, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.BoolVar(&o.help, "help", false, "Show help")
	trackFlag(fs, "help")
	fs.Var(fs.Lookup("help").Value, "h", "Show help")
	var rawName string
	if o.action == "view" {
		fs.StringVar(&rawName, "name", "", "View a named trainer")
		trackFlag(fs, "name")
	}
	if err := parseFlagSet(fs, args); err != nil {
		return o, err
	}
	if o.action == "create" || o.action == "use" {
		if o.help && fs.NArg() == 0 {
			return o, nil
		}
		if fs.NArg() != 1 {
			return o, fmt.Errorf("trainer %s requires exactly one name; quote names containing spaces", o.action)
		}
		rawName, o.named = fs.Arg(0), true
	} else {
		if fs.NArg() != 0 {
			return o, fmt.Errorf("unexpected positional argument %q", fs.Arg(0))
		}
		o.named = rawName != ""
	}
	if o.named {
		var err error
		o.name, err = trainer.ParseName(rawName)
		if err != nil {
			return o, err
		}
	}
	return o, nil
}

type profileRepository interface {
	trainer.Profiles
	TrainerRecords(context.Context, int64) (trainer.TrainerRecords, error)
	Close() error
}
type profileOpener func(context.Context, string, string) (profileRepository, error)

func openProfiles(ctx context.Context, path, action string) (profileRepository, error) {
	switch action {
	case "create":
		return storage.Initialize(ctx, path)
	case "use":
		return storage.Open(ctx, path)
	default:
		return storage.ReadOnly(ctx, path)
	}
}

func runTrainer(args []string, stdout, stderr io.Writer) int {
	return executeTrainer(args, stdout, stderr, storage.ResolvePath, openProfiles, time.Now)
}

func executeTrainer(args []string, stdout, stderr io.Writer, resolve func() (string, error), open profileOpener, now func() time.Time) int {
	o, err := parseTrainerFlags(args)
	if err != nil {
		return invocationError(stderr, err)
	}
	if o.help {
		output := trainerHelp
		if o.action != "view" {
			output = fmt.Sprintf("%s trainer profiles\n\nUsage:\n  pokecrt trainer %s", strings.ToUpper(o.action[:1])+o.action[1:], o.action)
			if o.action == "create" || o.action == "use" {
				output += " <trainer-name>"
			}
			output += "\n\nOptions:\n  --help, -h   Show this help\n"
			if o.action == "create" || o.action == "use" {
				output += "\nFlags precede the name. Quote names containing spaces.\n"
			}
			if o.action == "achievements" {
				output += "\nRequires an active trainer. Groups unlocked and locked achievements. Earned\ndates are UTC; locked goals show progress. Reading never grants rewards.\n"
			}
		}
		return writeOutput(stdout, stderr, []byte(output))
	}
	path, err := resolve()
	if err != nil {
		if errors.Is(err, storage.ErrInvalidPath) {
			return invocationError(stderr, err)
		}
		return trainerError(stderr, err)
	}
	ctx := context.Background()
	repo, err := open(ctx, path, o.action)
	if errors.Is(err, storage.ErrNoState) {
		if o.action == "list" || (o.action == "view" && !o.named) {
			return writeOutput(stdout, stderr, []byte(noProfiles))
		}
		if o.action == "achievements" {
			return trainerError(stderr, fmt.Errorf("no trainer profiles; create your first trainer with 'pokecrt trainer create <name>'"))
		}
		return trainerError(stderr, fmt.Errorf("%w; create your first trainer with 'pokecrt trainer create <name>'", trainer.ErrNotFound))
	}
	if err != nil {
		return trainerError(stderr, err)
	}
	defer repo.Close()
	var output string
	switch o.action {
	case "create":
		p, err := repo.CreateProfile(ctx, o.name, now().UTC().UnixMilli())
		if err != nil {
			return trainerError(stderr, err)
		}
		output = fmt.Sprintf("Created trainer: %s\n", p.Name)
		if p.Active {
			output += fmt.Sprintf("Active trainer: %s\n", p.Name)
		}
	case "use":
		p, err := repo.UseProfile(ctx, o.name)
		if err != nil {
			return trainerError(stderr, fmt.Errorf("select trainer %q: %w", o.name.Display(), err))
		}
		output = fmt.Sprintf("Active trainer: %s\n", p.Name)
	case "list":
		profiles, err := repo.ListProfiles(ctx)
		if err != nil {
			return trainerError(stderr, err)
		}
		output = formatProfiles(profiles)
	default:
		var p trainer.Profile
		if o.named {
			p, err = repo.FindProfile(ctx, o.name)
		} else {
			p, err = repo.ActiveProfile(ctx)
		}
		if errors.Is(err, trainer.ErrNoActive) {
			profiles, listErr := repo.ListProfiles(ctx)
			if listErr != nil {
				return trainerError(stderr, listErr)
			}
			if len(profiles) == 0 {
				if o.action == "achievements" {
					fmt.Fprint(stderr, "pokecrt: no active trainer.\n", noProfiles)
					return 1
				}
				return writeOutput(stdout, stderr, []byte(noProfiles))
			}
			fmt.Fprint(stderr, "pokecrt: no active trainer.\n", formatProfiles(profiles), "Select a trainer:\n  pokecrt trainer use <name>\n")
			return 1
		}
		if err != nil {
			return trainerError(stderr, err)
		}
		active := "No"
		if p.Active {
			active = "Yes"
		}
		records, readErr := repo.TrainerRecords(ctx, p.ID)
		if readErr != nil {
			return trainerError(stderr, readErr)
		}
		available := func(k catalog.VariantKey) bool { _, ok := sprite.Lookup(k); return ok }
		if o.action == "achievements" {
			targets, err := trainer.NewAchievementTargets(catalog.All(), available)
			if err != nil {
				return trainerError(stderr, err)
			}
			output = formatAchievementViews(p.Name, targets.Views(records.AchievementState, records.Unlocks))
		} else {
			statistics, err := trainer.TrainerStatistics(records, catalog.All(), available)
			if err != nil {
				return trainerError(stderr, err)
			}
			output = fmt.Sprintf("Trainer: %s\nCreated: %s\nActive: %s\n", p.Name, time.UnixMilli(p.CreatedAtMS).UTC().Format(time.RFC3339Nano), active) + formatTrainerStatistics(statistics)
		}
	}
	return writeOutput(stdout, stderr, []byte(output))
}

func formatProfiles(profiles []trainer.Profile) string {
	if len(profiles) == 0 {
		return noProfiles
	}
	var b strings.Builder
	b.WriteString("Trainer profiles:\n")
	for _, p := range profiles {
		fmt.Fprintf(&b, "  %s", p.Name)
		if p.Active {
			b.WriteString(" (active)")
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func trainerError(stderr io.Writer, err error) int {
	fmt.Fprintf(stderr, "pokecrt: %v\n", err)
	return 1
}

func formatTrainerStatistics(s trainer.Statistics) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Level: %d\nTotal XP: %d\nLevel progress: %d / 1000 XP\nXP to next level: %d\nEncounters: %d\nSpecies discovered (historical): %d\nVariants collected (historical): %d\nShiny encounters: %d\nShiny variants collected (historical): %d\n", s.Progress.Level, s.Progress.Total, s.Progress.InLevel, s.Progress.ToNext, s.Encounters, s.Species, s.Variants, s.ShinyEncounters, s.ShinyCollections)
	first, last := "None", "None"
	if s.FirstEncounterMS != nil {
		first = time.UnixMilli(*s.FirstEncounterMS).UTC().Format(time.RFC3339Nano)
	}
	if s.LastEncounterMS != nil {
		last = time.UnixMilli(*s.LastEncounterMS).UTC().Format(time.RFC3339Nano)
	}
	fmt.Fprintf(&b, "First encounter: %s\nLast encounter: %s\n\nCurrent eligible completion:\nSpecies: %d / %d\nVariants: %d / %d\n\nGeneration discoveries:\n", first, last, s.Completion.Species, s.Completion.SpeciesTotal, s.Completion.Variants, s.Completion.VariantsTotal)
	for _, g := range s.Generations {
		fmt.Fprintf(&b, "  Generation %d: %d discovered; eligible %d / %d\n", g.Generation, g.Discovered, g.Eligible, g.EligibleTotal)
	}
	if s.UnclassifiedSpecies > 0 {
		fmt.Fprintf(&b, "  Without current generation metadata: %d discovered species\n", s.UnclassifiedSpecies)
	}
	return b.String()
}
func formatAchievementViews(name string, views trainer.AchievementViews) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Achievements: %s\n\nUnlocked (%d):\n", name, len(views.Unlocked))
	if len(views.Unlocked) == 0 {
		b.WriteString("  None yet.\n")
	}
	for _, v := range views.Unlocked {
		fmt.Fprintf(&b, "  %s\n    %s\n    Earned: %s\n", v.Name, v.Description, time.UnixMilli(v.EarnedAtMS).UTC().Format(time.RFC3339Nano))
		if v.HadTargetAtUnlock {
			fmt.Fprintf(&b, "    Target at unlock: %d\n", v.TargetAtUnlock)
		}
	}
	fmt.Fprintf(&b, "\nLocked (%d):\n", len(views.Locked))
	if len(views.Locked) == 0 {
		b.WriteString("  None.\n")
	}
	for _, v := range views.Locked {
		fmt.Fprintf(&b, "  %s\n    %s\n", v.Name, v.Description)
		if v.HasTarget {
			fmt.Fprintf(&b, "    Progress: %d / %d\n", v.Current, v.Target)
		} else if v.Current >= v.Target && v.Target > 0 {
			b.WriteString("    Requirement met; unlocks on the next committed encounter.\n")
		} else {
			b.WriteString("    Not yet earned.\n")
		}
	}
	return b.String()
}
