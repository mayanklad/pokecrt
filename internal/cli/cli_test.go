package cli

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestRootOutput(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "root", want: rootHelp},
		{name: "long help", args: []string{"--help"}, want: rootHelp},
		{name: "short help", args: []string{"-h"}, want: rootHelp},
		{name: "explicit help", args: []string{"--help=true"}, want: rootHelp},
		{
			name: "version",
			args: []string{"--version"},
			want: "pokecrt dev\ndataset: unbundled\n",
		},
		{
			name: "explicit version",
			args: []string{"--version=true"},
			want: "pokecrt dev\ndataset: unbundled\n",
		},
		{name: "false version", args: []string{"--version=false"}, want: rootHelp},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			status := Run(test.args, &stdout, &stderr, "dev", "unbundled")

			if status != 0 {
				t.Fatalf("status = %d; stderr = %q", status, stderr.String())
			}
			if stdout.String() != test.want {
				t.Errorf("stdout = %q; want %q", stdout.String(), test.want)
			}
			if stderr.Len() != 0 {
				t.Errorf("unexpected stderr: %q", stderr.String())
			}
		})
	}
}

func TestInvalidInvocations(t *testing.T) {
	tests := [][]string{
		{"charizard"},
		{"print"},
		{"--unknown"},
		{"-version"},
		{"--version", "--version"},
		{"--version=false", "--version=true"},
		{"--help", "-h"},
		{"--help", "--version"},
		{"--version=invalid"},
		{"--version", "extra"},
		{"--", "extra"},
	}

	for _, args := range tests {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			status := Run(args, &stdout, &stderr, "dev", "unbundled")

			if status != 2 {
				t.Errorf("status = %d; want 2", status)
			}
			if stdout.Len() != 0 {
				t.Errorf("unexpected stdout: %q", stdout.String())
			}
			if !strings.Contains(stderr.String(), "pokecrt --help") {
				t.Errorf("missing usage guidance: %q", stderr.String())
			}
		})
	}
}

func TestRootCommandsDoNotCreateState(t *testing.T) {
	base := t.TempDir()
	dataDir := filepath.Join(base, "trainer-data")
	xdgDir := filepath.Join(base, "xdg-data")

	t.Setenv("POKECRT_DATA_DIR", dataDir)
	t.Setenv("XDG_DATA_HOME", xdgDir)

	for _, args := range [][]string{nil, {"--help"}, {"--version"}} {
		var stdout, stderr bytes.Buffer
		if status := Run(args, &stdout, &stderr, "dev", "unbundled"); status != 0 {
			t.Fatalf("args %v: status = %d; stderr = %q", args, status, stderr.String())
		}
	}

	for _, path := range []string{dataDir, xdgDir} {
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("state path %q: expected absent; got %v", path, err)
		}
	}
}

func TestVersionUsesBuildMetadata(t *testing.T) {
	var stdout, stderr bytes.Buffer

	status := Run(
		[]string{"--version"},
		&stdout,
		&stderr,
		"v0.1",
		"fixture-dataset",
	)

	if status != 0 {
		t.Fatalf("status = %d; stderr = %q", status, stderr.String())
	}
	if want := "pokecrt v0.1\ndataset: fixture-dataset\n"; stdout.String() != want {
		t.Errorf("stdout = %q; want %q", stdout.String(), want)
	}
}

type failingWriter struct {
	err error
}

func (w failingWriter) Write([]byte) (int, error) {
	return 0, w.err
}

func TestOutputFailures(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantError  bool
	}{
		{"broken pipe", syscall.EPIPE, 0, false},
		{"wrapped broken pipe", fmt.Errorf("write: %w", syscall.EPIPE), 0, false},
		{"other failure", errors.New("output unavailable"), 1, true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stderr bytes.Buffer

			status := Run(nil, failingWriter{test.err}, &stderr, "dev", "unbundled")

			if status != test.wantStatus {
				t.Errorf("status = %d; want %d", status, test.wantStatus)
			}
			if got := stderr.Len() != 0; got != test.wantError {
				t.Errorf("stderr = %q; want error = %v", stderr.String(), test.wantError)
			}
		})
	}
}
