package tui

import "strings"

// A static device display keeps first-run presentation sharp without revealing
// undiscovered species or introducing image assets and animation timers.
func (m Model) paintScene(c *canvas, x, y, w, h int) {
	if w < 8 || h < 6 {
		return
	}
	p := m.palette()
	dw, dh := min(w-4, 76), min(h, 17)
	dx, dy := x+(w-dw)/2, y+(h-dh)/2
	red := fg(237, 99, 112)
	if m.noColor {
		red = ""
	}
	c.box(dx, dy, dw, dh, "POKÉDEX / FIELD DEVICE", red)
	c.put(dx+3, dy+1, "◉  ● ● ●", p.muted)
	if dw >= 36 {
		c.put(dx+dw-11, dy+1, "OFFLINE", p.muted)
	}
	sw, sh := dw-8, dh-6
	c.box(dx+4, dy+3, sw, sh, "ENTRY ???", p.muted)
	center := func(row int, s, style string) { c.put(dx+(dw-len([]rune(s)))/2, dy+row, s, style) }
	if sh >= 3 {
		center(4+(sh-3)/2, "?", p.accent)
	}
	if sh >= 7 {
		center(dh-7, strings.Repeat("─", min(sw-6, 28)), p.muted)
		status := "Awaiting first discovery"
		if m.snapshot.Active {
			status = "Entry preview coming next"
		}
		center(dh-5, status, p.muted)
	} else if sh >= 4 {
		center(dh-5, "Awaiting discovery", p.muted)
	}
	c.put(dx+3, dy+dh-2, "▣  ━━━   +", p.accent)
	if dw > 45 {
		c.put(dx+dw-25, dy+dh-2, "1025 SPECIES / GEN 1–9", p.muted)
	}
}
