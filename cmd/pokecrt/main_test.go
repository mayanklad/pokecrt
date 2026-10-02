package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/mayanklad/pokecrt/internal/catalog"
)

// Exercise the executable from an empty directory with no source assets or
// development cache. Production uses this same embedded-art composition root.
func TestInstalledBinary(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("release target is Linux amd64")
	}
	temp := t.TempDir()
	binary := filepath.Join(temp, "pokecrt")
	build := exec.Command("go", "build", "-trimpath", "-buildvcs=false", "-ldflags", "-s -w -X main.version=v0.1-test", "-o", binary, ".")
	build.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS=linux", "GOARCH=amd64", "GOPROXY=off", "GOSUMDB=off")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build installed binary: %v\n%s", err, output)
	}
	empty := filepath.Join(temp, "empty")
	if err := os.Mkdir(empty, 0700); err != nil {
		t.Fatal(err)
	}
	data := filepath.Join(temp, "trainer-data")
	xdg := filepath.Join(temp, "xdg-data")
	baseEnv := make([]string, 0, len(os.Environ()))
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "NO_COLOR=") {
			baseEnv = append(baseEnv, entry)
		}
	}
	baseEnv = append(baseEnv, "POKECRT_DATA_DIR="+data, "XDG_DATA_HOME="+xdg, "GOPROXY=off", "GOSUMDB=off")
	invoke := func(args []string, extraEnv ...string) (int, []byte, []byte) {
		t.Helper()
		command := exec.Command(binary, args...)
		command.Dir = empty
		command.Env = append(append([]string{}, baseEnv...), extraEnv...)
		var stdout, stderr bytes.Buffer
		command.Stdout = &stdout
		command.Stderr = &stderr
		err := command.Run()
		status := 0
		if err != nil {
			var exit *exec.ExitError
			if !errors.As(err, &exit) {
				t.Fatal(err)
			}
			status = exit.ExitCode()
		}
		return status, stdout.Bytes(), stderr.Bytes()
	}
	for _, args := range [][]string{nil, {"--help"}, {"--version"}, {"print", "--help"}, {"print"}, {"print", "--name", "charizard"}, {"print", "--name", "ivysaur"}, {"print", "--name", "blastoise"}, {"print", "--name", "charizard", "--form", "mega-x", "--shiny", "--type", "fire,dragon"}, {"print", "--name", "meowstic", "--gender", "female", "--shiny"}, {"list", "--help"}, {"list", "--name", "charizard"}, {"list", "--name", "charizard", "--form", "mega-x", "--shiny", "--details"}, {"list", "--name", "oinkologne", "--details"}} {
		status, stdout, stderr := invoke(args)
		if status != 0 || len(stdout) == 0 || len(stderr) != 0 {
			t.Fatalf("%v: status=%d stdout=%q stderr=%q", args, status, stdout, stderr)
		}
		if len(args) == 1 && args[0] == "--version" {
			want := "pokecrt v0.1-test\ndataset: " + catalog.DatasetID + "\n"
			if string(stdout) != want {
				t.Fatalf("version: %q; want %q", stdout, want)
			}
		}
	}
	args := []string{"print", "--name", "charizard", "--output", "sprite"}
	_, colored, _ := invoke(args)
	_, emptyColor, _ := invoke(args, "NO_COLOR=")
	_, plain, _ := invoke(args, "NO_COLOR=1")
	if !bytes.Contains(colored, []byte("\x1b[38;2;")) || !bytes.Equal(colored, emptyColor) || bytes.Contains(plain, []byte("\x1b")) {
		t.Fatal("piped ANSI or NO_COLOR contract failed")
	}
	_, compact, _ := invoke([]string{"print", "--name", "charizard"}, "NO_COLOR=1")
	if !bytes.Equal(compact, append(append([]byte{}, plain...), []byte("\n#006 Charizard\n")...)) {
		t.Fatal("compact output boundary differs from sprite output")
	}
	for _, test := range []struct {
		args   []string
		status int
	}{
		{[]string{"print", "--name", "ogerpon"}, 1},
		{[]string{"print", "--name", "unknown"}, 2},
		{[]string{"print", "--output", "full"}, 2},
		{[]string{"print", "--name", "charizard", "--name", "squirtle"}, 2},
	} {
		status, stdout, stderr := invoke(test.args)
		if status != test.status || len(stdout) != 0 || len(stderr) == 0 {
			t.Fatalf("%v: status=%d stdout=%q stderr=%q", test.args, status, stdout, stderr)
		}
	}
	// Public command status contracts must also hold in the installed binary,
	// away from the repository and development cache.
	for _, test := range []struct {
		args           []string
		status         int
		stdout, stderr string
	}{
		{[]string{"list", "--name", "charizard", "--type", "dragon"}, 0, "No Pokémon match the specified filters.", ""},
		{[]string{"print", "--name", "charizard", "--type", "dragon"}, 1, "", "No Pokémon match the specified filters."},
		{[]string{"list", "--name", "charizard", "--gender", "female"}, 0, "No Pokémon match the specified filters.", ""},
		{[]string{"print", "--name", "charizard", "--gender", "female"}, 1, "", "distinct visual-gender"},
		{[]string{"list", "--name", "minior", "--shiny"}, 0, "No Pokémon match the specified filters.", ""},
		{[]string{"print", "--name", "minior", "--shiny"}, 1, "", "shiny artwork unavailable"},
		{[]string{"list", "--form", "unknown"}, 2, "", "unknown form"},
		{[]string{"print", "--gen", "1,,2"}, 2, "", "empty --gen list element"},
		{[]string{"list", "--details", "--details=false"}, 2, "", "only once"},
		{[]string{"list", "--output", "sprite"}, 2, "", "flag provided but not defined"},
		{[]string{"print", "--details"}, 2, "", "flag provided but not defined"},
	} {
		status, out, errout := invoke(test.args, "NO_COLOR=1")
		if status != test.status || (test.stdout == "" && len(out) != 0) || (test.stderr == "" && len(errout) != 0) || !strings.Contains(string(out), test.stdout) || !strings.Contains(string(errout), test.stderr) {
			t.Fatalf("public status %v: %d stdout=%q stderr=%q", test.args, status, out, errout)
		}
	}
	_, detailColored, _ := invoke([]string{"list", "--name", "charizard", "--details"})
	_, detailPlain, _ := invoke([]string{"list", "--name", "charizard", "--details"}, "NO_COLOR=1")
	if !bytes.Contains(detailColored, []byte("\x1b[38;2;")) || bytes.Contains(detailPlain, []byte("\x1b")) {
		t.Fatal("detail ANSI/NO_COLOR contract failed")
	}
	for _, name := range []string{"sprigatito", "annihilape", "terapagos", "pecharunt"} {
		status, output, stderr := invoke([]string{"print", "--name", name}, "NO_COLOR=1")
		if status != 0 || len(output) == 0 || len(stderr) != 0 {
			t.Fatalf("new provider %s: status=%d stderr=%q", name, status, stderr)
		}
	}
	// A closed reader reliably exercises real SIGPIPE/EPIPE handling rather than
	// depending on whether a short output happened to fit inside a pipe buffer.
	for _, pipeArgs := range [][]string{args, {"list", "--name", "charizard", "--details"}, {"list", "--name", "charizard", "--type", "dragon"}} {
		reader, writer, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		if err := reader.Close(); err != nil {
			t.Fatal(err)
		}
		command := exec.Command(binary, pipeArgs...)
		command.Dir = empty
		command.Env = baseEnv
		command.Stdout = writer
		var pipeErrors bytes.Buffer
		command.Stderr = &pipeErrors
		runErr := command.Run()
		closeErr := writer.Close()
		if runErr != nil || closeErr != nil || pipeErrors.Len() != 0 {
			t.Fatalf("closed pipe: run=%v close=%v stderr=%q", runErr, closeErr, pipeErrors.String())
		}
	}
	for _, path := range []string{data, xdg} {
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("created trainer path %s: %v", path, err)
		}
	}
	entries, err := os.ReadDir(empty)
	if err != nil || len(entries) != 0 {
		t.Fatalf("runtime working directory changed: entries=%v err=%v", entries, err)
	}
}
