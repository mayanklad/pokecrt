package tui

import (
	"context"
	"strings"
	"testing"
)

func TestFramedControlMouseBorderAndSelection(t *testing.T) {
	for _, nc := range []bool{false, true} {
		m := New(context.Background(), nil, Dark, nc)
		m.focus = 4
		c := newCanvas(40, 12)
		c.framedControl(2, 3, 18, 4, "Selected theme", m, true)
		if len(c.hits) != 3 {
			t.Fatal("all three rows must be clickable")
		}
		for i, h := range c.hits {
			if h.x != 2 || h.y != 3+i || h.width != 18 || h.id != 4 {
				t.Fatal("border hit target", h)
			}
		}
		var line strings.Builder
		for _, cell := range c.rows[4] {
			line.WriteString(cell.text)
		}
		if !strings.Contains(line.String(), "▶●") {
			t.Fatal("focus must not erase selection", line.String())
		}
	}
}

func TestDirectionalNavigationUsesAdjacentRowAndNoDiagonalHorizontalJump(t *testing.T) {
	controls := []dexControl{{0, 0, 10, 1, ""}, {0, 10, 10, 2, ""}, {15, 1, 10, 3, ""}, {15, 0, 10, 4, ""}}
	if got := directionalTarget(controls, 1, "down"); got != 3 {
		t.Fatal("skipped adjacent row", got)
	}
	if got := directionalTarget(controls, 1, "right"); got != 4 {
		t.Fatal("horizontal arrow left row", got)
	}
	if got := directionalTarget(controls, 4, "right"); got != 4 {
		t.Fatal("right edge wrapped", got)
	}
	if got := directionalTarget(controls, 1, "left"); got != 1 {
		t.Fatal("left edge wrapped", got)
	}
}

func TestActivityPagesStartOnVisibleControls(t *testing.T) {
	for _, section := range []int{1, 2, 3} {
		m := New(context.Background(), nil, Dark, false)
		m.width, m.height = 120, 40
		m.openActivity(section)
		found := false
		for _, control := range m.activityControls() {
			if control.id == m.focus {
				found = true
			}
		}
		if !found {
			t.Fatal("initial focus not rendered", section, m.focus)
		}
	}
}

func TestPurposeColoursKeepNativeBackgroundAndNoColor(t *testing.T) {
	m := New(context.Background(), nil, Dark, false)
	surfaces := map[string]bool{}
	for _, label := range []string{"Create", "Cancel", "Appearance", "Refresh", "Quit"} {
		surface, _, _ := m.buttonColours(label)
		if surfaces[surface] {
			t.Fatal("purpose colours are indistinguishable", label)
		}
		surfaces[surface] = true
	}
	m.appearance = TerminalNative
	for _, label := range []string{"Create", "Cancel", "Appearance", "Refresh", "Quit"} {
		surface, _, _ := m.buttonColours(label)
		if strings.Contains(surface, "[48;") {
			t.Fatal("native background replaced", label)
		}
	}
	m.noColor = true
	for _, label := range []string{"Create", "Cancel", "Appearance", "Refresh", "Quit"} {
		surface, edge, lower := m.buttonColours(label)
		if surface+edge+lower != "" {
			t.Fatal("NO_COLOR styling")
		}
	}
}

func TestActionFootersPrecedeHintStrip(t *testing.T) {
	for _, size := range [][2]int{{120, 40}, {80, 28}, {40, 12}} {
		m := New(context.Background(), nil, Dark, false)
		m.width, m.height = size[0], size[1]
		g := m.dexGeometry()
		hintY := g.y + g.h - 2
		if g.h >= 24 {
			hintY = g.y + g.h - 4
		}
		for _, control := range m.dexControls() {
			if control.id == 5 || control.id == 6 || control.id == 7 || control.id == 8 {
				last := control.y
				if !g.short {
					last++
				}
				if last >= hintY {
					t.Fatal("Dex actions overlap hints", size, control)
				}
			}
		}
		m.section = 2
		for _, control := range m.activityControls() {
			if control.id >= 11 && control.id <= 14 {
				last := control.y
				if !g.short {
					last++
				}
				if last >= hintY {
					t.Fatal("activity actions overlap hints", size, control)
				}
			}
		}
	}
}

func TestActivityPanControlsRequireOverflow(t *testing.T) {
	m := New(context.Background(), nil, Dark, true)
	m.width, m.height, m.section = 120, 40, 1
	m.activity.art = []string{"small sprite"}
	for _, c := range m.activityControls() {
		if c.id == 18 || c.id == 19 || c.id == 21 || c.id == 33 || c.id == 34 {
			t.Fatal("fitting artwork has inert controls", c)
		}
	}
	m.activity.art = []string{strings.Repeat("x", 100)}
	found := false
	for _, c := range m.activityControls() {
		if c.id == 19 {
			found = true
		}
	}
	if !found {
		t.Fatal("oversized artwork has no mouse pan action")
	}
}

func TestHomeDescriptionFollowsFocus(t *testing.T) {
	m := New(context.Background(), nil, Dark, true)
	m.snapshot.Active = true
	m.focus = 0
	first := m.dialogue()
	m.focus = 2
	if m.dialogue() == first || m.homeTitle() != "Trainer" {
		t.Fatal("home preview does not follow focus")
	}
}
