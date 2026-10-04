package tui

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func configDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "config")
	t.Setenv("POKECRT_CONFIG_DIR", dir)
	return dir
}
func TestConfigReadMissingAndCancelledSaveDoNotCreatePaths(t *testing.T) {
	dir := configDir(t)
	a, err := loadAppearance(context.Background())
	if err != nil || a != "" {
		t.Fatal(a, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if !errors.Is(saveAppearance(ctx, Light), context.Canceled) {
		t.Fatal("cancelled save succeeded")
	}
	if _, err = os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("passive/cancelled settings created paths")
	}
	if err = saveAppearance(context.Background(), Appearance("invalid")); err == nil {
		t.Fatal("invalid saved")
	}
}
func TestConfigAtomicRoundTripAndProtectedMode(t *testing.T) {
	dir := configDir(t)
	for _, a := range appearances {
		if err := saveAppearance(context.Background(), a); err != nil {
			t.Fatal(err)
		}
		got, err := loadAppearance(context.Background())
		if err != nil || got != a {
			t.Fatal(got, err)
		}
		info, err := os.Stat(filepath.Join(dir, "appearance.conf"))
		if err != nil || info.Mode().Perm() != 0600 {
			t.Fatal("permissions")
		}
	}
	files, _ := os.ReadDir(dir)
	if len(files) != 1 {
		t.Fatal("temporary files leaked")
	}
	var group sync.WaitGroup
	errs := make(chan error, 20)
	for i := 0; i < 20; i++ {
		group.Add(1)
		go func(i int) { defer group.Done(); errs <- saveAppearance(context.Background(), appearances[i%4]) }(i)
	}
	group.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if _, err := loadAppearance(context.Background()); err != nil {
		t.Fatal("concurrent save left partial config", err)
	}
}
func TestMalformedAndFutureConfigStaySafe(t *testing.T) {
	dir := configDir(t)
	os.MkdirAll(dir, 0700)
	path := filepath.Join(dir, "appearance.conf")
	cases := []string{"bad", "version=1\nappearance=wrong\n", "version=1\nappearance=dark\nextra\n", strings.Repeat("x", 257)}
	for _, data := range cases {
		os.WriteFile(path, []byte(data), 0600)
		if _, err := loadAppearance(context.Background()); err == nil {
			t.Fatal("bad config accepted")
		}
		if err := saveAppearance(context.Background(), Light); err == nil {
			t.Fatal("invalid settings silently replaced")
		}
		got, _ := os.ReadFile(path)
		if string(got) != data {
			t.Fatal("read repaired file")
		}
	}
	future := "version=2\nappearance=dark\n"
	os.WriteFile(path, []byte(future), 0600)
	if !errors.Is(saveAppearance(context.Background(), Light), errSettingsVersion) {
		t.Fatal("future config overwritten")
	}
	got, _ := os.ReadFile(path)
	if string(got) != future {
		t.Fatal("future config changed")
	}
	os.Remove(path)
	target := filepath.Join(t.TempDir(), "target")
	os.WriteFile(target, []byte("target"), 0600)
	os.Symlink(target, path)
	if _, err := loadAppearance(context.Background()); err == nil {
		t.Fatal("symlink read accepted")
	}
	if err := saveAppearance(context.Background(), Dark); err == nil {
		t.Fatal("symlink write accepted")
	}
	got, _ = os.ReadFile(target)
	if string(got) != "target" {
		t.Fatal("symlink target changed")
	}
}
func TestSavedAppearancePrecedenceAndLateRead(t *testing.T) {
	m := New(context.Background(), nil, FollowTerminal, false)
	_ = m.applyConfig(configResult{appearance: Dark})
	if m.appearance != Dark {
		t.Fatal("saved mode not loaded")
	}
	m = New(context.Background(), nil, Light, false)
	m.appearanceOverride = true
	_ = m.applyConfig(configResult{appearance: Dark})
	if m.appearance != Light || m.savedAppearance != Dark {
		t.Fatal("flag precedence")
	}
	m = New(context.Background(), nil, FollowTerminal, false)
	m.openSettings()
	_ = m.activate(1)
	_ = m.applyConfig(configResult{appearance: Dark})
	if m.appearance != Light {
		t.Fatal("late read overrode user")
	}
	m.configSave = func(context.Context, Appearance) error { return nil }
	cmd := m.saveSettings()
	if cmd == nil || m.saveSettings() != nil {
		t.Fatal("save serialization")
	}
	_ = m.activate(0)
	m, _ = update(m, cmd())
	if m.appearance != Dark || m.savedAppearance != Light || m.savingConfig || !strings.Contains(m.configStatus(), "not saved") {
		t.Fatal("save result overwrote live selection")
	}
	_ = m.applyConfig(configResult{appearance: TerminalNative})
	if m.appearance != Dark || m.savedAppearance != Light {
		t.Fatal("late startup result replaced saved state")
	}
	m, _ = update(m, savedConfigMsg{appearance: Dark, err: errors.New("blocked\x1b[31m")})
	if strings.Contains(m.configError, "\x1b") || m.appearance != Dark {
		t.Fatal("save error unsafe")
	}
	m.screen = createScreen
	m.settings = true
	m.focus = 5
	m, _ = update(m, key(tea.KeyEnter))
	if m.settings || m.screen != createScreen || m.focus != 0 {
		t.Fatal("settings Back lost form")
	}
}
