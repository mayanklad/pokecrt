package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/mayanklad/pokecrt/internal/catalog"
	"github.com/mayanklad/pokecrt/internal/render"
	"github.com/mayanklad/pokecrt/internal/sprite"
)

func TestNamedPrintBytes(t *testing.T) {
	species, _ := catalog.ByName("charizard")
	key, _ := catalog.StandardKey(species)
	pixels, err := sprite.Decode(key)
	if err != nil {
		t.Fatal(err)
	}
	for _, noColor := range []string{"", "1"} {
		t.Setenv("NO_COLOR", noColor)
		artwork, err := render.Render(pixels, noColor == "")
		if err != nil {
			t.Fatal(err)
		}
		for _, mode := range []string{"compact", "sprite"} {
			var stdout, stderr bytes.Buffer
			status := runPrint([]string{"--name", " CHARIZARD ", "--output", mode}, &stdout, &stderr, func(int) (int, error) { t.Fatal("named print called random selector"); return 0, nil })
			want := string(artwork)
			if mode == "compact" {
				want += "\n#006 Charizard\n"
			}
			if status != 0 || stderr.Len() != 0 || stdout.String() != want {
				t.Fatalf("%s NO_COLOR=%q: status=%d stderr=%q stdout=%q", mode, noColor, status, stderr.String(), stdout.String())
			}
		}
	}
}

func TestPrintInvocationErrors(t *testing.T) {
	tests := [][]string{
		{"charizard"}, {"--random"}, {"--name"}, {"--name="}, {"--name", " "}, {"--name", "charizard", "--name=squirtle"},
		{"--output", "full"}, {"--output", "sprite", "--output=compact"}, {"--help", "-h"}, {"--help=invalid"},
		{"--name", "char"}, {"--name", "not-a-pokemon"}, {"-name", "charizard"}, {"--", "charizard"},
	}
	for _, args := range tests {
		var stdout, stderr bytes.Buffer
		status := Run(append([]string{"print"}, args...), &stdout, &stderr, "dev", catalog.DatasetID)
		if status != 2 || stdout.Len() != 0 || stderr.Len() == 0 {
			t.Fatalf("%v: status=%d stdout=%q stderr=%q", args, status, stdout.String(), stderr.String())
		}
	}
}

func TestNewlyBundledNamedArtwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	status := Run([]string{"print", "--name", "ivysaur"}, &stdout, &stderr, "dev", catalog.DatasetID)
	if status != 0 || stderr.Len() != 0 || !strings.HasSuffix(stdout.String(), "#002 Ivysaur\n") {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
}

func TestRandomPrintAndFailures(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	for _, test := range []struct {
		index int
		name  string
	}{{0, "#001 Bulbasaur"}, {24, "#025 Pikachu"}, {150, "#151 Mew"}, {151, "#152 Chikorita"}, {250, "#251 Celebi"}, {251, "#252 Treecko"}, {385, "#386 Deoxys"}, {386, "#387 Turtwig"}, {492, "#493 Arceus"}, {493, "#494 Victini"}, {648, "#649 Genesect"}, {649, "#650 Chespin"}, {720, "#721 Volcanion"}, {721, "#722 Rowlet"}, {808, "#809 Melmetal"}, {809, "#810 Grookey"}, {897, "#898 Calyrex"}, {898, "#906 Sprigatito"}, {1012, "#1025 Pecharunt"}} {
		index := test.index
		want := test.name + "\n"
		var stdout, stderr bytes.Buffer
		status := runPrint(nil, &stdout, &stderr, func(n int) (int, error) {
			if n != 1013 {
				t.Fatalf("n=%d", n)
			}
			return index, nil
		})
		if status != 0 || stderr.Len() != 0 || !strings.HasSuffix(stdout.String(), "\n"+want) {
			t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
		}
	}
	var stdout, stderr bytes.Buffer
	status := runPrint(nil, &stdout, &stderr, func(int) (int, error) { return 0, errors.New("entropy unavailable") })
	if status != 1 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "entropy unavailable") {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
	for _, test := range []struct {
		err    error
		status int
	}{{syscall.EPIPE, 0}, {errors.New("write failed"), 1}} {
		stderr.Reset()
		status := Run([]string{"print", "--name", "charizard"}, failingWriter{test.err}, &stderr, "dev", catalog.DatasetID)
		if status != test.status || (stderr.Len() != 0) != (test.status != 0) {
			t.Fatalf("status=%d stderr=%q", status, stderr.String())
		}
	}
}

func TestPrintHelpAndNoState(t *testing.T) {
	base := t.TempDir()
	data := filepath.Join(base, "data")
	xdg := filepath.Join(base, "xdg")
	t.Setenv("POKECRT_DATA_DIR", data)
	t.Setenv("XDG_DATA_HOME", xdg)
	for _, args := range [][]string{{"print", "--help"}, {"print", "-h"}, {"print", "--name", "charizard"}, {"print"}} {
		var stdout, stderr bytes.Buffer
		if status := Run(args, &stdout, &stderr, "dev", catalog.DatasetID); status != 0 {
			t.Fatalf("%v: %d %q", args, status, stderr.String())
		}
		if len(args) == 2 && stdout.String() != printHelp {
			t.Fatal("unexpected help")
		}
	}
	for _, path := range []string{data, xdg} {
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("state path created: %s: %v", path, err)
		}
	}
}

func TestUnauditedArtworkHasNoFallback(t *testing.T) {
	for _, name := range []string{"wyrdeer", "oinkologne", "palafin", "tatsugiri", "ogerpon", "dudunsparce"} {
		var stdout, stderr bytes.Buffer
		status := Run([]string{"print", "--name", name}, &stdout, &stderr, "dev", catalog.DatasetID)
		if status != 1 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "artwork unavailable") {
			t.Fatalf("%s: status=%d stdout=%q stderr=%q", name, status, stdout.String(), stderr.String())
		}
	}
}

func TestSelectorFlagSetValidation(t *testing.T) {
	for _, args := range [][]string{
		{"--gen=1,,2"}, {"--gen=0"}, {"--gen=+1"}, {"--gen=1.0"}, {"--gen=10"}, {"--stage=99"}, {"--type=fire,"}, {"--type-any=stellar"}, {"--color=silver"}, {"--form=base"}, {"--gender=default"},
		{"--shiny=true", "--shiny=false"}, {"--baby", "--baby"}, {"--gen=1", "--gen=2"}, {"--type=fire", "--type=water"}, {"--shiny=1"}, {"--legendary=false", "--legendary"}, {"--shiny", "false"}, {"--gender="}, {"--form="},
		{"--name", "charizard", "--unknown"}, {"--name", "--help"}, {"--color", "--shiny"},
	} {
		var out, errout bytes.Buffer
		status := Run(append([]string{"print"}, args...), &out, &errout, "dev", catalog.DatasetID)
		if status != 2 || out.Len() != 0 || errout.Len() == 0 {
			t.Fatalf("%v: %d stdout=%q stderr=%q", args, status, out.String(), errout.String())
		}
	}
	q, err := parsePrintFlags([]string{"--gen= 1,2,1 ", "--type= FIRE,fire ", "--legendary=false"})
	if err != nil || len(q.selection.Generations) != 2 || len(q.selection.Types) != 1 || q.selection.Legendary {
		t.Fatalf("options=%+v err=%v", q, err)
	}
}

func TestSelectedVariantBytesAndHeadings(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	for _, tc := range []struct {
		args    []string
		key     catalog.VariantKey
		heading string
	}{
		{[]string{"--name=charizard", "--form=mega-x", "--shiny"}, catalog.VariantKey{SpeciesID: 6, FormID: "mega-x", Gender: "default", Palette: "shiny"}, "#006 Charizard · Mega X · Shiny"},
		{[]string{"--name=meowstic", "--gender=female", "--shiny"}, catalog.VariantKey{SpeciesID: 678, FormID: "standard", Gender: "female", Palette: "shiny"}, "#678 Meowstic · Female · Shiny"},
		{[]string{"--name=hippowdon"}, catalog.VariantKey{SpeciesID: 450, FormID: "standard", Gender: "male", Palette: "regular"}, "#450 Hippowdon"},
	} {
		pixels, err := sprite.Decode(tc.key)
		if err != nil {
			t.Fatal(err)
		}
		art, err := render.Render(pixels, false)
		if err != nil {
			t.Fatal(err)
		}
		for _, mode := range []string{"compact", "sprite"} {
			var out, errout bytes.Buffer
			args := append(append([]string{}, tc.args...), "--output="+mode)
			status := runPrint(args, &out, &errout, func(int) (int, error) { t.Fatal("named variant randomized"); return 0, nil })
			want := string(art)
			if mode == "compact" {
				want += "\n" + tc.heading + "\n"
			}
			if status != 0 || errout.Len() != 0 || out.String() != want {
				t.Fatalf("%v: %d stderr=%q heading=%q", args, status, errout.String(), out.String())
			}
		}
	}
}

func TestValidContradictionsAndUnsupportedAppearancesAreOperational(t *testing.T) {
	for _, args := range [][]string{
		{"--name=charizard", "--type=dragon"}, {"--name=charizard", "--form=hisui"}, {"--name=charizard", "--gender=female"}, {"--name=oinkologne", "--gender=female"}, {"--name=charizard", "--gen=2"}, {"--name=charizard", "--baby"}, {"--name=mew", "--legendary"}, {"--type=fire,water,grass"}, {"--name=minior", "--shiny"},
	} {
		var out, errout bytes.Buffer
		status := Run(append([]string{"print"}, args...), &out, &errout, "dev", catalog.DatasetID)
		if status != 1 || out.Len() != 0 || errout.Len() == 0 {
			t.Fatalf("%v: %d stdout=%q stderr=%q", args, status, out.String(), errout.String())
		}
	}
	var out, errout bytes.Buffer
	status := Run([]string{"print", "--name=charizard", "--legendary=false", "--baby=false", "--mythical=false", "--shiny=false"}, &out, &errout, "dev", catalog.DatasetID)
	if status != 0 || !strings.HasSuffix(out.String(), "#006 Charizard\n") {
		t.Fatalf("false flags excluded species: %d %q", status, errout.String())
	}
}
