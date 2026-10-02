package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"github.com/mayanklad/pokecrt/internal/catalog"
	"github.com/mayanklad/pokecrt/internal/render"
	"github.com/mayanklad/pokecrt/internal/sprite"
)

func listOutput(t *testing.T, args ...string) string {
	t.Helper()
	var out, errout bytes.Buffer
	if status := Run(append([]string{"list"}, args...), &out, &errout, "dev", catalog.DatasetID); status != 0 || errout.Len() != 0 {
		t.Fatalf("%v: status=%d stderr=%q", args, status, errout.String())
	}
	return out.String()
}

func TestCompactCatalogIsSortedCompleteAndHonest(t *testing.T) {
	text := listOutput(t)
	lines := strings.Split(strings.TrimSpace(text), "\n")
	if len(lines) != len(catalog.All())+1 || !strings.Contains(lines[0], "NUMBER") || !strings.Contains(lines[0], "ARTWORK") {
		t.Fatalf("rows=%d header=%q", len(lines), lines[0])
	}
	previous := 0
	unavailable := 0
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		id, err := strconv.Atoi(strings.TrimPrefix(fields[0], "#"))
		if err != nil || id <= previous {
			t.Fatalf("unsorted row %q", line)
		}
		previous = id
		if strings.HasSuffix(line, "Unavailable") {
			unavailable++
		}
		if strings.ContainsAny(line, "▄▀\x1b") {
			t.Fatal("compact list rendered artwork")
		}
	}
	if unavailable != 12 || previous != 1025 {
		t.Fatalf("unavailable=%d last=%d", unavailable, previous)
	}
	for _, args := range [][]string{{"--name=charizard", "--form=mega-x", "--type=fire,dragon"}, {"--name=meowstic", "--gender=female", "--shiny"}} {
		text := listOutput(t, args...)
		if len(strings.Split(strings.TrimSpace(text), "\n")) != 2 || !strings.Contains(text, "Available") {
			t.Fatalf("selected row=%q", text)
		}
	}
	text = listOutput(t, "--name=charizard", "--form=mega-x")
	if !strings.Contains(text, "Charizard · Mega X") || !strings.Contains(text, "Fire / Dragon") {
		t.Fatalf("wrong selected form row: %q", text)
	}
	text = listOutput(t, "--name=oinkologne", "--gender=female")
	if !strings.Contains(text, "Female") || !strings.Contains(text, "Unavailable") {
		t.Fatalf("missing metadata-only gender row: %q", text)
	}
}

func TestDetailedCatalogDisplaysOneExactSpriteAndInventory(t *testing.T) {
	for _, noColor := range []string{"", "1"} {
		t.Setenv("NO_COLOR", noColor)
		text := listOutput(t, "--name=charizard", "--form=mega-x", "--shiny", "--details")
		key := catalog.VariantKey{SpeciesID: 6, FormID: "mega-x", Gender: "default", Palette: "shiny"}
		pixels, err := sprite.Decode(key)
		if err != nil {
			t.Fatal(err)
		}
		art, err := render.Render(pixels, noColor == "")
		if err != nil {
			t.Fatal(err)
		}
		if strings.Count(text, string(art)) != 1 {
			t.Fatal("selected sprite absent or repeated")
		}
		for _, want := range []string{"#006 Charizard · Mega X · Shiny", "Generation: 1", "Types: Fire / Dragon", "Color: Red", "Evolution stage: 3", "Baby: No", "Legendary: No", "Mythical: No", "#004 Charmander → #005 Charmeleon", "Standard [standard]: Regular, Shiny", "Mega X [mega-x]: Regular, Shiny", "Gigantamax [gmax]: Regular, Shiny"} {
			if !strings.Contains(text, want) {
				t.Fatalf("missing %q in details", want)
			}
		}
		if strings.Contains(text, "Height:") || strings.Contains(text, "Abilities:") {
			t.Fatal("invented unimplemented metadata")
		}
	}
	text := listOutput(t, "--name=meowstic", "--gender=female", "--shiny", "--details")
	for _, want := range []string{"#678 Meowstic · Female · Shiny", "Female: Regular, Shiny", "Male (default): Regular, Shiny"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing gender inventory %q", want)
		}
	}
	text = listOutput(t, "--name=minior", "--details")
	if !strings.Contains(text, "Standard [standard]: Regular\n") {
		t.Fatal("missing shiny palette was invented")
	}
}

func TestUnavailableDetailsNeverBorrowAlternateArtwork(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	for _, args := range [][]string{{"--name=ogerpon"}, {"--name=oinkologne", "--gender=female"}, {"--name=tatsugiri", "--form=curly-mega"}} {
		text := listOutput(t, append(args, "--details")...)
		if !strings.Contains(text, "Artwork unavailable for this selection.") || strings.ContainsAny(text, "▄▀\x1b") {
			t.Fatalf("borrowed artwork: %q", text)
		}
		if !strings.Contains(text, "Forms and artwork:") {
			t.Fatal("missing form inventory")
		}
	}
	text := listOutput(t, "--name=ogerpon", "--details")
	if !strings.Contains(text, "[cornerstone-mask]: Regular, Shiny") {
		t.Fatal("accepted alternate inventory hidden")
	}
}

func TestListEmptyQueriesAndInvalidInvocations(t *testing.T) {
	for _, args := range [][]string{{"--name=charizard", "--type=dragon"}, {"--name=charizard", "--form=hisui"}, {"--name=charizard", "--gender=female"}, {"--name=minior", "--shiny"}, {"--name=charizard", "--gen=2", "--details"}} {
		if text := listOutput(t, args...); text != "No Pokémon match the specified filters.\n" {
			t.Fatalf("empty query=%q", text)
		}
	}
	for _, args := range [][]string{{"charizard"}, {"--output=compact"}, {"--details", "--details=false"}, {"--help", "-h"}, {"--gen=10"}, {"--form=unknown"}, {"--type=fire,,water"}, {"--gender=default"}, {"--name=char"}, {"--details=invalid"}, {"--name="}, {"--shiny", "false"}} {
		var out, errout bytes.Buffer
		status := Run(append([]string{"list"}, args...), &out, &errout, "dev", catalog.DatasetID)
		if status != 2 || out.Len() != 0 || errout.Len() == 0 {
			t.Fatalf("%v: status=%d stderr=%q", args, status, errout.String())
		}
	}
	if listOutput(t, "--name=charizard", "--details=false") != listOutput(t, "--name=charizard") {
		t.Fatal("false details changed compact output")
	}
}

func TestEvolutionBranchesSeparatorsAndStateFreeListing(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	base := t.TempDir()
	data := filepath.Join(base, "data")
	xdg := filepath.Join(base, "xdg")
	t.Setenv("POKECRT_DATA_DIR", data)
	t.Setenv("XDG_DATA_HOME", xdg)
	text := listOutput(t, "--name=eevee", "--details")
	for _, want := range []string{"#133 Eevee → #134 Vaporeon, #135 Jolteon, #136 Flareon", "#700 Sylveon"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing family branch %q", want)
		}
	}
	compact := listOutput(t, "--gen=1", "--type=fire")
	count := len(strings.Split(strings.TrimSpace(compact), "\n")) - 1
	details := listOutput(t, "--gen=1", "--type=fire", "--details")
	if count < 2 || strings.Count(details, "\n---\n\n") != count-1 {
		t.Fatal("results capped or separators inconsistent")
	}
	if listOutput(t, "--help") != listHelp || listOutput(t, "-h") != listHelp {
		t.Fatal("help mismatch")
	}
	for _, path := range []string{data, xdg} {
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("created state path %s", path)
		}
	}
	for _, tc := range []struct {
		err    error
		status int
	}{{syscall.EPIPE, 0}, {errors.New("write failed"), 1}} {
		var errout bytes.Buffer
		status := Run([]string{"list", "--name=charizard"}, failingWriter{tc.err}, &errout, "dev", catalog.DatasetID)
		if status != tc.status || (errout.Len() != 0) != (tc.status != 0) {
			t.Fatalf("writer status=%d stderr=%q", status, errout.String())
		}
	}
}
