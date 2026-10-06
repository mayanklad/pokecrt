package tui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"strings"
	"testing"
)

func TestPokedexFactLabelValueGap(t *testing.T) {
	for _, noColor := range []bool{false, true} {
		m := Model{width: 120, height: 40, noColor: noColor}
		for _, label := range []string{"Evolution stage", "First discovered", "Last encountered"} {
			text := ansi.Strip(m.informationFact(label, "VALUE", 36))
			if !strings.HasPrefix(text, label+"  ") || !strings.HasSuffix(text, "VALUE") {
				t.Fatalf("label/value separation lost: %q", text)
			}
		}
	}
}

func TestPokedexControlsDoNotOverlapAcrossModes(t *testing.T) {
	for _, size := range [][2]int{{40, 12}, {40, 16}, {40, 20}, {48, 23}, {56, 24}, {64, 24}, {80, 28}, {99, 40}, {100, 24}, {120, 40}, {190, 50}} {
		for tab := 0; tab < 4; tab++ {
			for _, filters := range []bool{false, true} {
				m := dexModel(t, collectedDex())
				m.noColor = true
				m.dex.query = "charizard"
				m = applyEntry(t, m, m.filterDex())
				m = applyEntry(t, m, m.activateDex(3001))
				m.width, m.height = size[0], size[1]
				m.dex.detail = true
				m.dex.tab = tab
				m.dex.filters = filters
				cs := m.dexControls()
				for i, a := range cs {
					if a.x < 1 || a.x+a.w > m.width-1 || a.y < 1 || a.y >= m.height-1 {
						t.Fatalf("out of bounds %v tab%d filter%v: %+v", size, tab, filters, a)
					}
					for _, b := range cs[i+1:] {
						if a.y == b.y && a.x < b.x+b.w && b.x < a.x+a.w {
							t.Fatalf("overlap %v tab%d filter%v: %+v %+v", size, tab, filters, a, b)
						}
					}
				}
				m.noColor = true
				text := m.View().Content
				if strings.Contains(text, "\x1b") {
					t.Fatal("mono color")
				}
				if !strings.Contains(text, "├") || !strings.Contains(text, "[Tab]") {
					t.Fatalf("missing joined footer %v", size)
				}
			}
		}
	}
}
func TestPokedexWheelOnlyChangesContentUnderPointer(t *testing.T) {
	m := dexModel(t, collectedDex())
	m.width, m.height = 120, 40
	selected := m.dex.selected
	for _, pos := range [][2]int{{5, 1}, {5, 3}, {5, 37}, {34, 8}, {119, 8}} {
		m, _ = update(m, tea.MouseWheelMsg{X: pos[0], Y: pos[1], Button: tea.MouseWheelDown})
		if m.dex.selected != selected {
			t.Fatal("wheel outside index moved selection", pos)
		}
	}
	m, _ = update(m, tea.MouseWheelMsg{X: 5, Y: 8, Button: tea.MouseWheelDown})
	if m.dex.selected != selected+1 {
		t.Fatal("index wheel did not browse")
	}
}
func TestPokedexFrameControlsTraverseBeforeFooter(t *testing.T) {
	for _, size := range [][2]int{{40, 12}, {64, 24}, {120, 40}} {
		m := dexModel(t, collectedDex())
		m.width, m.height = size[0], size[1]
		group := m.pokedexFrameGroup(19)
		m.focus = group[0]
		m.navigateDexControl("left")
		if m.focus != 6 {
			t.Fatal("left frame edge")
		}
		for i := 0; i < len(group)-1; i++ {
			m.focus = group[i]
			m.navigateDexControl("right")
			if m.focus != group[i+1] {
				t.Fatal("skipped index control")
			}
		}
		m.focus = group[len(group)-1]
		m.navigateDexControl("right")
		if m.focus != 8 {
			t.Fatal("right frame edge")
		}
	}
}
func TestPokedexPanLimitHasKeyboardExitAndPageUsesViewport(t *testing.T) {
	m := dexModel(t, collectedDex())
	m.dex.query = "charizard"
	m = applyEntry(t, m, m.filterDex())
	m = applyEntry(t, m, m.activateDex(3001))
	m.width, m.height = 40, 12
	m.focus = 10
	m.dex.horizontal = m.dexMaxHorizontal()
	m, _ = update(m, key(tea.KeyRight))
	if m.focus != 8 {
		t.Fatal("pan limit trapped focus")
	}
	m = dexModel(t, collectedDex())
	m.width, m.height = 40, 12
	m.focus = 0
	page := m.dexBodyHeight()
	m, _ = update(m, key(tea.KeyPgDown))
	if m.dex.selected != page {
		t.Fatal("page jump differs from viewport")
	}
}
func TestPokedexActiveTabsAndSearchFooter(t *testing.T) {
	for _, size := range [][2]int{{40, 12}, {40, 20}, {56, 24}, {120, 40}} {
		m := dexModel(t, collectedDex())
		m.width, m.height = size[0], size[1]
		m.dex.detail = true
		m.dex.tab = 1
		m.focus = m.dexTabFocus()
		view := ansi.Strip(m.View().Content)
		if m.height >= 20 && !strings.Contains(view, "●") {
			t.Fatal("active tab not indicated", size)
		}
		m.openDexSearch()
		view = ansi.Strip(m.View().Content)
		if !strings.Contains(view, "[Tab]") || !strings.Contains(view, "├") {
			t.Fatal("search footer inconsistent", size)
		}
		if !strings.Contains(view, "Search") {
			t.Fatal("search action missing", size)
		}
	}
}

func TestPokedexFieldNotesScrollIndependently(t *testing.T) {
	m := dexModel(t, collectedDex())
	m.dex.query = "charizard"
	m = applyEntry(t, m, m.filterDex())
	m = applyEntry(t, m, m.activateDex(3001))
	m.width, m.height = 100, 24
	if m.dexFactsMaxScroll() == 0 {
		t.Fatal("facts overflow not detected")
	}
	m.focus = 40
	artScroll, pan := m.dex.scroll, m.dex.horizontal
	m, _ = update(m, key(tea.KeyDown))
	if m.dex.factsScroll != 1 || m.dex.scroll != artScroll || m.dex.horizontal != pan {
		t.Fatal("facts moved artwork")
	}
	x, y, _, _ := m.dexFactsRect()
	m, _ = update(m, tea.MouseWheelMsg{X: x + 2, Y: y + 1, Button: tea.MouseWheelDown})
	if m.dex.factsScroll != 2 || m.dex.scroll != artScroll {
		t.Fatal("facts wheel moved artwork")
	}
	m.focus = 40
	m.navigateDexControl("right")
	if m.focus != 41 {
		t.Fatal("facts scroll skipped arrow")
	}
	m.navigateDexControl("right")
	if m.focus != 42 {
		t.Fatal("facts arrow skipped")
	}
	m.navigateDexControl("right")
	if m.focus != 8 {
		t.Fatal("facts edge exit")
	}
	m, _ = update(m, tea.WindowSizeMsg{Width: 190, Height: 50})
	if m.dexFactsMaxScroll() == 0 && m.dex.factsScroll != 0 {
		t.Fatal("facts resize did not clamp")
	}
}

func TestPokedexMinimumVariantLabelStaysVisible(t *testing.T) {
	m := dexModel(t, collectedDex())
	m.dex.query = "charizard"
	m = applyEntry(t, m, m.filterDex())
	m.width, m.height = 40, 12
	m.dex.detail = true
	m.dex.tab = 1
	m.focus = 15
	m.dex.optionIndex = 1
	m.revealDexOption()
	if !strings.Contains(ansi.Strip(m.View().Content), "▶") {
		t.Fatal("focused variant label outside viewport")
	}
}

func TestPokedexEveryVisibleControlHasOneFocusPointer(t *testing.T) {
	for _, size := range [][2]int{{40, 12}, {48, 20}, {56, 24}, {100, 24}, {120, 40}} {
		m := dexModel(t, collectedDex())
		m.noColor = true
		m.dex.query = "charizard"
		m = applyEntry(t, m, m.filterDex())
		m = applyEntry(t, m, m.activateDex(3001))
		m.width, m.height = size[0], size[1]
		m.dex.detail = true
		for tab := 0; tab < 4; tab++ {
			m.dex.tab = tab
			for _, control := range m.dexControls() {
				m.focus = control.id
				if count := strings.Count(ansi.Strip(m.View().Content), "▶"); count != 1 {
					t.Fatalf("%v tab%d focus%d: %d pointers", size, tab, control.id, count)
				}
			}
		}
	}
}
