package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/mayanklad/pokecrt/internal/catalog"
	"github.com/mayanklad/pokecrt/internal/sprite"
	"github.com/mayanklad/pokecrt/internal/storage"
	"github.com/mayanklad/pokecrt/internal/trainer"
)

func executeProfileTest(t *testing.T, path string, args []string, stdout io.Writer) (int, string) {
	t.Helper()
	var stderr bytes.Buffer
	status := executeTrainer(args, stdout, &stderr, func() (string, error) { return path, nil }, openProfiles, func() time.Time { return time.UnixMilli(1234) })
	return status, stderr.String()
}

func TestTrainerSyntaxAndHelpBeforeStorage(t *testing.T) {
	for _, tc := range []struct {
		args   []string
		status int
	}{
		{[]string{"--help"}, 0}, {[]string{"-h"}, 0},
		{[]string{"create", "--help"}, 0}, {[]string{"list", "--help"}, 0}, {[]string{"use", "-h"}, 0}, {[]string{"achievements", "--help"}, 0},
		{[]string{"create"}, 2}, {[]string{"use"}, 2},
		{[]string{"create", "Oak", "Extra"}, 2}, {[]string{"list", "Oak"}, 2},
		{[]string{"create", "Oak", "--help"}, 2}, {[]string{"use", "Oak", "-h"}, 2},
		{[]string{"--name"}, 2}, {[]string{"--name="}, 2},
		{[]string{"--name", "Oak", "--name", "Alice"}, 2},
		{[]string{"--help", "-h"}, 2}, {[]string{"--help", "--unknown"}, 2},
		{[]string{"-name", "Oak"}, 2}, {[]string{"create", "--name", "Oak"}, 2},
		{[]string{"create", ""}, 2}, {[]string{"create", "A\nB"}, 2},
		{[]string{"create", strings.Repeat("雪", 33)}, 2},
		{[]string{"delete", "Oak"}, 2}, {[]string{"show"}, 2}, {[]string{"achievements", "--name", "Oak"}, 2},
	} {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			touched := false
			resolve := func() (string, error) { touched = true; return "", errors.New("storage must not be resolved") }
			open := func(context.Context, string, string) (profileRepository, error) {
				touched = true
				return nil, errors.New("storage must not be opened")
			}
			var out, err bytes.Buffer
			status := executeTrainer(tc.args, &out, &err, resolve, open, time.Now)
			if touched || status != tc.status {
				t.Fatalf("status %d, touched=%v, stderr=%q", status, touched, err.String())
			}
			if tc.status == 0 && (out.Len() == 0 || err.Len() != 0) {
				t.Fatal("help streams")
			}
			if tc.status == 2 && (out.Len() != 0 || err.Len() == 0) {
				t.Fatal("syntax streams")
			}
		})
	}
}

func TestTrainerFirstRunAndProfileFlow(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "trainers.sqlite3")
	var out bytes.Buffer
	for _, args := range [][]string{nil, {"list"}} {
		status, stderr := executeProfileTest(t, path, args, &out)
		if status != 0 || stderr != "" || out.String() != noProfiles {
			t.Fatalf("first run: %d %q %q", status, out.String(), stderr)
		}
		out.Reset()
	}
	for _, args := range [][]string{{"use", "Oak"}, {"--name", "Oak"}} {
		status, _ := executeProfileTest(t, path, args, &out)
		if status != 1 || out.Len() != 0 {
			t.Fatal("unknown profile should fail")
		}
	}
	if _, err := os.Stat(filepath.Dir(path)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("read-only first run created state")
	}
	cases := []struct {
		args     []string
		status   int
		contains string
	}{
		{[]string{"create", "  Professor Oak  "}, 0, "Active trainer: Professor Oak"},
		{[]string{"create", "Alice"}, 0, "Created trainer: Alice"},
		{[]string{"create", "PROFESSOR OAK"}, 1, ""},
		{[]string{"list"}, 0, "Professor Oak (active)"},
		{[]string{"--name=ALICE"}, 0, "Active: No"},
		{nil, 0, "Trainer: Professor Oak"},
		{[]string{"use", "alice"}, 0, "Active trainer: Alice"},
		{nil, 0, "Trainer: Alice"},
		{[]string{"--name", "Professor Oak"}, 0, "Created: 1970-01-01T00:00:01.234Z"},
		{[]string{"use", "Missing"}, 1, ""},
		{nil, 0, "Trainer: Alice"},
		{[]string{"create", "--", "--help"}, 0, "Created trainer: --help"},
		{[]string{"--name", "--help"}, 0, "Trainer: --help"},
	}
	for _, tc := range cases {
		out.Reset()
		status, stderr := executeProfileTest(t, path, tc.args, &out)
		if status != tc.status || !strings.Contains(out.String(), tc.contains) {
			t.Fatalf("%v: %d %q %q", tc.args, status, out.String(), stderr)
		}
		if tc.status == 0 && stderr != "" {
			t.Fatalf("unexpected error %q", stderr)
		}
		if tc.status == 1 && (out.Len() != 0 || stderr == "") {
			t.Fatal("error streams")
		}
	}
	repo, err := storage.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	if err := repo.Write(context.Background(), func(tx *storage.Tx) error {
		_, err := tx.ExecContext(context.Background(), "UPDATE app_state SET active_trainer_id = NULL")
		return err
	}); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	status, stderr := executeProfileTest(t, path, nil, &out)
	if status != 1 || out.Len() != 0 || !strings.Contains(stderr, "trainer use <name>") || !strings.Contains(stderr, "Professor Oak") {
		t.Fatalf("inactive guidance: %d %q", status, stderr)
	}
	out.Reset()
	status, stderr = executeProfileTest(t, path, []string{"achievements"}, &out)
	if status != 1 || out.Len() != 0 || !strings.Contains(stderr, "Professor Oak") || !strings.Contains(stderr, "trainer use <name>") {
		t.Fatal("achievement inactive guidance", status, stderr)
	}
}

func TestTrainerCorruptStateAndPathErrors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trainers.sqlite3")
	original := []byte("corrupt trainer storage")
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{nil, {"list"}, {"create", "Oak"}, {"use", "Oak"}} {
		var out bytes.Buffer
		status, _ := executeProfileTest(t, path, args, &out)
		if status != 1 || out.Len() != 0 {
			t.Fatalf("corrupt storage accepted: %v", args)
		}
	}
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, original) {
		t.Fatal("corrupt storage changed")
	}
	for _, tc := range []struct {
		err    error
		status int
	}{
		{storage.ErrInvalidPath, 2}, {errors.New("home resolution failed"), 1},
	} {
		var out, stderr bytes.Buffer
		status := executeTrainer(nil, &out, &stderr, func() (string, error) { return "", tc.err }, openProfiles, time.Now)
		if status != tc.status || out.Len() != 0 {
			t.Fatalf("path status %d", status)
		}
	}
}

func TestBrokenPipeAfterProfileCommit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "trainers.sqlite3")
	status, stderr := executeProfileTest(t, path, []string{"create", "Mayank"}, pipeWriter{})
	if status != 0 || stderr != "" {
		t.Fatalf("pipe: %d %q", status, stderr)
	}
	repo, err := storage.ReadOnly(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	p, err := repo.ActiveProfile(context.Background())
	if err != nil || p.Name != "Mayank" {
		t.Fatal("broken pipe undid committed profile")
	}
}

type pipeWriter struct{}

func (pipeWriter) Write([]byte) (int, error) { return 0, syscall.EPIPE }

func TestTrainerStatisticsAchievementsAndNamedProfile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trainers.sqlite3")
	ctx := context.Background()
	repo, err := storage.Initialize(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	name, _ := trainer.ParseName("Collector")
	p, err := repo.CreateProfile(ctx, name, 1000)
	if err != nil {
		t.Fatal(err)
	}
	otherName, _ := trainer.ParseName("Other")
	other, err := repo.CreateProfile(ctx, otherName, 2000)
	if err != nil {
		t.Fatal(err)
	}
	species, _ := catalog.ByNumber(6)
	for _, f := range species.Forms {
		if f.ID == "mega-x" {
			species.Forms = []catalog.Form{f}
			break
		}
	}
	pool, err := trainer.NewPool([]catalog.Species{species}, func(k catalog.VariantKey) bool { _, ok := sprite.Lookup(k); return ok })
	if err != nil {
		t.Fatal(err)
	}
	choice, err := pool.Select(func(int) (int, error) { return 0, nil })
	if err != nil {
		t.Fatal(err)
	}
	for _, ms := range []int64{3000, 0} {
		if _, err = repo.RecordEncounter(ctx, p.ID, choice, ms); err != nil {
			t.Fatal(err)
		}
	}
	repo.Close()
	before, _ := os.ReadFile(path)
	hash := sha256.Sum256(before)
	for _, tc := range []struct {
		args   []string
		want   []string
		hidden []string
	}{
		{nil, []string{"Trainer: Collector", "Level: 1", "Total XP: 280", "Encounters: 2", "Shiny encounters: 2", "Shiny variants collected (historical): 1", "First encounter: 1970-01-01T00:00:00Z", "Generation 1: 1 discovered; eligible 1 / 151"}, []string{"Charizard", "Mega X"}},
		{[]string{"--name", "Other"}, []string{"Trainer: Other", "Active: No", "Total XP: 0", "First encounter: None"}, []string{"Charizard", "Mega X"}},
		{[]string{"achievements"}, []string{"Achievements: Collector", "First Contact", "Shiny Discovery", "Earned: 1970-01-01T00:00:03Z", "10 Encounters", "Progress: 2 / 10", "Type Explorer", "Progress: 2 / 18", "Locked ("}, []string{"Charizard", "Charmander", "Mega X"}},
	} {
		var out bytes.Buffer
		code, stderr := executeProfileTest(t, path, tc.args, &out)
		if code != 0 || stderr != "" {
			t.Fatal(code, stderr)
		}
		for _, want := range tc.want {
			if !strings.Contains(out.String(), want) {
				t.Fatal(tc.args, "missing", want, out.String())
			}
		}
		for _, hidden := range tc.hidden {
			if strings.Contains(out.String(), hidden) {
				t.Fatal("hidden identity", hidden)
			}
		}
	}
	for _, tc := range []struct {
		writer io.Writer
		want   int
	}{{pipeWriter{}, 0}, {failingWriter{errors.New("write failed")}, 1}} {
		code, _ := executeProfileTest(t, path, []string{"achievements"}, tc.writer)
		if code != tc.want {
			t.Fatal(code)
		}
	}
	after, _ := os.ReadFile(path)
	if sha256.Sum256(after) != hash {
		t.Fatal("view mutated DB")
	}
	repo, err = storage.ReadOnly(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	active, err := repo.ActiveProfile(ctx)
	if err != nil || active.ID != p.ID || active.ID == other.ID {
		t.Fatal("named view switched trainer", active, err)
	}
}

func TestTrainerAchievementsMissingStateAndDedicatedHelp(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "trainers.sqlite3")
	var out bytes.Buffer
	code, stderr := executeProfileTest(t, path, []string{"achievements"}, &out)
	if code != 1 || out.Len() != 0 || !strings.Contains(stderr, "trainer create") {
		t.Fatal(code, out.String(), stderr)
	}
	if _, err := os.Stat(filepath.Dir(path)); !os.IsNotExist(err) {
		t.Fatal("created state", err)
	}
	out.Reset()
	code, stderr = executeProfileTest(t, path, []string{"achievements", "--help"}, &out)
	if code != 0 || stderr != "" || !strings.Contains(out.String(), "pokecrt trainer achievements") || strings.Contains(out.String(), "<trainer-name>") {
		t.Fatal(code, out.String(), stderr)
	}
}
