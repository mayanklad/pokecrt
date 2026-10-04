package tui

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/mayanklad/pokecrt/internal/storage"
	"github.com/mayanklad/pokecrt/internal/trainer"
)

func setupModel(t *testing.T) Model {
	t.Helper()
	t.Setenv("POKECRT_DATA_DIR", filepath.Join(t.TempDir(), "state"))
	m := New(context.Background(), LoadProfile, Dark, false)
	m.width, m.height = 80, 30
	m.actions = &profileActions{create: createProfile, use: useProfile}
	cmd := m.reload()
	m, _ = update(m, cmd())
	return m
}
func TestFirstRunCancelAndTypingShortcuts(t *testing.T) {
	m := setupModel(t)
	if m.screen != createScreen {
		t.Fatal("first run did not open setup")
	}
	for _, r := range "qajrké" {
		m, _ = update(m, tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	if string(m.name) != "qajrké" || m.settings || m.busy {
		t.Fatal("name shortcuts invoked actions")
	}
	m, _ = update(m, key(tea.KeyLeft))
	m, _ = update(m, key(tea.KeyBackspace))
	if string(m.name) != "qajré" {
		t.Fatal("Unicode cursor edit failed")
	}
	old := string(m.name)
	m, _ = update(m, tea.PasteMsg{Content: "bad\nname"})
	if string(m.name) != old || m.formError == "" {
		t.Fatal("unsafe paste accepted")
	}
	_, cmd := update(m, key(tea.KeyEscape))
	if cmd == nil {
		t.Fatal("cancel did not quit")
	}
	path, _ := storage.ResolvePath()
	if _, err := os.Stat(filepath.Dir(path)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("cancel initialized state")
	}
}
func TestCreateDuplicateAndSeparateUse(t *testing.T) {
	m := setupModel(t)
	m.insertName("  Ash  ")
	cmd := m.startProfile(true)
	if !m.busy || cmd == nil {
		t.Fatal("create not queued")
	}
	if m.startProfile(true) != nil {
		t.Fatal("double submit queued")
	}
	m, _ = update(m, cmd())
	if m.screen != mainScreen || !m.snapshot.Active || m.snapshot.Name != "Ash" {
		t.Fatalf("first create: %+v", m.snapshot)
	}
	m.openCreate()
	m.insertName("Misty")
	cmd = m.startProfile(true)
	m, _ = update(m, cmd())
	if m.screen != profilesScreen || m.snapshot.Name != "Ash" || !strings.Contains(m.notice, "Use trainer") || m.snapshot.Entries[m.selected].Name != "Misty" {
		t.Fatal("additional profile auto-activated or lost selection")
	}
	cmd = m.startProfile(false)
	m, _ = update(m, cmd())
	if m.snapshot.Name != "Misty" || m.screen != mainScreen {
		t.Fatal("explicit Use failed")
	}
	m.openCreate()
	m.insertName("misty")
	cmd = m.startProfile(true)
	m, _ = update(m, cmd())
	if !strings.Contains(m.formError, "already exists") || len(m.snapshot.Entries) != 2 {
		t.Fatal("duplicate name not recovered")
	}
	path, _ := storage.ResolvePath()
	repo, err := storage.ReadOnly(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	for _, table := range []string{"encounters", "species_discoveries", "variant_discoveries", "achievement_unlocks"} {
		var count int
		// These are fixed table names from the schema, not user input.
		if err = repo.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM "+table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatal("setup wrote gameplay state")
		}
	}
}
func TestCancelledSubmissionAndValidationLeaveStateAbsent(t *testing.T) {
	m := setupModel(t)
	if m.startProfile(true) != nil || m.formError == "" {
		t.Fatal("empty name not rejected")
	}
	m.insertName(strings.Repeat("é", 33))
	if m.startProfile(true) != nil {
		t.Fatal("overlong name queued")
	}
	m.name = []rune("Mayank")
	m.cursor = len(m.name)
	cmd := m.startProfile(true)
	_ = m.setupBack()
	m, _ = update(m, cmd())
	if m.busy || !strings.Contains(m.formError, "cancelled") {
		t.Fatal("cancel result lost")
	}
	path, _ := storage.ResolvePath()
	if _, err := os.Stat(filepath.Dir(path)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("cancelled command initialized paths")
	}
}
func TestBusyOperationResponsiveAndStaleReadRejected(t *testing.T) {
	m := New(context.Background(), func(context.Context) (Snapshot, error) { return Snapshot{}, nil }, Dark, false)
	m.width, m.height = 80, 30
	m.openCreate()
	m.insertName("Ash")
	started := make(chan struct{})
	m.actions = &profileActions{create: func(ctx context.Context, _ string) (trainer.Profile, error) {
		close(started)
		<-ctx.Done()
		return trainer.Profile{}, ctx.Err()
	}}
	staleID := m.loadID
	cmd := m.startProfile(true)
	result := make(chan tea.Msg, 1)
	go func() { result <- cmd() }()
	<-started
	m, _ = update(m, tea.WindowSizeMsg{Width: 40, Height: 12})
	m, _ = update(m, key(tea.KeyTab))
	if m.width != 40 || m.focus != 1 || !m.busy || m.reload() != nil {
		t.Fatal("busy state blocked navigation or raced a read")
	}
	m, _ = update(m, snapshotMsg{generation: staleID, snapshot: Snapshot{Name: "Stale", Active: true}})
	if m.snapshot.Name == "Stale" {
		t.Fatal("stale read accepted")
	}
	m, _ = update(m, key(tea.KeyEscape))
	select {
	case msg := <-result:
		m, _ = update(m, msg)
	case <-time.After(time.Second):
		t.Fatal("cancellation stuck")
	}
	if m.busy {
		t.Fatal("busy did not clear")
	}
	m, _ = update(m, profileResult{id: m.opID - 1, profile: trainer.Profile{Name: "Wrong"}})
	if strings.Contains(m.notice, "Wrong") {
		t.Fatal("stale result accepted")
	}
}
func TestSetupFramesKeyboardAndMouseFit(t *testing.T) {
	for _, size := range [][2]int{{120, 40}, {190, 50}, {80, 24}, {48, 36}, {40, 12}, {32, 8}} {
		for _, screen := range []setupScreen{createScreen, profilesScreen} {
			m := New(context.Background(), nil, Dark, true)
			m.width, m.height = size[0], size[1]
			m.screen = screen
			for i := 0; i < 40; i++ {
				m.snapshot.Entries = append(m.snapshot.Entries, trainer.Profile{Name: "Trainer " + strings.Repeat("é", i%8), Active: i == 20})
			}
			m.selected = 20
			m.name = []rune("Long keyboard name")
			m.cursor = 6
			v := m.View()
			lines := strings.Split(v.Content, "\n")
			if len(lines) != size[1] {
				t.Fatal("frame height")
			}
			for _, line := range lines {
				if ansi.StringWidth(line) != size[0] {
					t.Fatalf("%v frame width", size)
				}
			}
			ids := map[int]bool{}
			for y := 0; y < size[1]; y++ {
				for x := 0; x < size[0]; x++ {
					cmd := v.OnMouse(tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft})
					if cmd != nil {
						id := 0
						switch message := cmd().(type) {
						case activateMsg:
							id = int(message)
						case nameCursorMsg:
							id = 0
						}
						if id >= 1000 {
							id = 0
						}
						ids[id] = true
					}
				}
			}
			if size[0] >= 40 && size[1] >= 12 {
				count := 7
				if screen == createScreen {
					count = 8
					for i := 0; i < 26; i++ {
						if !ids[100+i] {
							t.Fatalf("%v missing onscreen key %d", size, i)
						}
					}
				}
				for i := 0; i < count; i++ {
					if !ids[i] {
						t.Fatalf("%v screen %d missing %d", size, screen, i)
					}
				}
			} else {
				quit := 5
				if screen == createScreen {
					quit = 7
				}
				if !ids[quit] {
					t.Fatal("small quit missing")
				}
			}
			if strings.Contains(v.Content, "\x1b") {
				t.Fatal("NO_COLOR styling")
			}
		}
	}
	m := New(context.Background(), nil, Dark, false)
	m.openCreate()
	m.activateSetup(100)
	m.activateSetup(101)
	m.activateSetup(5)
	m.activateSetup(102)
	if string(m.name) != "ab c" {
		t.Fatal("mouse-only name input")
	}
	m.activateSetup(4)
	if string(m.name) != "ab " {
		t.Fatal("mouse backspace")
	}
}
func TestCorruptStateNotReplacedBySetup(t *testing.T) {
	m := setupModel(t)
	path, _ := storage.ResolvePath()
	os.MkdirAll(filepath.Dir(path), 0700)
	os.WriteFile(path, []byte("corrupt"), 0600)
	m.insertName("Ash")
	cmd := m.startProfile(true)
	m, _ = update(m, cmd())
	data, _ := os.ReadFile(path)
	if string(data) != "corrupt" || m.formError == "" || m.busy {
		t.Fatal("corrupt state overwritten or unusable")
	}
}

func TestMouseCanPlaceNameCursor(t *testing.T) {
	m := New(context.Background(), nil, Dark, false)
	m.width, m.height = 80, 30
	m.openCreate()
	m.insertName("ABCD")
	cmd := m.View().OnMouse(tea.MouseClickMsg{X: 7, Y: 8, Button: tea.MouseLeft})
	if cmd == nil {
		t.Fatal("field has no hit target")
	}
	m, _ = update(m, cmd())
	if m.cursor != 1 {
		t.Fatal("mouse cursor", m.cursor)
	}
	m.activateSetup(100)
	if string(m.name) != "AaBCD" {
		t.Fatal("middle edit", string(m.name))
	}
}
func TestProfilesWithoutActiveRouteAndRemainUnchangedUntilUse(t *testing.T) {
	m := setupModel(t)
	m.insertName("Ash")
	cmd := m.startProfile(true)
	m, _ = update(m, cmd())
	path, _ := storage.ResolvePath()
	repo, err := storage.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	err = repo.Write(context.Background(), func(tx *storage.Tx) error {
		_, e := tx.ExecContext(context.Background(), "UPDATE app_state SET active_trainer_id=NULL WHERE id=1")
		return e
	})
	repo.Close()
	if err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)
	m = New(context.Background(), LoadProfile, Dark, false)
	m.width, m.height = 80, 24
	m.actions = &profileActions{create: createProfile, use: useProfile}
	cmd = m.reload()
	m, _ = update(m, cmd())
	if m.screen != profilesScreen || m.snapshot.Active || len(m.snapshot.Entries) != 1 {
		t.Fatal("missing chooser route")
	}
	m, _ = update(m, key(tea.KeyDown))
	cmd = m.reload()
	m, _ = update(m, cmd())
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("chooser browsing mutated database")
	}
	cmd = m.startProfile(false)
	m, _ = update(m, cmd())
	if !m.snapshot.Active || m.snapshot.Name != "Ash" {
		t.Fatal("explicit activation failed")
	}
}

func TestArrowKeyboardGridAndExits(t *testing.T) {
	for _, size := range [][2]int{{120, 40}, {80, 24}, {40, 12}} {
		m := setupModel(t)
		m.width, m.height = size[0], size[1]
		m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyDown})
		if m.focus != 100 {
			t.Fatalf("Down did not enter letters: %d", m.focus)
		}
		m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyEnter})
		if string(m.name) != "a" || m.focus != 100 {
			t.Fatalf("letter activation lost grid: %q/%d", m.name, m.focus)
		}
		m.navigateSetup("right")
		m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyEnter})
		if string(m.name) != "ab" {
			t.Fatal(string(m.name))
		}
		for page := 0; page < 3; page++ {
			m.keyboardPage = page
			for i := range m.keyboardLetters() {
				m.focus = 100 + i
				m.navigateSetup("up")
				if m.focus >= 100 && m.focus != 100+i-10 {
					t.Fatal("wrong grid row")
				}
				m.focus = 100 + i
				m.navigateSetup("down")
				if i+10 >= len(m.keyboardLetters()) && m.focus >= 100 {
					t.Fatal("grid traps focus")
				}
			}
		}
		m.focus = 100
		m.navigateSetup("up")
		if m.focus != 0 {
			t.Fatal("cannot return to name")
		}
		seen := map[int]bool{}
		m.focus = 0
		for i := 0; i < 8+len(m.keyboardLetters()); i++ {
			seen[m.focus] = true
			m.moveFocus(1)
		}
		for i := range m.keyboardLetters() {
			if !seen[100+i] {
				t.Fatal("Tab skipped key")
			}
		}
	}
}
func TestTrainerListArrowsLeaveAtBoundaries(t *testing.T) {
	m := setupModel(t)
	m.screen = profilesScreen
	m.snapshot.Entries = []trainer.Profile{{Name: "Ash"}, {Name: "Misty"}}
	m.focus = 0
	m.selected = 0
	m.navigateSetup("down")
	if m.selected != 1 || m.focus != 0 {
		t.Fatal("did not select next trainer")
	}
	m.navigateSetup("down")
	if m.focus != 1 {
		t.Fatal("list trapped Down")
	}
	m.navigateSetup("right")
	if m.focus != 2 {
		t.Fatal("Use not reachable")
	}
	m.navigateSetup("up")
	if m.focus != 0 || m.selected != 1 {
		t.Fatal("did not return to selection")
	}
	m.selected = 0
	m.navigateSetup("up")
	if m.focus == 0 {
		t.Fatal("list trapped Up")
	}
	for _, direction := range []string{"left", "right"} {
		m.focus = 0
		m.navigateSetup(direction)
		if m.focus == 0 {
			t.Fatal("list trapped horizontal arrow")
		}
	}
}

func TestTrainerActionsFollowTwoColumns(t *testing.T) {
	m := setupModel(t)
	m.screen = profilesScreen
	for _, column := range [][]int{{1, 3, 6}, {2, 4, 5}} {
		for i := 0; i < 2; i++ {
			m.focus = column[i]
			m.navigateSetup("down")
			if m.focus != column[i+1] {
				t.Fatal("Down crossed columns")
			}
			m.navigateSetup("up")
			if m.focus != column[i] {
				t.Fatal("Up crossed columns")
			}
		}
	}
	for _, row := range [][2]int{{1, 2}, {3, 4}, {6, 5}} {
		m.focus = row[0]
		m.navigateSetup("right")
		if m.focus != row[1] {
			t.Fatal("Right changed rows")
		}
		m.navigateSetup("left")
		if m.focus != row[0] {
			t.Fatal("Left changed rows")
		}
	}
}

func TestHelpTracksFocusAndMouseActions(t *testing.T) {
	for _, size := range [][2]int{{120, 40}, {80, 24}, {40, 12}} {
		m := setupModel(t)
		m.width, m.height = size[0], size[1]
		if !strings.Contains(m.View().Content, "↓ K") {
			t.Fatal("name help missing")
		}
		m, _ = update(m, key(tea.KeyDown))
		if !strings.Contains(m.View().Content, "↑↓←→") || !strings.Contains(m.View().Content, "Enter Type") {
			t.Fatal("grid help did not change")
		}
		m.activateSetup(3)
		if m.focus != 3 || strings.Contains(m.View().Content, "Enter Type") {
			t.Fatal("mouse action kept grid help")
		}
		m.activateSetup(5)
		if m.focus != 5 {
			t.Fatal("Space lost button focus")
		}
		m.focus = 113
		m.openSettings()
		m, _ = update(m, key(tea.KeyEscape))
		if m.focus != 113 {
			t.Fatal("Appearance lost grid focus")
		}
		m.screen = profilesScreen
		m.focus = 0
		if !strings.Contains(m.View().Content, "Choose / leave") {
			t.Fatal("list help missing")
		}
		m.activateSetup(1) // New trainer, returns to form.
		m.screen = profilesScreen
		m.focus = 4
		m.openSettings()
		m.activate(5)
		if m.focus != 4 || !strings.Contains(m.View().Content, "Column") {
			t.Fatal("button help or return focus missing")
		}
	}
}
func TestUndersizedTerminalCannotActivateHiddenControls(t *testing.T) {
	for _, screen := range []setupScreen{mainScreen, createScreen, profilesScreen} {
		m := setupModel(t)
		m.screen = screen
		m.name = []rune("Ash")
		m.cursor = 3
		m.width, m.height = 32, 8
		for _, k := range []rune{tea.KeyDown, tea.KeyTab, 'a', 'r', 'x'} {
			var cmd tea.Cmd
			m, cmd = update(m, key(k))
			if cmd != nil || m.busy || m.settings {
				t.Fatal("hidden action ran")
			}
		}
		m, _ = update(m, tea.PasteMsg{Content: "Hidden"})
		if string(m.name) != "Ash" {
			t.Fatal("hidden typing")
		}
		_, cmd := update(m, key(tea.KeyEnter))
		if cmd == nil {
			t.Fatal("visible Quit not keyboard accessible")
		}
	}
}

func TestSetupAllModesPagesAndFocusFit(t *testing.T) {
	for _, size := range [][2]int{{190, 50}, {120, 40}, {80, 24}, {48, 36}, {40, 12}} {
		for _, a := range appearances {
			for _, nc := range []bool{false, true} {
				m := New(context.Background(), nil, a, nc)
				m.width, m.height = size[0], size[1]
				m.screen = createScreen
				for page := 0; page < 3; page++ {
					m.keyboardPage = page
					for i := range m.keyboardLetters() {
						m.focus = 100 + i
						v := m.View()
						lines := strings.Split(v.Content, "\n")
						if len(lines) != size[1] {
							t.Fatal("height")
						}
						for _, line := range lines {
							if ansi.StringWidth(line) != size[0] {
								t.Fatal("width")
							}
						}
						if !strings.Contains(v.Content, "Enter Type") || !strings.Contains(v.Content, "▶") {
							t.Fatal("missing focus/help")
						}
						if nc && strings.Contains(v.Content, "\x1b") {
							t.Fatal("NO_COLOR")
						}
					}
				}
			}
		}
	}
}
