// Package tui implements the interactive Adventure Menu. Domain work is issued
// as commands; Update and View never perform database or filesystem operations.
package tui

import (
	"context"
	"errors"
	"io"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/term"
	"github.com/mayanklad/pokecrt/internal/storage"
)

type Snapshot struct {
	Name     string
	Profiles int
	Active   bool
}
type Loader func(context.Context) (Snapshot, error)

// LoadProfile opens only an existing current-schema database, reads its profile
// selection and closes it inside the worker. Navigation cannot initialize,
// migrate, select a trainer, or record an encounter.
func LoadProfile(ctx context.Context) (Snapshot, error) {
	path, err := storage.ResolvePath()
	if err != nil {
		return Snapshot{}, err
	}
	repo, err := storage.ReadOnly(ctx, path)
	if errors.Is(err, storage.ErrNoState) {
		return Snapshot{}, nil
	}
	if err != nil {
		return Snapshot{}, err
	}
	defer repo.Close()
	profiles, err := repo.ListProfiles(ctx)
	if err != nil {
		return Snapshot{}, err
	}
	for _, p := range profiles {
		if p.Active {
			return Snapshot{Name: p.Name, Profiles: len(profiles), Active: true}, nil
		}
	}
	return Snapshot{Profiles: len(profiles)}, nil
}

var ErrTerminal = errors.New("the interactive interface requires terminal input and output; run 'pokecrt tui --help' for usage")

// Run rejects redirection before any query, terminal control or state access.
func Run(input *os.File, output io.Writer, appearance Appearance) error {
	out, ok := output.(*os.File)
	if !ok || input == nil || !term.IsTerminal(input.Fd()) || !term.IsTerminal(out.Fd()) || os.Getenv("TERM") == "dumb" {
		return ErrTerminal
	}
	if _, _, err := term.GetSize(out.Fd()); err != nil {
		return ErrTerminal
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	options := []tea.ProgramOption{tea.WithInput(input), tea.WithOutput(out), tea.WithFPS(60)}
	noColor := os.Getenv("NO_COLOR") != ""
	if noColor {
		options = append(options, tea.WithColorProfile(colorprofile.NoTTY))
	}
	m := New(ctx, LoadProfile, appearance, noColor)
	_, err := tea.NewProgram(m, options...).Run()
	if errors.Is(err, tea.ErrInterrupted) {
		return nil
	}
	return err
}
