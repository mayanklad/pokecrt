package tui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/mayanklad/pokecrt/internal/trainer"
	"strings"
	"testing"
)

func TestPokedexArrowRoutesReachEveryVisibleControl(t *testing.T) {
	sizes := [][2]int{{40, 12}, {40, 16}, {48, 20}, {56, 24}, {64, 24}, {99, 24}, {100, 24}, {120, 40}, {190, 50}}
	for _, size := range sizes {
		for view := 0; view < 5; view++ {
			for _, filters := range []bool{false, true} {
				m := dexModel(t, collectedDex())
				m.dex.query = "charizard"
				m = applyEntry(t, m, m.filterDex())
				m = applyEntry(t, m, m.activateDex(3001))
				m.width, m.height = size[0], size[1]
				m.dex.detail = view > 0
				m.dex.tab = max(0, view-1)
				m.dex.filters = filters
				order := m.dexFocusOrder()
				allowed := map[int]bool{}
				for _, id := range order {
					if allowed[id] {
						t.Fatal("duplicate Tab target", id)
					}
					allowed[id] = true
				}
				reached := map[int]bool{order[0]: true}
				queue := []int{order[0]}
				for len(queue) > 0 {
					id := queue[0]
					queue = queue[1:]
					for _, code := range []rune{tea.KeyUp, tea.KeyDown, tea.KeyLeft, tea.KeyRight} {
						next := m
						next.focus = id
						next.dex.scroll = 0
						next.dex.factsScroll = 0
						next.revealDexFocus()
						next, _ = update(next, key(code))
						if next.dex.detail != m.dex.detail {
							continue
						}
						if !allowed[next.focus] {
							t.Fatalf("hidden target %v view%d filters%v %d %v -> %d", size, view, filters, id, code, next.focus)
						}
						if !reached[next.focus] {
							reached[next.focus] = true
							queue = append(queue, next.focus)
						}
					}
				}
				for _, id := range order {
					if !reached[id] {
						t.Fatalf("unreachable %v view%d filters%v: %d reached%v", size, view, filters, id, reached)
					}
				}
			}
		}
	}
}
func TestPokedexOutlinedControlsReserveTheirFullHeight(t *testing.T) {
	for _, size := range [][2]int{{56, 24}, {64, 24}, {80, 24}, {99, 24}, {100, 24}, {120, 40}} {
		for view := 0; view < 5; view++ {
			m := dexModel(t, collectedDex())
			m.width, m.height = size[0], size[1]
			m.dex.detail = view > 0
			m.dex.tab = max(0, view-1)
			g := m.dexGeometry()
			bottom := g.bodyY + g.bodyH - 1
			for _, control := range m.dexControls() {
				outlined := control.y == m.pokedexFooterY() || pokedexTab(control.id)
				if outlined && control.y-1 <= bottom {
					t.Fatalf("frame overlaps outlined control %v view%d bottom%d %+v", size, view, bottom, control)
				}
			}
		}
	}
}
func TestPokedexFrameHighlightBelongsToItsControls(t *testing.T) {
	m := dexModel(t, collectedDex())
	m.dex.query = "charizard"
	m = applyEntry(t, m, m.filterDex())
	m = applyEntry(t, m, m.activateDex(3001))
	for _, size := range [][2]int{{40, 12}, {56, 24}, {100, 24}, {120, 40}} {
		m.width, m.height = size[0], size[1]
		m.dex.detail = true
		for tab := 0; tab < 4; tab++ {
			m.dex.tab = tab
			for _, id := range m.dexFocusOrder() {
				m.focus = id
				m.revealDexFocus()
				c := newCanvas(m.width, m.height)
				m.paintDex(c)
				m.styleFrames(c)
				x, y, w, h, ok := m.pokedexFocusedFrame()
				if !ok {
					continue
				}
				_, active := m.frameStyles()
				if c.rows[y][x].style != active || c.rows[y+h-1][x+w-1].style != active {
					t.Fatalf("wrong frame focus %v tab%d id%d", size, tab, id)
				}
			}
		}
	}
}
func TestPokedexEvolutionHasCardsAndSelectionScroll(t *testing.T) {
	records := collectedDex()
	records.Species[133] = trainer.Discovery{Count: 1}
	for _, size := range [][2]int{{40, 12}, {64, 24}, {120, 40}, {190, 50}} {
		m := dexModel(t, records)
		m.dex.query = "133"
		m = applyEntry(t, m, m.filterDex())
		m.width, m.height = size[0], size[1]
		m.dex.detail = true
		m.dex.tab = 2
		m.focus = m.dexContentFocus()
		if m.focus < 2000 {
			t.Fatal("evolution does not enter card")
		}
		m.revealDexFocus()
		order := m.dexFocusOrder()
		seen := map[int]bool{}
		for _, id := range order {
			if seen[id] {
				t.Fatal("duplicate card target")
			}
			seen[id] = true
		}
		for _, node := range m.dex.entry.Evolution {
			m.focus = 2000 + node.Number
			m.revealDexFocus()
			v := ansi.Strip(m.View().Content)
			if !strings.Contains(v, "▶") {
				t.Fatalf("hidden focused card at%v node%d", size, node.Number)
			}
		}
	}
}
func TestPokedexRoundedLabelsDoNotChangeWhenFocused(t *testing.T) {
	m := dexModel(t, collectedDex())
	m.width, m.height = 40, 20
	m.dex.detail = true
	for _, label := range []string{"Art", "Facts", "Index", "Filters", "Refresh"} {
		control := dexControl{x: 2, y: 2, w: ansi.StringWidth(label) + 4, id: 26, label: label}
		for _, focused := range []bool{false, true} {
			m.focus = -1
			if focused {
				m.focus = 26
			}
			c := newCanvas(40, 20)
			c.pokedexButton(control, m, true)
			row := ""
			for _, cell := range c.rows[2] {
				row += cell.text
			}
			if !strings.Contains(row, label) {
				t.Fatal("focus truncates label", label, row)
			}
			if c.rows[2][2].text != "❨" || c.rows[2][2+control.w-1].text != "❩" {
				t.Fatal("focus replaces delimiter")
			}
		}
	}
}

func TestPokedexResizeKeepsScrollAndDetailsFocus(t *testing.T) {
	for _, id := range []int{18, 11, 12, 13, 14, 40, 41, 42} {
		for _, overlay := range []bool{false, true} {
			m := dexModel(t, collectedDex())
			m.dex.query = "charizard"
			m = applyEntry(t, m, m.filterDex())
			m = applyEntry(t, m, m.activateDex(3001))
			m.width, m.height = 120, 40
			m.dex.detail = false
			m.focus = id
			m.dex.factsScroll = 2
			if overlay {
				m.openSettings()
			}
			m, _ = update(m, tea.WindowSizeMsg{Width: 40, Height: 12})
			focus := m.focus
			if overlay {
				focus = m.settingsReturnFocus
			}
			if !m.dex.detail {
				t.Fatalf("resize hid entry for%d", id)
			}
			found := false
			for _, control := range m.dexFocusOrder() {
				if focus == control {
					found = true
				}
			}
			if !found {
				t.Fatalf("hidden focus after resize%d ->%d", id, focus)
			}
			if id >= 40 && !m.dex.overviewFacts {
				t.Fatal("details focus became artwork")
			}
		}
	}
}
func TestPokedexBackMatchesCompactHierarchy(t *testing.T) {
	m := dexModel(t, collectedDex())
	m.width, m.height = 40, 12
	m.dex.detail = true
	m.dex.filters = true
	m.activateDex(6)
	if !m.dex.detail || m.dex.filters || m.screen != dexScreen {
		t.Fatal("Back did not close filters first")
	}
	m.activateDex(6)
	if m.dex.detail || m.screen != dexScreen || m.focus != 0 {
		t.Fatal("Back did not return to index")
	}
	m.activateDex(6)
	if m.screen != mainScreen {
		t.Fatal("Back did not return to Home")
	}
}

func TestPokedexSelectionScrollAndWideIndexSpace(t *testing.T) {
	for _, size := range [][2]int{{40, 12}, {64, 24}, {100, 24}, {120, 40}, {190, 50}} {
		m := dexModel(t, collectedDex())
		m.dex.query = "charizard"
		m = applyEntry(t, m, m.filterDex())
		m.width, m.height = size[0], size[1]
		m.dex.detail = true
		m.dex.tab = 1
		m.focus = 18
		m.dex.optionIndex = 0
		if len(m.dex.options) < 2 {
			t.Fatal("missing variants fixture")
		}
		m, _ = update(m, key(tea.KeyDown))
		if m.dex.optionIndex != 1 || m.focus != 18 {
			t.Fatalf("variant Scroll does not choose at %v", size)
		}
		m, _ = update(m, key(tea.KeyUp))
		if m.dex.optionIndex != 0 || m.focus != 18 {
			t.Fatal("variant reverse selection")
		}
		m.dex.tab = 2
		m.focus = 18
		m.ensureDexFocus()
		nodes := orderedEvolution(m.dex.entry.Evolution)
		m, _ = update(m, key(tea.KeyDown))
		if m.dexMaxScroll() > 0 && len(nodes) > 1 && (m.dex.cardNode != 2000+nodes[1].Number || m.focus != 18) {
			t.Fatalf("evolution Scroll does not choose at %v", size)
		}
		if m.dexWide() {
			g := m.dexGeometry()
			if g.listW >= 31 || g.listH <= g.bodyH {
				t.Fatal("wide index did not reclaim space")
			}
			if g.bodyY+g.listH-1 >= m.pokedexFooterY()-1 {
				t.Fatal("index overlaps outlined footer")
			}
		}
	}
}

func TestPokedexDetailsIndicatorsHaveLabelSpacing(t *testing.T) {
	m := dexModel(t, collectedDex())
	m.width, m.height = 80, 40
	m.dex.detail = true
	m.dex.overviewFacts = true
	m.focus = 26
	for _, c := range m.dexControls() {
		if c.id == 26 {
			canvas := newCanvas(m.width, m.height)
			canvas.pokedexButton(c, m, true)
			// The full label and both indicators must fit with padding reserved around them.
			if c.w < ansi.StringWidth(c.label)+7 {
				t.Fatal("crowded view indicators")
			}
		}
	}
}

func TestPokedexArtworkScrollPansAndExitsLimits(t *testing.T) {
	for _, size := range [][2]int{{40, 12}, {64, 24}, {100, 24}, {120, 40}} {
		m := dexModel(t, collectedDex())
		m.dex.query = "charizard"
		m = applyEntry(t, m, m.filterDex())
		m.width, m.height = size[0], size[1]
		m.dex.detail = true
		m.dex.tab = 0
		m.dex.artWidth = 250
		m.focus = 18
		m, _ = update(m, key(tea.KeyRight))
		if m.dex.horizontal != 5 || m.focus != 18 {
			t.Fatalf("Scroll cannot pan right at %v", size)
		}
		m, _ = update(m, key(tea.KeyLeft))
		if m.dex.horizontal != 0 || m.focus != 18 {
			t.Fatal("Scroll cannot pan left")
		}
		m, _ = update(m, key(tea.KeyLeft))
		if m.focus == 18 {
			t.Fatal("left limit traps focus")
		}
		m.focus = 18
		m.dex.horizontal = m.dexMaxHorizontal()
		m, _ = update(m, key(tea.KeyRight))
		if m.focus == 18 {
			t.Fatal("right limit traps focus")
		}
	}
}
