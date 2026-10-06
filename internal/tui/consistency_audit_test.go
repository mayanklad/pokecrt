package tui

import (
	tea "charm.land/bubbletea/v2"
	"testing"
)

func TestLockedEvolutionScrollDoesNotOpenPreviousCard(t *testing.T) {
	m := dexModel(t, collectedDex())
	m.width, m.height = 40, 12
	m.dex.detail = true
	m.dex.tab = 2
	m.dex.cardNode = 2006
	m.focus = 18
	if m.dex.entry.Seen || m.dexMaxScroll() == 0 {
		t.Fatal("fixture must be a scrolling undiscovered panel")
	}
	before := m.dex.entry.Number
	messages := []tea.Msg{key(tea.KeyEnter)}
	for _, control := range m.dexControls() {
		if control.id == 18 {
			click := m.View().OnMouse(tea.MouseClickMsg{X: control.x + control.w/2, Y: control.y, Button: tea.MouseLeft})
			if click == nil {
				t.Fatal("missing Scroll mouse target")
			}
			messages = append(messages, click())
		}
	}
	if len(messages) != 2 {
		t.Fatal("fixture lacks Scroll control")
	}
	for _, msg := range messages {
		next, cmd := update(m, msg)
		if cmd != nil || next.dex.entryLoading || next.dex.tab != 2 || next.dex.entry.Number != before {
			t.Fatal("locked Scroll activated a card from the previous entry")
		}
	}

}

func TestUndiscoveredPokedexArrowRoutesReachEveryVisibleControl(t *testing.T) {
	for _, size := range [][2]int{{40, 12}, {40, 16}, {48, 20}, {56, 24}, {99, 24}, {100, 24}, {120, 40}, {190, 60}} {
		for tab := 0; tab < 4; tab++ {
			m := dexModel(t, collectedDex())
			m.width, m.height = size[0], size[1]
			m.dex.detail = true
			m.dex.tab = tab
			order := m.dexFocusOrder()
			allowed := map[int]bool{}
			for _, id := range order {
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
					next, _ = update(next, key(code))
					if next.dex.detail != m.dex.detail {
						continue
					}
					if !allowed[next.focus] {
						t.Fatalf("%v tab%d: hidden target %d -> %d", size, tab, id, next.focus)
					}
					if !reached[next.focus] {
						reached[next.focus] = true
						queue = append(queue, next.focus)
					}
				}
			}
			for _, id := range order {
				if !reached[id] {
					t.Fatalf("%v tab%d: unreachable control %d", size, tab, id)
				}
			}
		}
	}
}

func TestActivityStateArrowRoutesAndMouseTargets(t *testing.T) {
	for _, size := range [][2]int{{40, 12}, {40, 16}, {47, 20}, {48, 20}, {55, 24}, {56, 24}, {99, 24}, {100, 24}, {120, 40}} {
		for section := 1; section <= 3; section++ {
			for state := 0; state < 4; state++ {
				m := trainerReviewModel(size[0], size[1])
				m.section = section
				switch state {
				case 1:
					m.activity.historyMode = true
				case 2:
					m.activity.error = "Unable to read saved progress."
				case 3:
					m.activity.loading = true
				}
				controls := m.activityControls()
				if len(controls) == 0 {
					t.Fatal("no recovery controls")
				}
				allowed := map[int]bool{}
				for _, c := range controls {
					allowed[c.id] = true
				}
				reached := map[int]bool{controls[0].id: true}
				queue := []int{controls[0].id}
				for len(queue) > 0 {
					id := queue[0]
					queue = queue[1:]
					for _, code := range []rune{tea.KeyUp, tea.KeyDown, tea.KeyLeft, tea.KeyRight} {
						next := m
						next.focus = id
						next, _ = update(next, key(code))
						if !allowed[next.focus] {
							t.Fatalf("%v section%d state%d: hidden target %d -> %d", size, section, state, id, next.focus)
						}
						if !reached[next.focus] {
							reached[next.focus] = true
							queue = append(queue, next.focus)
						}
					}
				}
				v := m.View()
				for _, c := range controls {
					if !reached[c.id] {
						t.Fatalf("%v section%d state%d: unreachable %d", size, section, state, c.id)
					}
					cmd := v.OnMouse(tea.MouseClickMsg{X: c.x + c.w/2, Y: c.y, Button: tea.MouseLeft})
					if cmd == nil {
						t.Fatalf("%v section%d state%d: missing mouse target %d", size, section, state, c.id)
					}
					msg := cmd()
					id, ok := msg.(activateMsg)
					if !ok || int(id) != c.id {
						t.Fatalf("%v section%d state%d: wrong mouse target %d: %v", size, section, state, c.id, msg)
					}
				}
			}
		}
	}
}
