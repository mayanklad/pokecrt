package tui

import (
	tea "charm.land/bubbletea/v2"
	"context"
	"github.com/charmbracelet/x/ansi"
	"strings"
	"testing"
)

func TestHomeLogoClearOfHeaderActions(t *testing.T) {
	for width := 90; width <= 130; width++ {
		m := New(context.Background(), nil, Dark, false)
		m.width, m.height = width, 28
		c := newCanvas(width, 28)
		m.paintMain(c)
		expected := newCanvas(width, 28)
		x := min((width-41)/2, width-72)
		m.paintLogo(expected, x, 2)
		for y := 2; y < 5; y++ {
			for col := x; col < x+41; col++ {
				if c.rows[y][col].text != expected.rows[y][col].text {
					t.Fatalf("logo overwritten at width %d cell %d,%d", width, col, y)
				}
			}
		}
		if x+41 > width-31 {
			t.Fatal("logo has no clearance from actions", width)
		}
	}
}

func TestShortHomeHeaderNavigationAndHints(t *testing.T) {
	for _, size := range [][2]int{{40, 12}, {40, 15}, {40, 16}, {48, 20}, {64, 24}, {89, 27}, {120, 24}} {
		for _, tc := range []struct {
			from int
			key  rune
			want int
		}{{0, tea.KeyUp, 5}, {5, tea.KeyRight, 6}, {6, tea.KeyLeft, 5}, {5, tea.KeyDown, 0}, {6, tea.KeyDown, 0}, {4, tea.KeyDown, 4}, {5, tea.KeyUp, 5}, {6, tea.KeyUp, 6}, {0, 'k', 5}, {6, 'j', 0}} {
			m := New(context.Background(), nil, Dark, false)
			m.width, m.height = size[0], size[1]
			m.focus = tc.from
			m, cmd := update(m, key(tc.key))
			if m.focus != tc.want || cmd != nil {
				t.Fatalf("size %v key %v from %d: got %d", size, tc.key, tc.from, m.focus)
			}
		}
		m := New(context.Background(), nil, Dark, true)
		m.width, m.height = size[0], size[1]
		v := m.View()
		plain := ansi.Strip(v.Content)
		for _, label := range []string{"POKECRT", "Refresh", "Quit", "Pokédex", "Appearance", "↑↓←→", "Tab", "Enter"} {
			if !strings.Contains(plain, label) {
				t.Fatal("missing Home content", size, label)
			}
		}
		seen := map[int]bool{}
		for i := 0; i < 7; i++ {
			seen[m.focus] = true
			m, _ = update(m, key(tea.KeyTab))
		}
		if len(seen) != 7 {
			t.Fatal("Tab misses controls", size, seen)
		}
		ids := map[int]bool{}
		for y := 0; y < size[1]; y++ {
			for x := 0; x < size[0]; x++ {
				if cmd := v.OnMouse(tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft}); cmd != nil {
					ids[int(cmd().(activateMsg))] = true
				}
			}
		}
		if len(ids) != 7 {
			t.Fatal("mouse misses controls", size, ids)
		}
	}
}

func TestHomeRoadLabelPadding(t *testing.T) {
	for _, size := range [][2]int{{40, 9}, {64, 13}, {90, 16}, {90, 22}} {
		m := New(context.Background(), nil, Dark, true)
		c := newCanvas(size[0], size[1])
		m.paintScene(c, 0, 0, size[0], size[1])
		plain := ansi.Strip(c.content(m.palette()))
		for _, label := range []string{" ◆ YOU ", " ROUTE 01 → "} {
			if !strings.Contains(plain, label) {
				t.Fatal("road sign lacks padding", size, label)
			}
		}
		if strings.Contains(plain, "─◆") || strings.Contains(plain, "YOU─") || strings.Contains(plain, "─ROUTE") || strings.Contains(plain, "→─") {
			t.Fatal("road line touches sign", size)
		}
	}
}

func TestShortHomeHeaderClearance(t *testing.T) {
	for width := 44; width < 90; width++ {
		m := New(context.Background(), nil, Dark, true)
		m.width, m.height = width, 16
		m.snapshot.Name = "Mayank"
		m.snapshot.Active = true
		m.snapshot.Level = 1
		c := newCanvas(width, 16)
		m.paintMain(c)
		lines := strings.Split(ansi.Strip(c.content(m.palette())), "\n")
		if !strings.Contains(lines[1], "POKECRT") || !strings.Contains(lines[2], "TRAINER HUB") || !strings.Contains(lines[4], "Trainer: Mayank") {
			t.Fatal("header text overwritten", width)
		}
		for _, id := range []int{5, 6} {
			found := false
			for _, h := range c.hits {
				if h.id == id {
					found = true
					if h.y > 3 {
						t.Fatal("action not in top header", width, id)
					}
				}
			}
			if !found {
				t.Fatal("missing header action", width, id)
			}
		}
	}
}
