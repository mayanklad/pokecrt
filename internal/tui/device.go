package tui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// The town is composed inside its viewport, without assets or animation timers.
func (m Model) paintScene(c *canvas, x, y, w, h int) {
	if w < 20 || h < 1 {
		return
	}
	p := m.palette()
	road, leaf, water, roof := p.gold, p.muted, p.accent, p.accent
	if !m.noColor && !m.nativePalette() {
		road, leaf = fg(168, 151, 110), fg(104, 173, 137)
		water, roof = fg(111, 182, 204), fg(217, 119, 137)
		if m.lightPalette() {
			road, leaf = fg(125, 104, 57), fg(40, 107, 73)
			water, roof = fg(30, 107, 140), fg(158, 53, 76)
		}
	}
	draw := func(dx, dy int, text, style string) {
		if dx < 0 || dx >= w || dy < 0 || dy >= h {
			return
		}
		c.put(x+dx, y+dy, ansi.Truncate(text, w-dx, ""), style)
	}
	// Switch composition before a detailed landmark becomes too small to draw.
	if w < 68 || h < 17 {
		if h == 1 {
			draw(0, 0, "HOME  POND  ◆ YOU →", p.foreground)
			return
		}
		if h < 9 || w < 40 {
			if h == 2 {
				draw(0, 0, "CENTER MART HOME POND", p.foreground)
			} else {
				draw(1, 0, "CENTER   MART", roof)
				draw(1, 1, "HOME     ≋ POND", water)
			}
			draw(0, h-1, "◆ YOU   ROUTE 01 →", road)
			return
		}
		bx, bw := 2, min(16, (w-8)/2)
		rx := w - bw - 2
		landmark := func(dx, dy int, label, style string) {
			draw(dx, dy, "╭"+strings.Repeat("─", bw-2)+"╮", style)
			draw(dx, dy+1, "│"+strings.Repeat(" ", bw-2)+"│", style)
			draw(dx+(bw-ansi.StringWidth(label))/2, dy+1, label, style)
			draw(dx, dy+2, "╰"+strings.Repeat("─", bw-2)+"╯", style)
		}
		landmark(bx, 1, "CENTER", roof)
		landmark(rx, 1, "MART", roof)
		landmark(bx, h-4, "YOUR HOME", p.foreground)
		landmark(rx, h-4, "≋ POND ≋", water)
		ry := h / 2
		draw(0, ry, strings.Repeat("─", w), road)
		draw(w/2-4, ry, " ◆ YOU ", p.accent)
		draw(w-12, ry, " ROUTE 01 → ", road)
		return
	}
	// Streets reach the viewport edges; buildings and gardens fill four blocks.
	mid, avenue := h/2, w/2
	for row := 0; row < h; row++ {
		draw(avenue-4, row, "│       │", road)
	}
	for _, row := range []int{mid - 1, mid + 1} {
		draw(0, row, strings.Repeat("─", w), road)
	}
	draw(avenue-4, mid-1, "╯       ╰", road)
	draw(avenue-4, mid+1, "╮       ╭", road)
	draw(avenue-4, mid, "         ", road)
	draw(avenue-3, mid, " ◆ YOU ", p.accent)
	draw(w-14, mid, " ROUTE 01 → ", road)
	draw(avenue-3, 0, "↑", road)
	building := func(bx, by, bw, bh int, name string, upper bool) {
		if bw < 16 || bh < 5 {
			return
		}
		draw(bx+2, by, "╱"+strings.Repeat("─", bw-6)+"╲", roof)
		draw(bx+1, by+1, "╱"+strings.Repeat(" ", bw-4)+"╲", roof)
		draw(bx+(bw-ansi.StringWidth(name))/2, by+1, name, roof)
		draw(bx, by+2, "╱"+strings.Repeat("─", bw-2)+"╲", roof)
		for row := 3; row < bh-1; row++ {
			draw(bx, by+row, "│"+strings.Repeat(" ", bw-2)+"│", p.foreground)
			if row == 3 || row%3 == 0 {
				for col := 3; col < bw-3; col += 6 {
					draw(bx+col, by+row, "▪ ▪", water)
				}
			}
		}
		door := bx + bw/2 - 1
		draw(door, by+bh-2, "╭─╮", p.foreground)
		draw(bx, by+bh-1, "╰"+strings.Repeat("─", bw-2)+"╯", p.foreground)
		draw(door, by+bh-1, "┴─┴", p.foreground)
		if upper {
			for row := by + bh; row < mid-1; row++ {
				draw(door, row, "│ │", road)
			}
			draw(door, mid-1, "╯ ╰", road)
		} else {
			for row := mid + 2; row < by; row++ {
				draw(door, row, "│ │", road)
			}
			draw(door, mid+1, "╮ ╭", road)
		}
	}
	blockW := avenue - 6
	bw := min(38, blockW-12)
	bh := max(5, (mid-3)*2/3)
	left, right := max(4, (blockW-bw)/2), avenue+6+max(2, (blockW-bw)/2)
	building(left, 2, bw, bh, "✚ POKÉMON CENTER", true)
	building(right, 2, bw, bh, "POKÉ MART", true)
	lowerY := mid + 3
	lowerH := h - lowerY - 1
	building(left, lowerY, bw, lowerH, "YOUR HOME", false)
	// A rectangular stone bank gives the pond a continuous, aligned outline.
	pondW := bw
	if lowerH >= 4 {
		draw(right, lowerY, "╭"+strings.Repeat("─", pondW-2)+"╮", water)
		for row := 1; row < lowerH-1; row++ {
			draw(right, lowerY+row, "│"+strings.Repeat(" ", pondW-2)+"│", water)
			for col := 3 + row%2; col < pondW-2; col += 5 {
				draw(right+col, lowerY+row, "≋", water)
			}
		}
		draw(right, lowerY+lowerH-1, "╰"+strings.Repeat("─", pondW-2)+"╯", water)
		draw(right+max(1, (pondW-11)/2), lowerY, " TOWN POND ", p.foreground)
	}
	// Plant the remaining lawn, keeping streets, signs and landmarks clear.
	for row := 2; row+2 < h; row += 4 {
		if row+2 >= mid-2 && row <= mid+2 {
			continue
		}
		for col := 1; col+4 < w; col += 8 {
			inStreet := col+4 >= avenue-5 && col <= avenue+5
			inLeft := col+4 >= left-1 && col <= left+bw+1
			inRight := col+4 >= right-1 && col <= right+bw+1
			if inStreet || inLeft || inRight {
				continue
			}
			draw(col+1, row, "▄█▄", leaf)
			draw(col, row+1, "▀███▀", leaf)
			draw(col+2, row+2, "│", road)
		}
	}
}
