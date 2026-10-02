package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/mayanklad/pokecrt/internal/storage"
	"github.com/mayanklad/pokecrt/internal/trainer"
)

const trainerHelp = `Manage local trainer profiles

Usage:
  pokecrt trainer [--name <trainer-name>]
  pokecrt trainer create <trainer-name>
  pokecrt trainer list
  pokecrt trainer use <trainer-name>

Options:
  --name       View another profile without changing the active trainer
  --help, -h   Show this help

Names use 1–32 Unicode code points after trimming. Internal spaces are allowed;
quote them in the shell. NFC-normalized, case-folded names identify one profile.
Only the first creation activates automatically. Listing and viewing create no
storage. Profile commands never record encounters or discoveries.
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
		if o.action != "create" && o.action != "list" && o.action != "use" {
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
			if o.action != "list" {
				output += " <trainer-name>"
			}
			output += "\n\nOptions:\n  --help, -h   Show this help\n\nFlags precede the name. Quote names containing spaces.\n"
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
		output = fmt.Sprintf("Trainer: %s\nCreated: %s\nActive: %s\n", p.Name, time.UnixMilli(p.CreatedAtMS).UTC().Format(time.RFC3339Nano), active)
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
