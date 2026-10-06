package tui

import (
	"github.com/charmbracelet/x/ansi"
	"strings"
)

func compactActionRows(width int) bool { return width < 48 }

func pageFooterControls(width, height, y int, ids []int, labels []string, outlined bool) []dexControl {
	out := []dexControl{}
	x := 2
	for i, label := range labels {
		if compactActionRows(width) && i == 2 {
			x = 2
			y++
		}

		w := ansi.StringWidth(label) + 4
		if outlined {
			w = max(12, w)
		}
		w = balancedControlWidth(w, label)
		out = append(out, dexControl{x, y, w, ids[i], label})
		x += w + 1
	}
	return out
}

func (m Model) paintMinimumPageHints(c *canvas) bool {
	if !compactActionRows(m.width) || m.height >= 16 {
		return false
	}
	c.put(0, c.height-3, "├"+strings.Repeat("─", c.width-2)+"┤", m.footerDividerStyle())
	c.put(2, c.height-2, "[Tab] Focus [Enter] Use [Esc] Back", m.hintStyle())
	return true
}

// Wrapped actions follow their rendered rows; the top edge returns to the page.
func footerRowNavigation(controls []dexControl, id int, direction string, ids []int) (int, bool) {
	group := []dexControl{}
	current := dexControl{}
	found := false
	for _, c := range controls {
		for _, action := range ids {
			if c.id == action {
				group = append(group, c)
				if c.id == id {
					current = c
					found = true
				}
			}
		}
	}
	if !found || len(group) == 0 {
		return id, false
	}
	top, bottom := group[0].y, group[0].y
	for _, c := range group {
		top = min(top, c.y)
		bottom = max(bottom, c.y)
	}
	if top == bottom {
		return id, false
	}
	candidates := []dexControl{}
	for _, c := range group {
		switch direction {
		case "left":
			if c.y == current.y && c.x < current.x {
				candidates = append(candidates, c)
			}
		case "right":
			if c.y == current.y && c.x > current.x {
				candidates = append(candidates, c)
			}
		case "up":
			if c.y < current.y {
				candidates = append(candidates, c)
			}
		case "down":
			if c.y > current.y {
				candidates = append(candidates, c)
			}
		}
	}
	if len(candidates) > 0 {
		return pokedexClosest(candidates, current.x+current.w/2), true
	}
	if direction == "up" {
		return id, false
	}
	if direction == "left" && id == ids[0] || direction == "right" && id == ids[len(ids)-1] {
		return id, false
	}
	return id, true
}
