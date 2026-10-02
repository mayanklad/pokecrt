package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/mayanklad/pokecrt/internal/storage"
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
		{[]string{"create", "--help"}, 0}, {[]string{"list", "--help"}, 0}, {[]string{"use", "-h"}, 0},
		{[]string{"create"}, 2}, {[]string{"use"}, 2},
		{[]string{"create", "Oak", "Extra"}, 2}, {[]string{"list", "Oak"}, 2},
		{[]string{"create", "Oak", "--help"}, 2}, {[]string{"use", "Oak", "-h"}, 2},
		{[]string{"--name"}, 2}, {[]string{"--name="}, 2},
		{[]string{"--name", "Oak", "--name", "Alice"}, 2},
		{[]string{"--help", "-h"}, 2}, {[]string{"--help", "--unknown"}, 2},
		{[]string{"-name", "Oak"}, 2}, {[]string{"create", "--name", "Oak"}, 2},
		{[]string{"create", ""}, 2}, {[]string{"create", "A\nB"}, 2},
		{[]string{"create", strings.Repeat("雪", 33)}, 2},
		{[]string{"delete", "Oak"}, 2}, {[]string{"show"}, 2}, {[]string{"achievements"}, 2},
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
