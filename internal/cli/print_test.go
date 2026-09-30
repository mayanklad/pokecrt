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
	"github.com/mayanklad/pokecrt/internal/query"
	"github.com/mayanklad/pokecrt/internal/render"
	"github.com/mayanklad/pokecrt/internal/sprite"
)

func TestNamedPrintBytes(t *testing.T) {
	species, _ := catalog.ByName("charizard")
	key, _ := query.StandardKey(species)
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

func TestMissingNamedArtwork(t *testing.T) {
	var stdout, stderr bytes.Buffer
	status := Run([]string{"print", "--name", "ivysaur"}, &stdout, &stderr, "dev", catalog.DatasetID)
	if status != 1 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "artwork unavailable") {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
}

func TestRandomPrintAndFailures(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	for index, want := range []string{"#001 Bulbasaur\n", "#006 Charizard\n", "#007 Squirtle\n"} {
		var stdout, stderr bytes.Buffer
		status := runPrint(nil, &stdout, &stderr, func(n int) (int, error) {
			if n != 3 {
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
