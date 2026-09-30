// render-preview is developer tooling, not part of the pokecrt command surface.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/mayanklad/pokecrt/internal/catalog"
	"github.com/mayanklad/pokecrt/internal/render"
	"github.com/mayanklad/pokecrt/internal/sprite"
)

func main() {
	signal.Ignore(syscall.SIGPIPE)
	os.Exit(run())
}

func run() int {
	flags := flag.NewFlagSet("render-preview", flag.ContinueOnError)
	name := flags.String("name", "charizard", "species to preview from the local dataset")
	if err := flags.Parse(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "render-preview: unexpected positional arguments")
		return 2
	}
	species, ok := catalog.ByName(*name)
	if !ok {
		fmt.Fprintf(os.Stderr, "render-preview: unknown species %q\n", *name)
		return 2
	}
	key := catalog.VariantKey{SpeciesID: species.ID, FormID: "standard", Palette: "regular"}
	for _, form := range species.Forms {
		if form.ID == key.FormID {
			key.Gender = form.DefaultGender
			break
		}
	}
	pixels, err := sprite.Decode(key)
	if err != nil {
		fmt.Fprintln(os.Stderr, "render-preview:", err)
		return 1
	}
	output, err := render.Render(pixels, os.Getenv("NO_COLOR") == "")
	if err != nil {
		fmt.Fprintln(os.Stderr, "render-preview:", err)
		return 1
	}
	if _, err := os.Stdout.Write(output); err != nil {
		if errors.Is(err, syscall.EPIPE) {
			return 0
		}
		fmt.Fprintln(os.Stderr, "render-preview:", err)
		return 1
	}
	return 0
}
