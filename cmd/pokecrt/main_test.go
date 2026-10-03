package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/mayanklad/pokecrt/internal/catalog"
	"github.com/mayanklad/pokecrt/internal/storage"
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
	// Public commands must bypass even invalid paths or existing corrupt state.
	corrupt := filepath.Join(temp, "corrupt-data")
	if err := os.Mkdir(corrupt, 0700); err != nil {
		t.Fatal(err)
	}
	corruptDB := filepath.Join(corrupt, "trainers.sqlite3")
	corruptBytes := []byte("deliberately corrupt trainer database")
	if err := os.WriteFile(corruptDB, corruptBytes, 0600); err != nil {
		t.Fatal(err)
	}
	for _, override := range []string{"relative-invalid-path", corrupt} {
		for _, publicArgs := range [][]string{{"--help"}, {"--version"}, {"print", "--name", "charizard"}, {"list", "--name", "eevee", "--details"}} {
			status, _, stderr := invoke(publicArgs, "POKECRT_DATA_DIR="+override)
			if status != 0 || len(stderr) != 0 {
				t.Fatalf("public command touched storage: %v: status=%d stderr=%q", publicArgs, status, stderr)
			}
		}
	}
	if got, err := os.ReadFile(corruptDB); err != nil || !bytes.Equal(got, corruptBytes) {
		t.Fatalf("public command changed corrupt database: %q, %v", got, err)
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

	// Trainer commands run from the installed binary, with no checkout/cache.
	profiles := filepath.Join(temp, "installed-profiles")
	for _, helpArgs := range [][]string{{"encounter", "--help"}, {"trainer", "--help"}, {"trainer", "create", "--help"}, {"trainer", "list", "-h"}, {"trainer", "use", "--help"}, {"trainer", "achievements", "--help"}} {
		status, _, stderr := invoke(helpArgs, "POKECRT_DATA_DIR=relative-invalid")
		if status != 0 || len(stderr) != 0 {
			t.Fatalf("trainer help touched storage: %v: %d %q", helpArgs, status, stderr)
		}
	}
	for _, emptyArgs := range [][]string{{"trainer"}, {"trainer", "list"}} {
		status, stdout, stderr := invoke(emptyArgs, "POKECRT_DATA_DIR="+profiles)
		if status != 0 || len(stderr) != 0 || !bytes.Contains(stdout, []byte("No trainer profiles found.")) {
			t.Fatalf("installed first run: %d %q %q", status, stdout, stderr)
		}
	}
	for _, invalidArgs := range [][]string{{"encounter", "--name", "charizard"}, {"encounter", "--sprite-only"}, {"encounter", "--output", "invalid"}} {
		status, stdout, stderr := invoke(invalidArgs, "POKECRT_DATA_DIR="+profiles)
		if status != 2 || len(stdout) != 0 || len(stderr) == 0 {
			t.Fatalf("installed encounter syntax %v: %d %q %q", invalidArgs, status, stdout, stderr)
		}
	}
	status, stdout, stderr := invoke([]string{"encounter"}, "POKECRT_DATA_DIR="+profiles)
	if status != 1 || len(stdout) != 0 || !bytes.Contains(stderr, []byte("trainer create <name>")) {
		t.Fatalf("installed encounter first run: %d %q %q", status, stdout, stderr)
	}
	status, stdout, stderr = invoke([]string{"trainer", "achievements"}, "POKECRT_DATA_DIR="+profiles)
	if status != 1 || len(stdout) != 0 || !bytes.Contains(stderr, []byte("trainer create")) {
		t.Fatalf("installed achievement first run: %d %q %q", status, stdout, stderr)
	}
	if _, err := os.Stat(profiles); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("first-run queries created profile storage")
	}
	for _, step := range []struct {
		args     []string
		status   int
		contains string
	}{
		{[]string{"trainer", "create", "Mayank"}, 0, "Active trainer: Mayank"},
		{[]string{"trainer", "create", "Professor Oak"}, 0, "Created trainer: Professor Oak"},
		{[]string{"trainer", "create", "MAYANK"}, 1, ""},
		{[]string{"trainer", "create", "Élodie"}, 0, "Created trainer: Élodie"},
		{[]string{"trainer", "create", "E\u0301LODIE"}, 1, ""},
		{[]string{"trainer", "--name", "Professor Oak"}, 0, "Active: No"},
		{[]string{"trainer"}, 0, "Trainer: Mayank"},
		{[]string{"trainer", "use", "professor oak"}, 0, "Active trainer: Professor Oak"},
		{[]string{"trainer"}, 0, "Trainer: Professor Oak"},
		{[]string{"trainer", "list"}, 0, "Professor Oak (active)"},
		{[]string{"trainer", "use", "missing"}, 1, ""},
		{[]string{"trainer", "create", "A\nB"}, 2, ""},
	} {
		status, stdout, stderr := invoke(step.args, "POKECRT_DATA_DIR="+profiles)
		if status != step.status || !bytes.Contains(stdout, []byte(step.contains)) {
			t.Fatalf("installed profile step %v: %d %q %q", step.args, status, stdout, stderr)
		}
		if step.status == 0 && len(stderr) != 0 {
			t.Fatalf("profile error %q", stderr)
		}
		if step.status != 0 && (len(stdout) != 0 || len(stderr) == 0) {
			t.Fatal("profile error streams")
		}
	}
	profileDB := filepath.Join(profiles, "trainers.sqlite3")
	before, err := os.ReadFile(profileDB)
	if err != nil {
		t.Fatal(err)
	}
	for _, publicArgs := range [][]string{{"print", "--name", "charizard"}, {"list", "--name", "eevee", "--details"}, {"--version"}, {"--help"}} {
		status, _, stderr := invoke(publicArgs, "POKECRT_DATA_DIR="+profiles)
		if status != 0 || len(stderr) != 0 {
			t.Fatalf("public command with profiles: %v", publicArgs)
		}
	}
	after, err := os.ReadFile(profileDB)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("public command mutated trainer database")
	}

	// Every installed mode uses embedded assets and records one encounter.
	for _, mode := range []string{"full", "compact", "no-title", "achievements", "sprite"} {
		status, out, errout := invoke([]string{"encounter", "--output", mode}, "POKECRT_DATA_DIR="+profiles, "NO_COLOR=1")
		if status != 0 || len(out) == 0 || len(errout) != 0 || bytes.Contains(out, []byte("\x1b")) {
			t.Fatalf("installed encounter %s: %d %q", mode, status, errout)
		}
		if (mode == "full" || mode == "no-title") && !bytes.Contains(out, []byte("XP gained:")) {
			t.Fatal("installed encounter progress missing")
		}
		if mode == "sprite" && (bytes.Contains(out, []byte("#")) || bytes.Contains(out, []byte("XP")) || bytes.Contains(out, []byte("🏆"))) {
			t.Fatal("installed sprite text")
		}
	}
	status, colored, errout := invoke([]string{"encounter", "--output", "sprite"}, "POKECRT_DATA_DIR="+profiles)
	if status != 0 || len(errout) != 0 || !bytes.Contains(colored, []byte("\x1b[38;2;")) {
		t.Fatal("installed encounter color")
	}
	// Real SIGPIPE handling preserves the seventh committed encounter.
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	reader.Close()
	command := exec.Command(binary, "encounter", "--output", "sprite")
	command.Dir = empty
	command.Env = append(append([]string{}, baseEnv...), "POKECRT_DATA_DIR="+profiles)
	command.Stdout = writer
	var pipeErrors bytes.Buffer
	command.Stderr = &pipeErrors
	runErr := command.Run()
	closeErr := writer.Close()
	if runErr != nil || closeErr != nil || pipeErrors.Len() != 0 {
		t.Fatalf("installed encounter pipe %v %v %q", runErr, closeErr, pipeErrors.String())
	}
	beforeViews, err := os.ReadFile(profileDB)
	if err != nil {
		t.Fatal(err)
	}
	for _, view := range []struct {
		args []string
		want string
	}{
		{[]string{"trainer"}, "Encounters: 7"},
		{[]string{"trainer", "--name", "Mayank"}, "Encounters: 0"},
		{[]string{"trainer", "achievements"}, "First Contact"},
	} {
		status, out, stderr := invoke(view.args, "POKECRT_DATA_DIR="+profiles)
		if status != 0 || len(stderr) != 0 || !bytes.Contains(out, []byte(view.want)) {
			t.Fatalf("installed trainer view %v: %d %q %q", view.args, status, out, stderr)
		}
	}
	afterViews, err := os.ReadFile(profileDB)
	if err != nil || !bytes.Equal(beforeViews, afterViews) {
		t.Fatal("installed browsing mutated state")
	}
	reader2, writer2, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	reader2.Close()
	achievementPipe := exec.Command(binary, "trainer", "achievements")
	achievementPipe.Dir = empty
	achievementPipe.Env = append(append([]string{}, baseEnv...), "POKECRT_DATA_DIR="+profiles)
	achievementPipe.Stdout = writer2
	var achievementErrors bytes.Buffer
	achievementPipe.Stderr = &achievementErrors
	runErr = achievementPipe.Run()
	writer2.Close()
	if runErr != nil || achievementErrors.Len() != 0 {
		t.Fatal("installed achievement pipe", runErr, achievementErrors.String())
	}
	repo, err := storage.ReadOnly(context.Background(), profileDB)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	var encounters, wrongTrainer, invalidXP int
	if err := repo.QueryRowContext(context.Background(), "SELECT count(*) FROM encounters").Scan(&encounters); err != nil {
		t.Fatal(err)
	}
	if err := repo.QueryRowContext(context.Background(), "SELECT count(*) FROM encounters e JOIN trainers t ON t.id=e.trainer_id WHERE t.display_name!='Professor Oak'").Scan(&wrongTrainer); err != nil {
		t.Fatal(err)
	}
	if err := repo.QueryRowContext(context.Background(), "SELECT count(*) FROM trainer_progress p WHERE xp_total!=COALESCE((SELECT sum(xp_awarded) FROM encounters e WHERE e.trainer_id=p.trainer_id),0)").Scan(&invalidXP); err != nil {
		t.Fatal(err)
	}
	if encounters != 7 || wrongTrainer != 0 || invalidXP != 0 {
		t.Fatalf("installed encounter counts %d %d %d", encounters, wrongTrainer, invalidXP)
	}
}
