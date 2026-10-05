package tui

import (
	"github.com/charmbracelet/x/ansi"
	"strings"
)

// Keycaps occupy three terminal rows. Their border/text are true characters;
// no graphics protocol, image, alternate font or raster overlay is involved.
func (c *canvas) keycap(x, y, w int, label, style string) {
	if w < 3 {
		return
	}
	c.put(x, y, "╭"+strings.Repeat("─", w-2)+"╮", style)
	c.put(x, y+1, "│"+strings.Repeat(" ", w-2)+"│", style)
	label = ansi.Truncate(label, w-2, "")
	c.put(x+1+max(0, (w-2-ansi.StringWidth(label))/2), y+1, label, style)
	c.put(x, y+2, "╰"+strings.Repeat("─", w-2)+"╯", style)
}

// controlStyle keeps focus (cyan pointer) separate from persistent selection (gold dot).
func (m Model) controlStyle(focused, selected bool) string {
	if focused {
		if m.noColor {
			return ""
		}
		return m.palette().accent + "\x1b[1m"
	}
	if selected {
		return m.palette().gold
	}
	return m.palette().foreground
}
func (c *canvas) outlinedButton(x, y, w, id int, label string, m Model) {
	c.framedControl(x, y, min(w, max(12, ansi.StringWidth(label)+6)), id, label, m, false)
}
func (c *canvas) framedControl(x, y, w, id int, label string, m Model, selected bool) {
	if w < 3 {
		return
	}
	label = strings.TrimSpace(strings.Trim(label, "[]"))
	marker := ""
	if selected {
		marker = "● "
	}
	if m.focus == id {
		marker = "▶ "
		if selected {
			marker = "▶● "
		}
	}
	style, border, _ := m.controlColours(id, label)
	if selected {
		border = m.palette().gold
	}
	if m.focus == id {
		border = m.palette().muted
		style = m.controlStyle(true, selected)
	}
	c.keycap(x, y, w, "", border)
	text := ansi.Truncate(marker+label, w-2, "…")
	left := max(0, (w-2-ansi.StringWidth(text))/2)
	c.put(x+1, y+1, strings.Repeat(" ", left)+text+strings.Repeat(" ", max(0, w-2-left-ansi.StringWidth(text))), style)
	for row := y; row < y+3; row++ {
		c.hits = append(c.hits, hit{x, row, w, id})
	}
}

func (c *canvas) navigationHints(x, y, w int, m Model, compact bool, action string) {
	p := m.palette()
	if compact && !m.settings && (m.screen == createScreen || m.screen == dexSearchScreen) {
		c.put(x, y, ansi.Truncate(m.createHelp(w), w, "…"), p.muted)
		return
	}
	if compact {
		c.put(x, y, ansi.Truncate("[↑↓←→] Move  [Enter] "+action+"  [Tab] Focus", w, "…"), p.muted)
		return
	}
	items := [][2]string{{"↑↓←→", "Move"}, {"Enter", action}, {"Tab", "Focus"}, {"Esc", "Back"}}
	if !m.settings && m.screen == profilesScreen {
		items = [][2]string{{"↑↓", "Column"}, {"←→", "Row"}, {"Enter", "Select"}}
		if m.focus == 0 {
			items = [][2]string{{"↑↓", "Choose / leave"}, {"Enter", "Use"}, {"Tab", "Focus"}}
		}
	}
	if !m.settings && (m.screen == createScreen || m.screen == dexSearchScreen) {
		if m.focus == 0 {
			items[0] = [2]string{"←→", "Cursor"}
			items[1][1] = "Create"
			if m.screen == dexSearchScreen {
				items[1][1] = "Search"
			}
		} else if m.focus >= 100 {
			items[1][1] = "Type"
		}
	}
	if !m.settings && m.screen == mainScreen {
		items[3][1] = "Exit"
	}
	if w < 65 {
		items = items[:3]
	}
	if !m.settings && (m.screen == createScreen || m.screen == dexSearchScreen) && m.focus == 0 {
		items = [][2]string{{"↓", "Keyboard"}, {"←→", "Cursor"}, {"Enter", items[1][1]}}
	}
	pos := x
	for _, item := range items {
		kw := ansi.StringWidth(item[0]) + 4
		if pos+kw+ansi.StringWidth(item[1])+2 > x+w {
			break
		}
		c.keycap(pos, y, kw, item[0], p.muted)
		c.put(pos+kw+1, y+1, item[1], p.foreground)
		pos += kw + ansi.StringWidth(item[1]) + 3
	}
}

func directionalTarget(controls []dexControl, currentID int, direction string) int {
	var current dexControl
	found := false
	for _, c := range controls {
		if c.id == currentID {
			current = c
			found = true
			break
		}
	}
	if !found {
		if len(controls) > 0 {
			return controls[0].id
		}
		return currentID
	}
	best := currentID
	bestRow := int(^uint(0) >> 1)
	bestDistance := bestRow
	for _, c := range controls {
		if c.id == currentID {
			continue
		}
		dx := 2*c.x + c.w - 2*current.x - current.w
		dy := c.y - current.y
		if direction == "left" || direction == "right" {
			if dy != 0 || direction == "left" && dx >= 0 || direction == "right" && dx <= 0 {
				continue
			}
			if abs(dx) < bestDistance {
				bestDistance = abs(dx)
				best = c.id
			}
		} else {
			if direction == "up" && dy >= 0 || direction == "down" && dy <= 0 {
				continue
			}
			// Adjacent row first; overlapping controls before distant columns.
			gap := max(0, max(current.x-c.x-c.w, c.x-current.x-current.w))
			horizontal := gap*1000 + abs(dx)
			if abs(dy) < bestRow || abs(dy) == bestRow && horizontal < bestDistance {
				bestRow = abs(dy)
				bestDistance = horizontal
				best = c.id
			}
		}
	}
	return best
}
