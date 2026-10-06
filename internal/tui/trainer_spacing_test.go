package tui

import (
	"github.com/charmbracelet/x/ansi"
	"strings"
	"testing"
)

func TestTrainerFactsKeepLabelValueSeparation(t *testing.T) {
	for _, size := range [][2]int{{40, 12}, {48, 20}, {64, 24}, {99, 40}, {100, 24}, {120, 40}, {190, 50}} {
		m := trainerReviewModel(size[0], size[1])
		timestamp := int64(1791288600000)
		m.activity.data.stats.FirstEncounterMS = &timestamp
		m.activity.data.stats.LastEncounterMS = &timestamp
		for _, mono := range []bool{false, true} {
			m.noColor = mono
			left, right := m.trainerInformationColumns()
			lines := append(left, right...)
			for _, label := range []string{"Shiny collections", "Shiny encounters", "First encounter", "Last encounter", "All discovered"} {
				found := false
				for _, line := range lines {
					plain := ansi.Strip(line)
					if !strings.HasPrefix(plain, label) {
						continue
					}
					found = true
					suffix := strings.TrimPrefix(plain, label)
					if m.dexWide() {
						if !strings.HasPrefix(suffix, "  ") {
							t.Fatalf("%v: %q lacks separation", size, plain)
						}
					} else if !strings.HasPrefix(suffix, ": ") {
						t.Fatalf("%v: missing compact separator", size)
					}
				}
				if !found {
					t.Fatalf("missing fact %q", label)
				}
			}
		}
	}
}

func TestTrainerFormFooterJoinsFrame(t *testing.T) {
	for _, size := range [][2]int{{40, 12}, {48, 13}, {40, 16}, {48, 20}, {64, 24}, {100, 24}, {120, 40}, {190, 50}} {
		for _, screen := range []setupScreen{createScreen, profilesScreen} {
			m := setupModel(t)
			m.width, m.height, m.screen = size[0], size[1], screen
			for _, focus := range []int{0, 1, 4, 100} {
				m.focus = focus
				for _, mono := range []bool{false, true} {
					m.noColor = mono
					rows := strings.Split(ansi.Strip(m.View().Content), "\n")
					found := false
					for _, row := range rows {
						start := strings.Index(row, "├")
						end := strings.Index(row, "┤")
						if start < 0 || end < 0 {
							continue
						}
						divider := row[start : end+len("┤")]
						if strings.Trim(divider, "├─┤") != "" {
							t.Fatal("divider contains gaps")
						}
						if !strings.Contains(rows[len(rows)-2], "[Enter]") && size[1] <= 24 {
							t.Fatal("footer placement")
						}
						found = true
					}
					if !found {
						t.Fatalf("%v screen %d focus %d: missing joined footer", size, screen, focus)
					}
				}
			}
		}
	}
}
