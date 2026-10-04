package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTUIHelpAndRedirectDoNotTouchState(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "must-not-exist")
	t.Setenv("POKECRT_DATA_DIR", dir)
	configDir := filepath.Join(t.TempDir(), "config-must-not-exist")
	t.Setenv("POKECRT_CONFIG_DIR", configDir)
	for _, args := range [][]string{{"tui", "--help"}, {"tui", "-h"}, {"tui"}} {
		var out, err bytes.Buffer
		status := Run(args, &out, &err, "dev", "test")
		if len(args) > 1 {
			if status != 0 || out.String() != tuiHelp || err.Len() != 0 {
				t.Fatalf("help: %d %s", status, err.String())
			}
		} else {
			if status != 1 || out.Len() != 0 || !strings.Contains(err.String(), "terminal input and output") {
				t.Fatalf("redirect: %d %s", status, err.String())
			}
		}
	}
	if _, err := os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("tui help/redirection initialized data")
	}
	if _, err := os.Stat(configDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("help/redirection touched config")
	}
}
func TestTUIRejectsInvalidArguments(t *testing.T) {
	for _, args := range [][]string{{"tui", "extra"}, {"tui", "--unknown"}, {"tui", "--help", "-h"}, {"tui", "--appearance", "dark", "--appearance", "light"}, {"tui", "--appearance", "system"}, {"tui", "-appearance", "dark"}} {
		var out, err bytes.Buffer
		if status := Run(args, &out, &err, "dev", "test"); status != 2 || out.Len() != 0 {
			t.Fatalf("%v: status %d", args, status)
		}
	}
}
