package tui

import (
	tea "charm.land/bubbletea/v2"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/mayanklad/pokecrt/internal/catalog"
	"github.com/mayanklad/pokecrt/internal/trainer"
)

func TestFullTerminalGeometryAndRenderedExtent(t *testing.T) {
	m := dexModel(t, collectedDex())
	for _, size := range [][2]int{{120, 40}, {180, 60}, {320, 120}, {64, 28}, {40, 12}} {
		m.width, m.height = size[0], size[1]
		g := m.dexGeometry()
		if g.x != 0 || g.y != 0 || g.w != size[0] || g.h != size[1] {
			t.Fatalf("unused terminal extent: %+v", g)
		}
		for _, screen := range []setupScreen{mainScreen, dexScreen, activityScreen} {
			m.screen = screen
			m.section = 2
			rows := strings.Split(ansi.Strip(m.View().Content), "\n")
			if len(rows) != size[1] {
				t.Fatalf("screen %d: rendered %d of %d rows", screen, len(rows), size[1])
			}
			for _, row := range rows {
				if ansi.StringWidth(row) != size[0] {
					t.Fatalf("screen %d: incorrect row width", screen)
				}
			}
		}
	}
}
func TestVisualEvolutionOnlyUsesCollectedArtwork(t *testing.T) {
	m := dexModel(t, collectedDex())
	m.dex.query = "charizard"
	m = applyEntry(t, m, m.filterDex())
	if m.dex.family[6] == "" || m.dex.family[4] != "" || m.dex.family[5] != "" {
		t.Fatal("family artwork must follow exact collection disclosure")
	}
	m.width, m.height = 180, 60
	m.dex.tab = 2
	targets := map[int]bool{}
	for _, line := range m.evolutionLines() {
		for _, node := range line.nodes {
			targets[node.id] = true
		}
	}
	for _, id := range []int{2004, 2005, 2006} {
		if !targets[id] {
			t.Fatalf("missing clickable node %d", id)
		}
	}
	content := ansi.Strip(m.View().Content)
	if strings.Contains(content, "Charmander") || strings.Contains(content, "Charmeleon") {
		t.Fatal("undiscovered family identities leaked")
	}
	m.noColor = true
	m = applyEntry(t, m, m.loadDexEntry(catalog.Selection{Form: "mega-x", Shiny: true}))
	if strings.Contains(m.View().Content, "\x1b[") {
		t.Fatal("visual family does not respect NO_COLOR")
	}
}

func BenchmarkFullTerminalInformationView(b *testing.B) {
	m := activityModel()
	m.width, m.height = 180, 60
	m.section = 2
	b.ReportAllocs()
	for b.Loop() {
		_ = m.View()
	}
}

func TestVisualEvolutionOrdersBabiesBeforeDescendants(t *testing.T) {
	nodes := orderedEvolution([]trainer.DexNode{{Number: 25, Children: []int{26}}, {Number: 26}, {Number: 172, Children: []int{25}}})
	for i, id := range []int{172, 25, 26} {
		if nodes[i].Number != id {
			t.Fatal("family order follows Dex number instead of evolution links")
		}
	}
}

func TestEvolutionArrowEntryMovementAndLockedReturn(t *testing.T) {
	m := dexModel(t, collectedDex())
	m.dex.query = "charizard"
	m = applyEntry(t, m, m.filterDex())
	m.width, m.height = 180, 60
	m.dex.tab = 2
	m.focus = 0
	m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyRight})
	if m.focus != 2004 {
		t.Fatal("Right from index does not enter first family card", m.focus)
	}
	m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyRight})
	if m.focus != 2005 {
		t.Fatal("Right does not follow visible family row", m.focus)
	}
	m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyLeft})
	m = applyEntry(t, m, m.activateDex(m.focus))
	if m.dex.entry.Seen || m.dex.art != "" || m.dex.familyOrigin != 6 {
		t.Fatal("undiscovered node disclosure or return state is wrong")
	}
	m = applyEntry(t, m, m.activateDex(6))
	if m.dex.tab != 2 || m.dex.entry.Number != 6 || m.focus != 2004 {
		t.Fatal("Family does not return to original graph")
	}
}
func TestAchievementPanelsScrollIndependently(t *testing.T) {
	m := activityModel()
	m.width, m.height = 120, 28
	m.section = 3
	m.focus = 40
	for i := 0; i < 20; i++ {
		m.activity.data.achievements.Unlocked = append(m.activity.data.achievements.Unlocked, trainer.AchievementView{Name: "Earned", Description: "A completed goal"})
		m.activity.data.achievements.Locked = append(m.activity.data.achievements.Locked, trainer.AchievementView{Name: "Goal", Description: "A future goal", HasTarget: true, Target: 10})
	}
	m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyDown})
	if m.activity.panelScroll != [2]int{1, 0} {
		t.Fatal("left scroll moves right panel", m.activity.panelScroll)
	}
	m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyRight})
	m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyDown})
	if m.focus != 41 || m.activity.panelScroll != [2]int{1, 1} {
		t.Fatal("right panel not independently focusable", m.focus, m.activity.panelScroll)
	}
	m, _ = update(m, tea.MouseWheelMsg{Button: tea.MouseWheelDown, X: 10, Y: 10})
	if m.activity.panelScroll != [2]int{2, 1} {
		t.Fatal("mouse wheel does not target panel under pointer")
	}
}
func TestVariantCardsRemainSelectableAndRevealTheirFocus(t *testing.T) {
	m := dexModel(t, collectedDex())
	m.dex.query = "charizard"
	m = applyEntry(t, m, m.filterDex())
	m.height = 24
	m.dex.tab = 1
	m.focus = 15
	m.dex.optionIndex = len(m.dex.options) - 1
	m.revealDexOption()
	if m.dex.scroll == 0 || !strings.Contains(ansi.Strip(m.View().Content), "▶") {
		t.Fatal("focused appearance card not revealed")
	}
	m = applyEntry(t, m, m.activateDex(15))
	if m.dex.tab != 0 || m.dex.entry.ArtworkKey == nil || m.dex.entry.ArtworkKey.Palette != "shiny" {
		t.Fatal("card selection lost exact collected variant")
	}
}

func TestAchievementOutwardArrowsExitWithoutWrapping(t *testing.T) {
	m := activityModel()
	m.width, m.height = 120, 40
	m.section = 3
	m.focus = 40
	m.activity.panelScroll = [2]int{2, 3}
	m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyLeft})
	if m.focus != 12 || m.activity.panelScroll != [2]int{2, 3} {
		t.Fatal("Earned Badges Left must exit to Back")
	}
	m.focus = 41
	m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyRight})
	if m.focus != 14 || m.activity.panelScroll != [2]int{2, 3} {
		t.Fatal("Next Goals Right must exit to Quit")
	}
	m.focus = 40
	m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyRight})
	if m.focus != 41 {
		t.Fatal("inward Right must reach Next Goals")
	}
	m, _ = update(m, tea.KeyPressMsg{Code: tea.KeyLeft})
	if m.focus != 40 {
		t.Fatal("inward Left must reach Earned Badges")
	}
}
func TestEventAchievementHasEarnablePlayerFacingStatus(t *testing.T) {
	m := activityModel()
	m.section = 3
	m.activity.data.achievements.Locked = []trainer.AchievementView{{Name: "Regional Discovery", Description: "Encounter a regional form.", HasTarget: false}}
	_, lines := m.achievementInformationColumns()
	content := ansi.Strip(strings.Join(lines, "\n"))
	if !strings.Contains(content, "Not yet earned") || strings.Contains(content, "Unavailable") {
		t.Fatal("event goal incorrectly presented as unavailable")
	}
}

func TestEncounterReadyHeadingUsesFullLogWidth(t *testing.T) {
	for _, width := range []int{100, 120, 180} {
		m := activityModel()
		m.width = width
		rows := m.activityWrapped()
		if ansi.StringWidth(rows[0]) != m.dexGeometry().w-8 {
			t.Fatalf("ready divider width at %d: %d", width, ansi.StringWidth(rows[0]))
		}
		if len(rows) != 4 {
			t.Fatal("ready text unexpectedly wraps", rows)
		}
		m.activity.art = []string{"sprite"}
		if ansi.StringWidth(m.activityWrapped()[0]) != m.dexGeometry().w/2-6 {
			t.Fatal("split details width mismatched")
		}
	}
}
