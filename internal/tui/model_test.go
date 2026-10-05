package tui

import (
	"context"
	"errors"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/mayanklad/pokecrt/internal/storage"
	"github.com/mayanklad/pokecrt/internal/trainer"
)

func update(m Model, msg tea.Msg) (Model, tea.Cmd) {
	next, cmd := m.Update(msg)
	return next.(Model), cmd
}
func key(code rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: code} }

func TestArrowFocusAndExplicitActivation(t *testing.T) {
	m := New(context.Background(), nil, Dark, false)
	m.width, m.height = 100, 32
	m, _ = update(m, key(tea.KeyDown))
	if m.focus != 1 || m.section != 0 {
		t.Fatal("movement activated a section")
	}
	m, _ = update(m, key(tea.KeyEnter))
	if m.section != 1 {
		t.Fatal("Enter did not open focused section")
	}
	m, _ = update(m, key('a'))
	if !m.settings {
		t.Fatal("appearance did not open")
	}
	m, _ = update(m, key(tea.KeyDown))
	m, _ = update(m, key(tea.KeyEnter))
	if m.appearance != Light || !m.settings {
		t.Fatal("appearance not applied live")
	}
	m, _ = update(m, key(tea.KeyEscape))
	if m.settings || m.focus != 4 || m.section != 1 {
		t.Fatal("Back lost section/focus")
	}
}

func TestLiveFollowFallbackAndFixedModes(t *testing.T) {
	m := New(context.Background(), nil, FollowTerminal, false)
	cmd := m.probe()
	if cmd == nil || !m.probing {
		t.Fatal("missing asynchronous probe")
	}
	id := m.probeID
	m, _ = update(m, tea.BackgroundColorMsg{Color: color.White})
	if !m.backgroundKnown || m.terminalDark || m.palette().background != bg(246, 245, 237) {
		t.Fatal("did not follow light reply")
	}
	m, _ = update(m, probeTimeoutMsg(id))
	if !m.backgroundKnown {
		t.Fatal("completed probe timeout discarded a valid reply")
	}
	m, _ = update(m, tea.BackgroundColorMsg{Color: color.Black})
	if !m.terminalDark {
		t.Fatal("did not follow live dark reply")
	}
	m.appearance = Light
	m, _ = update(m, tea.BackgroundColorMsg{Color: color.Black})
	if m.palette().background != bg(246, 245, 237) {
		t.Fatal("terminal changed fixed light mode")
	}
	m.appearance = FollowTerminal
	m.backgroundKnown = false
	m.probing = false
	_ = m.probe()
	m, _ = update(m, probeTimeoutMsg(m.probeID))
	if m.probing || m.palette().background != "" || m.poll() != nil {
		t.Fatal("unsupported terminal keeps polling or paints a background")
	}
	m.appearance = TerminalNative
	if p := m.palette(); p.background != "" || p.foreground != "" {
		t.Fatal("native mode overrides defaults")
	}
}

func TestAsyncLoadAndStaleResult(t *testing.T) {
	called := 0
	m := New(context.Background(), func(context.Context) (Snapshot, error) { called++; return Snapshot{Name: "Trainer", Active: true}, nil }, Dark, false)
	cmd := m.reload()
	if called != 0 || !m.loading || cmd == nil {
		t.Fatal("load ran synchronously")
	}
	m, _ = update(m, key(tea.KeyDown))
	m, _ = update(m, tea.WindowSizeMsg{Width: 48, Height: 20})
	if m.focus != 1 || m.width != 48 || !m.loading {
		t.Fatal("work blocked navigation/resize")
	}
	m, _ = update(m, snapshotMsg{generation: m.loadID + 1, snapshot: Snapshot{Name: "Wrong"}})
	if !m.loading || m.snapshot.Name != "" {
		t.Fatal("stale result accepted")
	}
	m, _ = update(m, cmd())
	if called != 1 || m.loading || m.snapshot.Name != "Trainer" {
		t.Fatal("load result missing")
	}
	m.loading = true
	m.loadID++
	m, _ = update(m, snapshotMsg{generation: m.loadID, err: errors.New("bad\x1b[31m state")})
	if m.loading || m.snapshot.Active || strings.Contains(m.loadError, "\x1b") {
		t.Fatal("unsafe error state")
	}
}

func TestEveryFrameFitsAndEveryControlClickable(t *testing.T) {
	for _, size := range [][2]int{{190, 50}, {120, 40}, {48, 36}, {120, 36}, {80, 30}, {80, 24}, {60, 22}, {40, 20}, {40, 12}, {32, 8}} {
		for _, settings := range []bool{false, true} {
			for _, appearance := range appearances {
				m := New(context.Background(), nil, appearance, false)
				m.width, m.height = size[0], size[1]
				m.settings = settings
				v := m.View()
				lines := strings.Split(v.Content, "\n")
				if len(lines) != size[1] {
					t.Fatalf("%v: %d rows", size, len(lines))
				}
				for _, line := range lines {
					if ansi.StringWidth(line) != size[0] {
						t.Fatalf("%v width=%d", size, ansi.StringWidth(line))
					}
				}
				ids := map[int]bool{}
				for y := 0; y < size[1]; y++ {
					for x := 0; x < size[0]; x++ {
						cmd := v.OnMouse(tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft})
						if cmd != nil {
							ids[int(cmd().(activateMsg))] = true
						}
					}
				}
				if size[0] >= 40 && size[1] >= 12 {
					count := 7
					if settings {
						count = 7
					}
					for id := 0; id < count; id++ {
						if !ids[id] {
							t.Fatalf("%v settings=%v missing mouse target %d", size, settings, id)
						}
					}
				} else {
					id := 6
					if settings {
						id = 6
					}
					if !ids[id] {
						t.Fatal("small screen has no clickable Quit")
					}
				}
			}
		}
	}
}

func TestNoColorAndFocusVisible(t *testing.T) {
	for _, a := range appearances {
		for _, settings := range []bool{false, true} {
			m := New(context.Background(), nil, a, true)
			m.width, m.height = 120, 36
			m.settings = settings
			s := m.View().Content
			if strings.Contains(s, "\x1b") || !strings.Contains(s, "▶") {
				t.Fatal("NO_COLOR lost focus or contains styling")
			}
			if settings && !strings.Contains(s, "●") {
				t.Fatal("selected appearance invisible")
			}
		}
	}
}

func TestReadOnlyProfileLoadDoesNotCreateOrChangeState(t *testing.T) {
	ctx := context.Background()
	dir := filepath.Join(t.TempDir(), "state")
	t.Setenv("POKECRT_DATA_DIR", dir)
	s, err := LoadProfile(ctx)
	if err != nil || s.Active {
		t.Fatalf("missing state: %v", err)
	}
	if _, err = os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("load initialized state")
	}
	path, _ := storage.ResolvePath()
	repo, err := storage.Initialize(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	name, _ := trainer.ParseName("Keyboard Trainer")
	_, err = repo.CreateProfile(ctx, name, 1)
	if err != nil {
		t.Fatal(err)
	}
	repo.Close()
	before, _ := os.ReadFile(path)
	s, err = LoadProfile(ctx)
	if err != nil || s.Name != "Keyboard Trainer" || !s.Active {
		t.Fatalf("profile: %+v %v", s, err)
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("read changed database")
	}
	if err = os.WriteFile(path, []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = LoadProfile(ctx); err == nil {
		t.Fatal("corrupt storage accepted")
	}
	after, _ = os.ReadFile(path)
	if string(after) != "corrupt" {
		t.Fatal("corrupt database replaced")
	}
}

func BenchmarkWideView(b *testing.B) {
	m := New(context.Background(), nil, Dark, false)
	m.width, m.height = 120, 36
	b.ReportAllocs()
	for b.Loop() {
		_ = m.View()
	}
}

func TestTownArtworkFitsViewport(t *testing.T) {
	for _, size := range [][2]int{{120, 40}, {48, 36}, {48, 28}, {90, 28}} {
		m := New(context.Background(), nil, Dark, true)
		m.width, m.height = size[0], size[1]
		c := newCanvas(size[0]+4, size[1]+4)
		viewportH := size[1] - 15
		m.paintScene(c, 2, 2, size[0], viewportH)
		s := c.content(m.palette())
		if !strings.Contains(s, "ROUTE") || strings.Contains(s, "VERDANT TOWN") {
			t.Fatalf("unexpected first-run artwork at %v", size)
		}
		if !strings.Contains(s, "◆ YOU") {
			t.Fatalf("missing status at %v", size)
		}
		for y, row := range c.rows {
			for x, cell := range row {
				if (x < 2 || x >= size[0]+2 || y < 2 || y >= viewportH+2) && cell.text != " " {
					t.Fatalf("artwork escaped viewport at %v: (%d, %d)", size, x, y)
				}
			}
		}
	}
}

func TestTownLandmarksSurviveShrinkingViewport(t *testing.T) {
	for _, size := range [][2]int{{188, 37}, {120, 25}, {114, 17}, {114, 16}, {68, 17}, {67, 17}, {60, 14}, {44, 9}, {44, 8}, {44, 3}, {44, 2}, {44, 1}} {
		m := New(context.Background(), nil, Dark, true)
		c := newCanvas(size[0], size[1])
		m.paintScene(c, 0, 0, size[0], size[1])
		s := c.content(m.palette())
		for _, label := range []string{"HOME", "POND", "◆ YOU"} {
			if !strings.Contains(s, label) {
				t.Fatalf("missing %q at viewport %v", label, size)
			}
		}
	}
}
