package tui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/mayanklad/pokecrt/internal/trainer"
	"strings"
	"testing"
)

func TestCompactDexReservedRowsAndControls(t *testing.T) {
	for _, size := range [][2]int{{40, 12}, {48, 19}, {48, 20}, {48, 23}, {48, 24}, {64, 28}, {99, 40}} {
		m := dexModel(t, collectedDex())
		m.width, m.height = size[0], size[1]
		for _, detail := range []bool{false, true} {
			for tab := 0; tab < 4; tab++ {
				m.dex.filters = detail && tab%2 == 1
				m.dex.detail, m.dex.tab = detail, tab
				g := m.dexGeometry()
				firstAction := g.h - 4
				if detail && g.h >= 24 {
					firstAction = g.h - 5
				}
				if g.bodyY+g.bodyH-1 >= firstAction {
					t.Fatalf("content overlaps actions at %v detail=%v tab=%d", size, detail, tab)
				}
				seen := map[int]bool{}
				for _, c := range m.dexControls() {
					if seen[c.id] {
						t.Fatalf("duplicate control %d", c.id)
					}
					seen[c.id] = true
					if c.x < 0 || c.y < 0 || c.x+c.w > m.width || c.y >= m.height-2 {
						t.Fatalf("unbounded control at %v: %+v", size, c)
					}
				}
				for _, line := range strings.Split(m.View().Content, "\n") {
					if ansi.StringWidth(line) > m.width {
						t.Fatal("render overflow", size)
					}
				}
			}
		}
	}
}

func TestCompactFactsAreScrollableAndDelimited(t *testing.T) {
	m := dexModel(t, collectedDex())
	m.dex.query = "charizard"
	m = applyEntry(t, m, m.filterDex())
	m.width, m.height = 40, 12
	m.dex.detail = true
	m.activateDex(26)
	if !m.dex.overviewFacts || m.dexMaxScroll() == 0 {
		t.Fatal("facts inaccessible")
	}
	rows := m.compactEntryFacts()
	var text string
	for _, r := range rows {
		text += ansi.Strip(r.text) + "\n"
	}
	for _, fact := range []string{"Generation: 1", "Encounters: 1", "First discovered:", "Last encountered:"} {
		if !strings.Contains(text, fact) {
			t.Fatal("missing fact", fact, text)
		}
	}
	if m.dexMaxHorizontal() != 0 {
		t.Fatal("facts retain artwork pan")
	}
}

func TestCompactResizeRemapsHiddenFilterAndClampsScroll(t *testing.T) {
	m := dexModel(t, collectedDex())
	m.focus = 32
	m.dex.scroll = 99999
	m.dex.horizontal = 99999
	m, _ = update(m, tea.WindowSizeMsg{Width: 48, Height: 24})
	if m.focus != 2 || m.dex.scroll > m.dexMaxScroll() || m.dex.horizontal > m.dexMaxHorizontal() {
		t.Fatal("resize leaves stale state", m.focus, m.dex.scroll, m.dex.horizontal)
	}
	m.openSettings()
	m.settingsReturnFocus = 32
	m.width = 120
	m.height = 40
	m, _ = update(m, tea.WindowSizeMsg{Width: 48, Height: 24})
	if m.settingsReturnFocus != 2 {
		t.Fatal("overlay return focus stale")
	}
}

func TestCompactEncounterControlsMatchDisplayedView(t *testing.T) {
	m := activityModel()
	m.width, m.height = 40, 12
	m.activity.art = []string{strings.Repeat("x", 80)}
	for i := 0; i < 15; i++ {
		m.activity.art = append(m.activity.art, strings.Repeat("x", 80))
	}
	ids := func() map[int]bool {
		out := map[int]bool{}
		for _, c := range m.activityControls() {
			out[c.id] = true
		}
		return out
	}
	art := ids()
	if !art[18] || !art[19] || !art[33] || !art[34] || art[16] || art[17] {
		t.Fatal("art controls mismatch", art)
	}
	m.activity.resultDetails = true
	details := ids()
	if details[18] || details[19] || details[33] || details[34] {
		t.Fatal("hidden art controls", details)
	}
}

func TestCompactHelpPreservesFullErrorsAndReturnFocus(t *testing.T) {
	m := setupModel(t)
	m.width, m.height = 40, 12
	m.focus = 90
	m.formError = strings.Repeat("Long failure detail. ", 40)
	m.openCompactMessage()
	if !strings.Contains(m.compactMessage, m.formError) || m.compactMessageMaxScroll() == 0 {
		t.Fatal("error lost")
	}
	m, _ = update(m, key(tea.KeyPgDown))
	if m.compactMessageScroll == 0 {
		t.Fatal("help does not scroll")
	}
	m, _ = update(m, key(tea.KeyEscape))
	if m.compactMessage != "" || m.focus != 90 {
		t.Fatal("help return lost")
	}
}

func TestCompactNoColorButtonsShowFocusWithoutPadding(t *testing.T) {
	m := setupModel(t)
	m.width, m.height = 40, 12
	m.noColor = true
	m.focus = 4
	v := m.View().Content
	if !strings.Contains(v, "▶Backspace") || strings.Contains(v, "\x1b") {
		t.Fatal("compact focus invisible", v)
	}
}

func TestCompactEvolutionExitsOnlyToVisibleControls(t *testing.T) {
	for _, size := range [][2]int{{40, 12}, {48, 19}, {48, 23}, {64, 28}} {
		for _, direction := range []string{"left", "right", "up", "down"} {
			m := dexModel(t, collectedDex())
			m.dex.query = "charizard"
			m = applyEntry(t, m, m.filterDex())
			m.width, m.height = size[0], size[1]
			m.dex.detail = true
			m.dex.tab = 2
			m.focus = 2004
			m.navigateEvolution(direction)
			valid := false
			for _, id := range m.dexFocusOrder() {
				valid = valid || id == m.focus
			}
			if !valid {
				t.Fatal("invisible evolution exit", size, direction, m.focus)
			}
		}
	}
}

func TestCompactEmptySearchHasRecoveryOnly(t *testing.T) {
	m := dexModel(t, collectedDex())
	m.width, m.height = 40, 12
	m.dex.query = "no-such-pokemon"
	m.filterDex()
	if len(m.dex.rows) != 0 {
		t.Fatal("fixture not empty")
	}
	for _, c := range m.dexControls() {
		if c.id == 4 || c.id == 16 || c.id == 17 {
			t.Fatal("empty list exposes action", c)
		}
	}
	for _, id := range m.dexFocusOrder() {
		if id == 0 {
			t.Fatal("empty list focus")
		}
	}
	if m.focus != 1 {
		t.Fatal("recovery focus", m.focus)
	}
}

func TestCompactFamilyArtworkRejectsStaleNode(t *testing.T) {
	m := dexModel(t, collectedDex())
	m.dex.query = "charizard"
	m = applyEntry(t, m, m.filterDex())
	m.width, m.height = 40, 12
	m.dex.detail = true
	m.dex.tab = 2
	m.focus = 27
	m.dex.cardNode = 2001
	m = applyEntry(t, m, m.openFamilyArtwork())
	if m.dex.entry.Number != 4 || m.dex.entry.Seen || m.dex.art != "" {
		t.Fatal("stale family node or disclosure", m.dex.entry.Number)
	}
	if trimSpriteMargins("   \n  XX  \n   X  \n     ") != "XX\n X" {
		t.Fatal("transparent margins not removed")
	}
}

func TestCompactFullArtworkOpensCollectedFamilyAppearance(t *testing.T) {
	m := dexModel(t, collectedDex())
	m.dex.query = "charizard"
	m = applyEntry(t, m, m.filterDex())
	m.width, m.height = 40, 12
	m.dex.tab = 2
	m.dex.detail = true
	m.focus = 27
	m.dex.cardNode = 2006
	m = applyEntry(t, m, m.openFamilyArtwork())
	if m.dex.entry.ArtworkKey == nil || m.dex.entry.ArtworkKey.FormID != "mega-x" || m.dex.entry.ArtworkKey.Palette != "shiny" || m.dex.art == "" {
		t.Fatal("full art did not open displayed collected appearance")
	}
	if m.dex.familyOrigin != 6 || m.dex.familyNode != 2006 {
		t.Fatal("family return missing")
	}
}

func TestCompactProfileCapacityDoesNotDropAtHeightBreakpoint(t *testing.T) {
	m := setupModel(t)
	m.screen = profilesScreen
	m.width = 48
	for i := 0; i < 30; i++ {
		m.snapshot.Entries = append(m.snapshot.Entries, trainer.Profile{Name: "Trainer"})
	}
	previous := 0
	for h := 12; h <= 36; h++ {
		m.height = h
		c := newCanvas(m.width, h)
		m.paintProfiles(c)
		count := 0
		for _, hit := range c.hits {
			if hit.id >= 1000 {
				count++
			}
		}
		if count < previous {
			t.Fatal("growing terminal reduced profile capacity", h, previous, count)
		}
		previous = count
	}
}
func TestCompactEncounterWheelScrollsVisibleArt(t *testing.T) {
	m := activityModel()
	m.width, m.height = 40, 12
	for i := 0; i < 20; i++ {
		m.activity.art = append(m.activity.art, "sprite")
	}
	m, _ = update(m, tea.MouseWheelMsg{Button: tea.MouseWheelDown, X: 10, Y: 6})
	if m.activity.artScroll != 1 || m.activity.scroll != 0 {
		t.Fatal("wheel changed hidden content")
	}
	m.activity.resultDetails = true
	m, _ = update(m, tea.MouseWheelMsg{Button: tea.MouseWheelDown, X: 10, Y: 6})
	if m.activity.artScroll != 1 {
		t.Fatal("details wheel changed hidden sprite")
	}
}
