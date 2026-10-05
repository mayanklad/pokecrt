package tui

import "strings"

// The town is a static character drawing: no image assets or animation timers.
func (m Model) paintScene(c *canvas, x, y, w, h int) {
	if w < 20 || h < 2 {
		return
	}
	p := m.palette()
	road, green, water, roof := p.gold, p.muted, p.accent, p.accent
	if !m.noColor && !m.nativePalette() {
		green, water, roof = fg(99, 171, 126), fg(103, 173, 211), fg(223, 126, 139)
		if m.lightPalette() {
			green, water, roof = fg(43, 112, 72), fg(39, 111, 154), fg(159, 62, 78)
		}
	}
	mw := min(w-2, 84)
	mx := x + (w-mw)/2
	if h == 2 {
		c.put(mx, y, "CENTER ───── ◆ YOU ───── MART", p.accent)
		c.put(mx, y+1, "VERDANT TOWN     ROUTE 01 →", road)
		return
	}
	if h < 8 {
		c.put(mx, y, "╭─ CENTER ─╮    ╭─ MART ─╮", roof)
		c.put(mx, y+1, "╰────┬─────╯    ╰───┬────╯", p.accent)
		c.put(mx, y+2, strings.Repeat("─", mw/2)+" ◆ YOU", road)
		return
	}
	mh := min(h, 15)
	my := y + (h-mh)/2
	cy := my + mh/2 + 1
	cx := mx + mw/2
	c.put(mx+2, my, "VERDANT TOWN", p.foreground)
	c.put(mx+mw-16, my, "ROUTE 01  →", road)
	c.put(mx, cy-1, strings.Repeat("─", mw), road)
	c.put(mx, cy+1, strings.Repeat("─", mw), road)
	for row := my + 1; row < my+mh; row++ {
		if row < cy-1 || row > cy+1 {
			c.put(cx-2, row, "│   │", road)
		}
	}
	c.put(cx-2, cy-1, "╯   ╰", road)
	c.put(cx-2, cy+1, "╮   ╭", road)
	building := func(bx, by int, name string) {
		c.put(bx, by, "╭────────────╮", roof)
		c.put(bx, by+1, "│            │", p.accent)
		c.put(bx+2, by+1, name, p.foreground)
		c.put(bx, by+2, "╰─────┬┬─────╯", p.accent)
	}
	if mw >= 55 {
		building(mx+6, my+2, "✚ CENTER")
		building(cx+7, my+2, "MART")
		for py := my + 5; py < cy-1; py++ {
			c.put(mx+12, py, "││", road)
			c.put(cx+13, py, "││", road)
		}
		if mh >= 12 {
			c.put(mx+12, cy+2, "││", road)
			building(mx+6, cy+3, "YOUR HOME")
			c.put(cx+9, cy+3, "╭──────────────╮", water)
			c.put(cx+9, cy+4, "│ ≋  ≋  ≋  ≋   │", water)
			c.put(cx+9, cy+5, "╰──────────────╯", water)
		}
	} else {
		c.put(mx+2, my+2, "╭─ CENTER ─╮", roof)
		c.put(mx+2, my+3, "╰────┬─────╯", p.accent)
		c.put(cx+5, my+2, "MART", p.foreground)
	}
	for _, pos := range [][2]int{{mx + 1, my + 2}, {mx + 1, my + 4}, {mx + mw - 3, my + 2}, {mx + mw - 3, cy + 3}} {
		if pos[1] < my+mh {
			c.put(pos[0], pos[1], "♣", green)
		}
	}
	if mw >= 75 && mh >= 12 {
		for _, ty := range []int{my + 2, my + 4, cy + 3, cy + 5} {
			c.put(mx+mw-11, ty, "♣  ♣  ♣", green)
			c.put(mx+1, ty, "♣  ♣", green)
		}
		c.put(cx+10, cy+6, "TOWN GARDEN", green)
	}
	c.put(cx-1, cy, "◆", p.accent)
	c.put(cx+3, cy, "YOU", p.foreground)
}
