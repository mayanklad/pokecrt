package tui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/mayanklad/pokecrt/internal/trainer"
	"strings"
	"testing"
)

func trainerReviewModel(w, h int) Model {
	m := activityModel()
	m.width, m.height, m.section, m.focus = w, h, 2, 15
	for i := 1; i <= 9; i++ {
		m.activity.data.stats.Generations = append(m.activity.data.stats.Generations, trainer.GenerationStatistics{Generation: i, Eligible: 2, EligibleTotal: 100})
	}
	return m
}
func TestTrainerResponsiveFooterAndFocus(t *testing.T) {
	for _, size := range [][2]int{{40, 12}, {40, 16}, {48, 20}, {56, 24}, {64, 24}, {80, 40}, {100, 20}, {100, 24}, {120, 40}} {
		m := trainerReviewModel(size[0], size[1])
		for _, mono := range []bool{false, true} {
			m.noColor = mono
			for _, control := range m.activityControls() {
				m.focus = control.id
				content := m.View().Content
				plain := ansi.Strip(content)
				if strings.Count(plain, "▶") != 1 {
					t.Fatalf("%v focus %d: ambiguous focus", size, control.id)
				}
				if mono && strings.Contains(content, "\x1b[") {
					t.Fatal("NO_COLOR leak")
				}
				for _, label := range []string{"Refresh", "[Tab] Focus", "[Enter] Select", "[Esc] Back"} {
					if !strings.Contains(plain, label) {
						t.Fatalf("%v missing %q", size, label)
					}
				}
				rows := strings.Split(plain, "\n")
				if len(rows) != size[1] {
					t.Fatal("rendered height")
				}
				for _, row := range rows {
					if ansi.StringWidth(row) != size[0] {
						t.Fatal("rendered width")
					}
				}
				if !strings.Contains(rows[size[1]-2], "[Enter] Select") {
					t.Fatal("footer not bottom anchored")
				}
			}
		}
	}
}
func TestTrainerPanelControlsAndChooserRoutes(t *testing.T) {
	for _, size := range [][2]int{{40, 12}, {64, 24}, {120, 40}} {
		m := trainerReviewModel(size[0], size[1])
		panes := []int{}
		for _, control := range m.activityControls() {
			if control.id == 22 || control.id == 16 || control.id == 17 || control.id >= 40 && control.id <= 45 {
				panes = append(panes, control.id)
			}
		}
		m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyDown})
		if len(panes) == 0 {
			if m.focus != 12 {
				t.Fatal("chooser Down must enter Back when content fits")
			}
			continue
		}
		if m.focus != panes[0] {
			t.Fatal("chooser Down must enter first Scroll")
		}
		for _, want := range append(panes[1:], 14) {
			m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyRight})
			if m.focus != want {
				t.Fatalf("%v right got %d want %d", size, m.focus, want)
			}
		}
		m.focus = panes[0]
		m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyLeft})
		if m.focus != 12 {
			t.Fatal("left pane edge must enter Back")
		}
		if m.activity.scroll != 0 || m.activity.panelScroll != [2]int{} {
			t.Fatal("navigation scrolled content")
		}
	}
}
