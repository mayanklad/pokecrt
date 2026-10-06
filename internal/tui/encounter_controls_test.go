package tui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/mayanklad/pokecrt/internal/storage"
	"github.com/mayanklad/pokecrt/internal/trainer"
	"strings"
	"testing"
)

func encounterReviewModel(w, h int, history, art bool) Model {
	m := activityModel()
	m.width, m.height = w, h
	m.activity.historyMode = history
	if history {
		for range 30 {
			m.activity.data.history = append(m.activity.data.history, storage.HistoryEntry{Snapshot: trainer.Snapshot{SpeciesName: "Charizard", FormName: "Standard"}})
		}
	}
	if art {
		for range 60 {
			m.activity.art = append(m.activity.art, strings.Repeat("█", 100))
		}
	}
	return m
}
func TestEncounterResponsiveControlsAndFooter(t *testing.T) {
	for _, size := range [][2]int{{40, 12}, {40, 16}, {48, 20}, {64, 24}, {80, 40}, {100, 24}, {120, 40}} {
		for _, history := range []bool{false, true} {
			for _, art := range []bool{false, true} {
				m := encounterReviewModel(size[0], size[1], history, art)
				for _, mono := range []bool{false, true} {
					m.noColor = mono
					for _, control := range m.activityControls() {
						m.focus = control.id
						content := m.View().Content
						plain := ansi.Strip(content)
						if strings.Count(plain, "▶") != 1 {
							t.Fatalf("%v history=%v art=%v focus=%d: missing/duplicate focus", size, history, art, m.focus)
						}
						if mono && strings.Contains(content, "\x1b[") {
							t.Fatal("NO_COLOR styling leak")
						}
						for _, label := range []string{"[Tab] Focus", "[Enter]", "[Esc] Back", "Refresh", "Result", "History"} {
							if !strings.Contains(plain, label) {
								t.Fatalf("%v missing %q", size, label)
							}
						}
						if strings.Contains(plain, "Previous") || strings.Contains(plain, "Show result") || strings.Contains(plain, "Reload") {
							t.Fatal("obsolete controls")
						}
						if history && strings.Count(plain, "RECENT DISCOVERIES") != 1 {
							t.Fatal("duplicate/inconsistent history heading")
						}
						rows := strings.Split(plain, "\n")
						if len(rows) != size[1] {
							t.Fatal("render height")
						}
						for _, row := range rows {
							if ansi.StringWidth(row) != size[0] {
								t.Fatal("render width")
							}
						}
					}
				}
			}
		}
	}
}
func TestEncounterHistorySelectionAndMouseWheel(t *testing.T) {
	m := encounterReviewModel(120, 40, true, false)
	m.focus = 20
	m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyDown})
	if m.activity.selected != 1 {
		t.Fatal("scroll does not choose next entry")
	}
	m, _ = update(m, tea.MouseWheelMsg{Button: tea.MouseWheelDown, X: 20, Y: 10})
	if m.activity.selected != 2 {
		t.Fatal("wheel does not choose next entry")
	}
	m.activateActivity(31)
	if m.activity.selected != 1 {
		t.Fatal("up arrow selection")
	}
	m.activateActivity(32)
	if m.activity.selected != 2 {
		t.Fatal("down arrow selection")
	}
	m.activity.scroll = 3
	m.activateActivity(30)
	m.activateActivity(30)
	if m.activity.scroll != 3 {
		t.Fatal("reselecting History resets viewport")
	}
	if !m.activity.historyMode {
		t.Fatal("History toggles into result")
	}
	m.activateActivity(36)
	if m.activity.historyMode {
		t.Fatal("Result does not select result view")
	}
}
func TestEncounterArtControlsOnRightAndWheelTargetsArt(t *testing.T) {
	for _, size := range [][2]int{{40, 12}, {64, 24}, {120, 40}} {
		m := encounterReviewModel(size[0], size[1], false, true)
		artW := m.width - 4
		if m.dexWide() {
			artW = m.width/2 - 3
		}
		for _, c := range m.activityControls() {
			if c.id == 18 || c.id == 19 || c.id == 33 || c.id == 34 {
				if c.x < artW-24 {
					t.Fatal("art arrows not right aligned", size, c)
				}
			}
		}
		m, _ = update(m, tea.MouseWheelMsg{Button: tea.MouseWheelDown, X: 10, Y: 8})
		if m.activity.artScroll != 1 || m.activity.scroll != 0 {
			t.Fatal("wheel scrolled hidden details")
		}
	}
}
func TestEncounterNavigationFollowsControlRows(t *testing.T) {
	m := encounterReviewModel(120, 40, true, false)
	m.focus = 20
	for _, want := range []int{37, 31, 32, 14} {
		m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyRight})
		if m.focus != want {
			t.Fatalf("right got %d want %d", m.focus, want)
		}
	}
	m.focus = 20
	m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyLeft})
	if m.focus != 12 {
		t.Fatal("left edge does not enter footer")
	}
	m.focus = 10
	m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyRight})
	if m.focus != 36 {
		t.Fatal("Encounter right must enter Result")
	}
	m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyRight})
	if m.focus != 30 {
		t.Fatal("Result right must enter History")
	}
}

func TestEncounterDownEntersVisibleContent(t *testing.T) {
	for _, test := range []struct {
		history, art bool
		want         int
	}{{false, false, 12}, {true, false, 20}, {false, true, 21}} {
		m := encounterReviewModel(120, 40, test.history, test.art)
		m.focus = 30
		m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyDown})
		if m.focus != test.want {
			t.Fatalf("Down reached %d want %d", m.focus, test.want)
		}
	}
}

func TestEncounterFrameCornersAndActiveTabs(t *testing.T) {
	for _, size := range [][2]int{{40, 12}, {64, 20}, {64, 24}, {100, 24}, {120, 40}} {
		for _, history := range []bool{false, true} {
			m := encounterReviewModel(size[0], size[1], history, !history)
			c := newCanvas(m.width, m.height)
			m.paintEncounter(c)
			g := m.encounterGeometry()
			bottom := g.bodyY + g.bodyH - 1
			corners := []int{2, m.width - 3}
			if m.dexWide() && !history {
				corners = []int{2, m.width/2 - 2, m.width / 2, m.width - 3}
			}
			for i, x := range corners {
				want := "╰"
				if i%2 == 1 {
					want = "╯"
				}
				if c.rows[bottom][x].text != want {
					t.Fatalf("%v history=%v missing corner at %d", size, history, x)
				}
			}
			selected := 36
			if history {
				selected = 30
			}
			for _, control := range m.encounterControls() {
				if control.id != selected {
					continue
				}
				var label strings.Builder
				for x := control.x; x < control.x+control.w; x++ {
					label.WriteString(c.rows[control.y][x].text)
				}
				if !strings.Contains(label.String(), "●") {
					t.Fatalf("%v active tab has no dot: %q", size, label.String())
				}
			}
		}
	}
}
