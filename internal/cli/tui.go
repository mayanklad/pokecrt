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
  --appearance  Starting appearance (default: follow-terminal)
  --help, -h    Show this help without opening the interface or trainer data

Arrows and Tab/Shift+Tab move focus; Enter activates the focused control.
Click controls or use the mouse wheel to move focus. A opens appearance;
Esc closes appearance or quits at the main menu. Q and Ctrl+C quit.

Appearance changes apply live for this session. Follow Terminal uses supported
background replies, with default terminal colors as its fallback. Terminal Native
preserves the terminal's default background and configured transparency.
NO_COLOR keeps the interface uncolored. A usable terminal is required on both
stdin and stdout; redirected streams are rejected before any terminal changes.

This first interface increment provides navigation, appearance and read-only
trainer status. Collection and gameplay remain available through the CLI;
interactive setup and section views are added in subsequent increments.
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
	if err = tui.Run(os.Stdin, stdout, appearance); err != nil {
		fmt.Fprintf(stderr, "pokecrt: %v\n", err)
		return 1
	}
	return 0
}
