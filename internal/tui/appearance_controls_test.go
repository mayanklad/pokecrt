package tui

import (
	tea "charm.land/bubbletea/v2"
	"context"
	"errors"
	"github.com/charmbracelet/x/ansi"
	"strings"
	"testing"
)

func TestAppearanceResponsiveControlsFocusAndSelection(t *testing.T) {
	for _, size := range [][2]int{{40, 12}, {40, 16}, {48, 20}, {55, 24}, {56, 24}, {64, 24}, {80, 28}, {99, 40}, {100, 24}, {120, 40}, {190, 50}} {
		for _, mode := range appearances {
			for _, mono := range []bool{false, true} {
				m := New(context.Background(), nil, mode, mono)
				m.width, m.height = size[0], size[1]
				m.openSettings()
				for _, control := range m.appearanceControls() {
					m.focus = control.id
					v := m.View()
					plain := ansi.Strip(v.Content)
					if strings.Count(plain, "▶") != 1 || strings.Count(plain, "●") != 1 {
						t.Fatalf("%v %s focus %d mono %v: ambiguous markers", size, mode, control.id, mono)
					}
					if mono && strings.Contains(v.Content, "\x1b[") {
						t.Fatal("NO_COLOR styling")
					}
					rows := strings.Split(plain, "\n")
					if len(rows) != size[1] {
						t.Fatal("height overflow")
					}
					for _, row := range rows {
						if ansi.StringWidth(row) != size[0] {
							t.Fatal("width overflow")
						}
					}
					for _, hint := range []string{"[Tab] Focus", "[Enter]", "[Esc] Back", "[Q] Quit"} {
						if !strings.Contains(plain, hint) {
							t.Fatalf("%v missing %s", size, hint)
						}
					}
					for _, c := range m.appearanceControls() {
						cmd := v.OnMouse(tea.MouseClickMsg{X: c.x + c.w/2, Y: c.y, Button: tea.MouseLeft})
						if cmd == nil || int(cmd().(activateMsg)) != c.id {
							t.Fatalf("%v missing mouse control %d", size, c.id)
						}
					}
					g := m.appearanceGeometry()
					if !strings.Contains(rows[g.y+g.h-2], "[Enter]") && !strings.Contains(rows[g.y+g.h-2], "[Q]") {
						t.Fatal("footer not bottom anchored")
					}
				}
			}
		}
	}
}

func TestAppearanceNavigationFollowsVisibleGroups(t *testing.T) {
	m := New(context.Background(), nil, Dark, false)
	m.width, m.height = 40, 12
	m.openSettings()
	cases := []struct {
		key  rune
		want int
	}{{tea.KeyUp, 6}, {tea.KeyDown, 0}, {tea.KeyDown, 1}, {tea.KeyDown, 2}, {tea.KeyDown, 3}, {tea.KeyDown, 5}, {tea.KeyRight, 4}, {tea.KeyRight, 90}, {tea.KeyUp, 3}, {tea.KeyLeft, 5}, {tea.KeyLeft, 3}, {tea.KeyRight, 90}}
	for _, tc := range cases {
		m, _ = update(m, key(tc.key))
		if m.focus != tc.want || m.appearance != Dark {
			t.Fatalf("focus %d want %d; movement applied mode", m.focus, tc.want)
		}
	}
	m.focus = 6
	for _, want := range []int{0, 1, 2, 3, 5, 4, 90, 6} {
		m, _ = update(m, key(tea.KeyTab))
		if m.focus != want {
			t.Fatal("Tab order", m.focus, want)
		}
	}
	m.focus = 1
	m, _ = update(m, key(tea.KeyEnter))
	if m.appearance != Light || m.focus != 1 {
		t.Fatal("live selection")
	}
}

func TestAppearanceRestoresEveryOriginAndRetainsPreview(t *testing.T) {
	for _, screen := range []setupScreen{mainScreen, createScreen, profilesScreen, dexScreen, dexSearchScreen, activityScreen} {
		m := New(context.Background(), nil, Dark, false)
		m.width, m.height, m.screen, m.focus = 120, 40, screen, 4
		if screen == activityScreen {
			m.section = 2
			m.focus = 13
		}
		if screen == createScreen {
			m.focus = 113
			m.name = []rune("Trainer")
		}
		original := m.focus
		m.openSettings()
		m.activate(1)
		m.activate(5)
		if m.settings || m.screen != screen || m.focus != original || m.appearance != Light {
			t.Fatalf("lost origin %d", screen)
		}
	}
}

func TestAppearanceHelpRetainsFullStatusAndReturnsToHelp(t *testing.T) {
	for _, size := range [][2]int{{40, 12}, {64, 24}, {120, 40}} {
		m := New(context.Background(), nil, Dark, false)
		m.width, m.height = size[0], size[1]
		m.openSettings()
		m.configError = strings.Repeat("Cannot save setting. ", 150) + "LAST DETAIL"
		m.activate(90)
		if m.compactMessage == "" || m.focus != 92 || !strings.Contains(m.compactMessage, "LAST DETAIL") {
			t.Fatal("full status inaccessible")
		}
		if m.compactMessageMaxScroll() == 0 {
			t.Fatal("long details not scrollable")
		}
		for range 100 {
			m, _ = update(m, key(tea.KeyPgDown))
		}
		if m.compactMessageScroll != m.compactMessageMaxScroll() || !strings.Contains(ansi.Strip(m.View().Content), strings.TrimSpace(ansi.Strip(m.compactMessageRows()[len(m.compactMessageRows())-1]))) {
			t.Fatal("help cannot reach final content")
		}
		m, _ = update(m, key(tea.KeyEscape))
		if m.compactMessage != "" || !m.settings || m.focus != 90 {
			t.Fatal("Help return lost")
		}
	}
	m := New(context.Background(), nil, Dark, false)
	m.width, m.height = 120, 40
	m.openSettings()
	m.activate(90)
	if m.compactMessageMaxScroll() == 0 {
		v := ansi.Strip(m.View().Content)
		if strings.Contains(v, "Scroll") || strings.Contains(v, "❨ ↑ ❩") {
			t.Fatal("unnecessary Help controls")
		}
	}
}

func TestAppearanceSaveFailureKeepsLiveModeAndReadableStatus(t *testing.T) {
	m := New(context.Background(), nil, Dark, false)
	m.width, m.height = 40, 12
	m.openSettings()
	m.configSave = func(context.Context, Appearance) error { return errors.New("blocked") }
	m.activate(1)
	cmd := m.activate(4)
	if cmd == nil || !m.savingConfig || m.focus != 4 {
		t.Fatal("save activation")
	}
	m, _ = update(m, cmd())
	if m.appearance != Light || m.savedAppearance != "" || !strings.Contains(ansi.Strip(m.View().Content), "See Help") {
		t.Fatal("save failure changed mode or hid error")
	}
}

func TestAppearanceHomeFocusAndPreviewDoNotWriteDefaults(t *testing.T) {
	for original := range 7 {
		m := New(context.Background(), nil, Dark, false)
		m.width, m.height, m.focus = 64, 24, original
		writes := 0
		m.configSave = func(context.Context, Appearance) error { writes++; return nil }
		m.savedAppearance = Dark
		m.openSettings()
		m.activate(1)
		m.activate(5)
		if m.focus != original || m.appearance != Light || m.savedAppearance != Dark || writes != 0 {
			t.Fatal("Back lost focus or preview wrote defaults")
		}
	}
}

func TestAppearanceHelpInlineMouseControlsAndResize(t *testing.T) {
	m := New(context.Background(), nil, Dark, false)
	m.width, m.height = 40, 12
	m.openSettings()
	m.activate(90)
	v := m.View()
	for _, tc := range []struct{ x, y, id int }{{3, 6, 95}, {26, 6, 93}, {32, 6, 94}, {2, 7, 92}} {
		cmd := v.OnMouse(tea.MouseClickMsg{X: tc.x, Y: tc.y, Button: tea.MouseLeft})
		if cmd == nil || int(cmd().(activateMsg)) != tc.id {
			t.Fatal("Help control has no mouse target", tc.id)
		}
	}
	m, _ = update(m, activateMsg(94))
	if m.compactMessageScroll != 1 || m.focus != 94 {
		t.Fatal("mouse scroll did not activate arrow")
	}
	m, _ = update(m, key(tea.KeyPgDown))
	if m.compactMessageScroll != 4 {
		t.Fatal("PageDown skipped more than one visible page")
	}
	m, _ = update(m, tea.WindowSizeMsg{Width: 120, Height: 40})
	if m.compactMessageMaxScroll() == 0 && (m.focus != 92 || m.compactMessageScroll != 0) {
		t.Fatal("resize retained hidden Help control")
	}
}
