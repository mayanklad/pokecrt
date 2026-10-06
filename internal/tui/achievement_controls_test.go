package tui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/mayanklad/pokecrt/internal/trainer"
	"strings"
	"testing"
)

func achievementControlsModel(w, h int) Model {
	m := activityModel()
	m.width, m.height, m.section = w, h, 3
	for range 30 {
		m.activity.data.achievements.Unlocked = append(m.activity.data.achievements.Unlocked, trainer.AchievementView{Name: "Earned badge", Description: "Completed goal"})
		m.activity.data.achievements.Locked = append(m.activity.data.achievements.Locked, trainer.AchievementView{Name: "Next goal", Description: "Future goal", HasTarget: true, Target: 100})
	}
	return m
}

func TestAchievementHorizontalControlOrder(t *testing.T) {
	for _, size := range [][2]int{{40, 12}, {48, 20}, {64, 24}, {80, 40}, {90, 24}, {100, 28}, {120, 40}} {
		m := achievementControlsModel(size[0], size[1])
		controls := m.activityControls()
		m.focus = controls[0].id
		for i := range len(controls) * 2 {
			m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyRight})
			if want := controls[(i+1)%len(controls)].id; m.focus != want {
				t.Fatalf("%v right: got %d want %d", size, m.focus, want)
			}
		}
		for i := range len(controls) * 2 {
			m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyLeft})
			if want := controls[(len(controls)-1-i%len(controls))%len(controls)].id; m.focus != want {
				t.Fatalf("%v left: got %d want %d", size, m.focus, want)
			}
		}
		if m.activity.scroll != 0 || m.activity.panelScroll != [2]int{} {
			t.Fatal("focus movement scrolled content")
		}
	}
}

func TestAchievementFooterAndFocusAtEverySize(t *testing.T) {
	for _, size := range [][2]int{{40, 12}, {48, 20}, {55, 24}, {56, 24}, {64, 24}, {80, 40}, {90, 20}, {90, 24}, {100, 28}, {120, 40}} {
		m := achievementControlsModel(size[0], size[1])
		for _, control := range m.activityControls() {
			m.focus = control.id
			content := ansi.Strip(m.View().Content)
			for _, text := range []string{"Refresh", "[Tab] Focus", "[Enter] Select", "[Esc] Back", "Move"} {
				if !strings.Contains(content, text) {
					t.Fatalf("%v focus %d missing %q", size, m.focus, text)
				}
			}
			if strings.Count(content, "▶") != 1 {
				t.Fatalf("%v focus %d: expected one marker", size, m.focus)
			}
			if strings.Contains(content, "▶ ACHIEVEMENT") || strings.Contains(content, "▶ EARNED") || strings.Contains(content, "▶ NEXT GOALS") {
				t.Fatal("duplicate title focus")
			}
			rows := strings.Split(content, "\n")
			if len(rows) != size[1] {
				t.Fatal("wrong height")
			}
			for _, row := range rows {
				if ansi.StringWidth(row) != size[0] {
					t.Fatal("wrong width")
				}
			}
			if control.y >= size[1]-3 {
				t.Fatalf("footer overlaps hints %v %+v", size, control)
			}
			if control.id == 22 || control.id == 16 || control.id == 17 || control.id == 40 || control.id == 42 || control.id == 43 {
				if m.achievementPanelStyle(0) != m.palette().muted+"\x1b[1m" {
					t.Fatal("active frame not highlighted")
				}
			}
		}
	}
}

func TestAchievementArrowActivationTargetsPanel(t *testing.T) {
	m := achievementControlsModel(120, 40)
	for _, id := range []int{43, 45, 42, 44} {
		m.activateActivity(id)
	}
	if m.activity.panelScroll != [2]int{} {
		t.Fatal("arrow buttons did not independently increment/decrement")
	}
	m.activateActivity(45)
	if m.activity.panelScroll != [2]int{0, 1} {
		t.Fatal("right arrow moved left panel")
	}
}

func TestAchievementNoColorHasOneFocusMarker(t *testing.T) {
	for _, size := range [][2]int{{40, 12}, {56, 24}, {80, 40}, {120, 40}} {
		m := achievementControlsModel(size[0], size[1])
		m.noColor = true
		for _, control := range m.activityControls() {
			m.focus = control.id
			content := m.View().Content
			if strings.Count(content, "▶") != 1 || strings.Contains(content, "\x1b[") {
				t.Fatalf("%v focus %d: ambiguous monochrome focus", size, m.focus)
			}
		}
	}
}

func TestAchievementControlsClearJournalBorder(t *testing.T) {
	for _, size := range [][2]int{{40, 12}, {64, 20}, {80, 24}} {
		m := achievementControlsModel(size[0], size[1])
		m.focus = 22
		rows := strings.Split(ansi.Strip(m.View().Content), "\n")
		for _, control := range m.activityControls() {
			if control.id != 22 && control.id != 16 && control.id != 17 {
				continue
			}
			text := []rune(rows[control.y])[control.x : control.x+control.w]
			if text[0] != '❨' || text[len(text)-1] != '❩' || strings.ContainsRune(string(text), '─') {
				t.Fatalf("%v control %d has damaged delimiters/border: %q", size, control.id, string(text))
			}
		}
	}
}

func TestAchievementHintsUseOneRowWhenTheyFit(t *testing.T) {
	for _, width := range []int{40, 64, 80, 120, 180} {
		m := achievementControlsModel(width, 40)
		m.focus = m.activityControls()[0].id
		c := newCanvas(width, 6)
		m.paintAchievementHints(c, 2, 0, width-4)
		rows := strings.Split(ansi.Strip(c.content(m.palette())), "\n")
		if width >= 80 {
			if !strings.Contains(rows[4], "[Enter] Select") || !strings.Contains(rows[3], "─") {
				t.Fatal("wide hints not bottom anchored", width)
			}
		} else if !strings.Contains(rows[4], "[Enter] Select") || !strings.Contains(rows[2], "─") {
			t.Fatal("narrow hints not bottom anchored", width)
		}
		m.focus = 12
		c = newCanvas(width, 6)
		m.paintAchievementHints(c, 2, 0, width-4)
		content := ansi.Strip(c.content(m.palette()))
		if !strings.Contains(content, "[↑↓←→] Move") || strings.Count(content, "Move") != 1 {
			t.Fatal("redundant move hint")
		}
	}
}
