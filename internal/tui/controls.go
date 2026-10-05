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
		return m.palette().muted + "\x1b[1m"
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

	style, border, _ := m.controlColours(id, label)
	if selected {
		border = m.palette().gold
	}
	if m.focus == id {
		border = m.palette().muted
		style = m.controlStyle(true, selected)
	}
	c.keycap(x, y, w, "", border)
	c.controlText(x+1, y+1, w-2, label, style, m.focus == id, selected)
	for row := y; row < y+3; row++ {
		c.hits = append(c.hits, hit{x, row, w, id})
	}
}

// Keep the label centred independently of focus and selection markers.
func (c *canvas) controlText(x, y, w int, label, style string, focused, selected bool) {
	text := ansi.Truncate(label, w, "…")
	left := max(0, (w-ansi.StringWidth(text))/2)
	c.put(x, y, strings.Repeat(" ", w), style)
	c.put(x+left, y, text, style)
	if focused && left >= 1 {
		c.put(x, y, "▶", style)
	}
	if selected {
		if left >= 2 {
			offset := 0
			if focused {
				offset = 1
			}
			c.put(x+offset, y, "●", style)
		} else if w-left-ansi.StringWidth(text) >= 1 {
			c.put(x+w-1, y, "●", style)
		}
	}

}

func (m Model) listFocusStyle() string {
	if m.noColor {
		return ""
	}
	if m.nativePalette() {
		return "\x1b[7m\x1b[1m"
	}
	if m.lightPalette() {
		return fg(246, 245, 237) + bg(20, 102, 117) + "\x1b[1m"
	}
	return fg(2, 19, 33) + bg(73, 213, 236) + "\x1b[1m"
}

func (c *canvas) navigationHints(x, y, w int, m Model, compact bool, action string) {
	guide := m.hintStyle()
	if compact && !m.settings && (m.screen == createScreen || m.screen == dexSearchScreen) {
		c.put(x, y, ansi.Truncate(m.createHelp(w), w, "…"), guide)
		return
	}
	if compact {
		c.put(x, y, ansi.Truncate("[↑↓←→] Move  [Enter] "+action+"  [Tab] Focus", w, "…"), guide)
		return
	}
	items := [][2]string{{"↑↓←→", "Move"}, {"Enter", action}, {"Tab", "Focus"}, {"Esc", "Back"}}
	if !m.settings && m.screen == profilesScreen {
		items = [][2]string{{"↑↓", "Column"}, {"←→", "Row"}, {"Enter", "Select"}}
		if m.focus == 0 {
			choose := "Choose / leave"
			if w < 65 {
				choose = "Choose"
			}
			items = [][2]string{{"↑↓", choose}, {"←→", "Buttons"}, {"Enter", "Use"}, {"Tab", "Focus"}}
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
		c.put(pos, y+2, "[ "+item[0]+" ]", guide)
		c.put(pos+kw+1, y+2, item[1], guide)
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

// Quiet structural borders; only the panel containing keyboard focus is accented.
func (m Model) styleFrames(c *canvas) {
	quiet, active := "", ""
	if !m.noColor {
		quiet, active = fg(64, 88, 104), fg(105, 165, 182)
		if m.lightPalette() {
			quiet, active = fg(157, 172, 175), fg(54, 109, 121)
		}
		if m.nativePalette() {
			quiet, active = "\x1b[2m", "\x1b[1m"
		}
	}
	fx, fy := -1, -1
	for _, hit := range c.hits {
		if hit.id == m.focus {
			fx, fy = hit.x+hit.width/2, hit.y
			break
		}
	}
	if !m.settings && m.screen == dexScreen {
		for _, target := range m.dexTargets() {
			if target.id == m.focus {
				fx, fy = target.x+target.w/2, target.y
				break
			}
		}
	}
	best, area, largest := -1, int(^uint(0)>>1), 0
	for _, f := range c.frames {
		largest = max(largest, f.w*f.h)
	}
	for i, f := range c.frames {
		inside := fx >= f.x && fx < f.x+f.w && fy >= f.y && fy < f.y+f.h
		if !m.settings && m.screen == activityScreen {
			if m.section == 3 && m.dexGeometry().wide && m.focus >= 40 && m.focus <= 45 {
				panel := m.activity.achievementPanel
				if m.focus == 40 || m.focus == 41 {
					panel = m.focus - 40
				}
				if m.focus >= 42 {
					panel = (m.focus - 42) / 2
				}
				px := m.dexGeometry().x + 2
				if panel == 1 {
					px = m.dexGeometry().x + m.dexGeometry().w/2
				}
				inside = f.x == px && f.h == m.activityBodyHeight()
			}
			if m.focus == 22 || m.focus == 16 || m.focus == 17 {
				inside = f.h == m.activityBodyHeight()
				if m.section == 1 && m.dexGeometry().wide && len(m.activity.art) > 0 && !m.activity.historyMode && m.activity.error == "" {
					inside = inside && f.x == m.dexGeometry().x+m.dexGeometry().w/2
				}
			}
			if m.focus == 21 || m.focus == 18 || m.focus == 19 || m.focus == 33 || m.focus == 34 {
				inside = f.x == m.dexGeometry().x+2 && f.h == m.activityBodyHeight()
			}
		}
		if inside && f.w*f.h < largest && f.w*f.h < area {
			best, area = i, f.w*f.h
		}
	}
	for i, f := range c.frames {
		style := quiet
		if i == best {
			style = active
		}
		if !m.settings && m.screen == activityScreen && m.section != 1 && (m.focus == 22 || m.focus == 16 || m.focus == 17) && f.h == m.activityBodyHeight() {
			style = active
		}
		for row := f.y; row < f.y+f.h; row++ {
			for col := f.x; col < f.x+f.w; col++ {
				if row != f.y && row != f.y+f.h-1 && col != f.x && col != f.x+f.w-1 {
					continue
				}
				if row < 0 || row >= c.height || col < 0 || col >= c.width {
					continue
				}
				v := &c.rows[row][col]
				if v.style == f.style {
					v.style = style
				}
			}
		}

	}
}

// Equal whole-cell padding for Dex actions and tabs, without changing other screens.
func balancedControlWidth(w int, label string) int {
	label = strings.TrimSpace(strings.Trim(label, "[]"))
	if w > 3 && (w-2-ansi.StringWidth(label))%2 == 1 {
		return w - 1
	}
	return w
}

func (m Model) brandStyle() string {
	if m.noColor {
		return ""
	}
	if m.lightPalette() {
		return fg(164, 48, 62)
	}
	return fg(232, 97, 109)
}
func (m Model) hintStyle() string {
	if m.noColor {
		return ""
	}
	if m.nativePalette() {
		return "\x1b[2m"
	}
	if m.lightPalette() {
		return fg(85, 105, 111)
	}
	return fg(132, 155, 168)
}
func (c *canvas) pageBox(x, y, w, h int, title string, m Model) {
	c.box(x, y, w, h, "POKÉCRT / "+title, m.palette().accent)
	c.put(x+3, y, "POKÉCRT", m.brandStyle())
}
