package cli

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/mayanklad/pokecrt/internal/tui"
)

const tuiHelp = `Open the interactive Adventure Menu

Usage:
  pokecrt tui [--appearance dark|light|follow-terminal|terminal-native]

Options:
  --appearance  Starting appearance (default: saved choice, then follow-terminal)
  --help, -h    Show this help without opening the interface or trainer data

Arrows and Tab/Shift+Tab move focus; Enter activates the focused control.
Click controls or use the mouse wheel to move focus. A opens appearance;
Esc closes appearance or quits at the main menu. Q and Ctrl+C quit.

Appearance changes apply live. Save appearance remembers the choice. Follow Terminal uses supported
background replies, with default terminal colors as its fallback. Terminal Native
preserves the terminal's default background and configured transparency.
NO_COLOR keeps the interface uncolored. A usable terminal is required on both
stdin and stdout; redirected streams are rejected before any terminal changes.

Trainer setup and selection are interactive. Mouse-only name entry uses an
on-screen keyboard. Additional trainers need a separate Use action. Collection
and gameplay remain available through the CLI; interactive section views follow.
`

func runTUI(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("tui", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var help bool
	fs.BoolVar(&help, "help", false, "Show help")
	trackFlag(fs, "help")
	fs.Var(fs.Lookup("help").Value, "h", "Show help")
	raw := string(tui.FollowTerminal)
	fs.StringVar(&raw, "appearance", raw, "Starting appearance")
	trackFlag(fs, "appearance")
	if err := parseFlags(fs, args); err != nil {
		return invocationError(stderr, err)
	}
	appearance, err := tui.ParseAppearance(raw)
	if err != nil {
		return invocationError(stderr, err)
	}
	if help {
		return writeOutput(stdout, stderr, []byte(tuiHelp))
	}
	explicit := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "appearance" {
			explicit = true
		}
	})
	if !explicit {
		appearance = ""
	}
	if err = tui.Run(os.Stdin, stdout, appearance); err != nil {
		fmt.Fprintf(stderr, "pokecrt: %v\n", err)
		return 1
	}
	return 0
}
