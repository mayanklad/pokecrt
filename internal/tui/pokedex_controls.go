package tui

import (
	"github.com/charmbracelet/x/ansi"
)

func (m Model) pokedexOutlined() bool { return m.width >= 56 && m.height >= 24 }
func (m Model) pokedexFooterY() int {
	if m.pokedexOutlined() {
		return m.height - 6
	}
	return m.height - 5
}
func (m Model) pokedexTabY() int {
	y := m.pokedexFooterY() - 1
	if m.pokedexOutlined() {
		y -= 3
	}
	if m.width < 64 && m.height >= 20 {
		if m.pokedexOutlined() {
			y -= 3
		} else {
			y--
		}
	}
	return y
}
func (m Model) pokedexGeometry(wide bool) dexLayout {
	w, h := max(1, m.width), max(1, m.height)
	g := dexLayout{w: w, h: h, wide: wide, short: h < 20, listX: 2, listW: w - 4, entryX: 2, entryW: w - 4}
	g.bodyY = 5
	if g.short {
		g.bodyY = 4
	}
	bottom := m.pokedexFooterY() - 1
	if m.pokedexOutlined() {
		bottom--
	}
	if wide || m.dex.detail {
		bottom = m.pokedexTabY() - 1
		if m.pokedexOutlined() {
			bottom--
		}
		if !wide {
			g.bodyY = 4
			if g.short {
				g.bodyY = 3
			}
			if m.dex.filters {
				g.bodyY += 2
			}
		}
	}
	if wide {
		g.listW = 26
		g.entryX = 30
		g.entryW = w - 32
	}
	g.bodyH = max(0, bottom-g.bodyY+1)
	if g.bodyH < 3 {
		g.bodyH = 0
	}
	g.listH = g.bodyH
	if wide {
		g.listH = max(0, m.pokedexFooterY()-2-g.bodyY+1)
	}
	return g
}
func (m Model) pokedexControls() []dexControl {
	g := m.dexGeometry()
	if m.width < 40 || m.height < 12 {
		return []dexControl{{0, max(0, m.height-1), min(12, m.width), 8, "Quit"}}
	}
	cs := []dexControl{}
	add := func(x, y, w, id int, label string) {
		cs = append(cs, dexControl{x, y, balancedControlWidth(w, label), id, label})
	}
	filters := func(y int) {
		if g.w >= 64 {
			add(2, y, 12, 31, "All")
			add(14, y, 13, 32, "Seen")
			add(27, y, 15, 33, "Unseen")
			if g.wide {
				add(42, y, 16, 3, m.dexGenLabel())
				add(58, y, min(g.w-60, max(16, ansi.StringWidth(m.dexSearchLabel())+6)), 1, m.dexSearchLabel())
			} else {
				add(42, y, min(16, g.w-44), 3, m.dexGenLabel())
			}
		} else {
			add(2, y, 15, 2, []string{"All", "Seen", "Unseen"}[m.dex.status])
			add(17, y, min(16, g.w-19), 3, m.dexGenLabel())
		}
		if !g.wide {
			add(2, y+1, min(g.w-4, max(16, ansi.StringWidth(m.dexSearchLabel())+6)), 1, m.dexSearchLabel())
		}
	}
	top := 3
	if g.short {
		top = 2
	}
	if g.wide || !m.dex.detail {
		filters(top)
	} else {
		add(2, top, 10, 4, "Index")
		if !m.dex.filters {
			add(13, top, 13, 1, "Search /")
		}
		label := "Filters"
		if m.dex.filters {
			label = "Close"
		}
		add(g.w-13, top, 11, 25, label)
		if m.dex.filters {
			filters(top + 1)
		}
	}
	if g.wide || !m.dex.detail {
		bottom := g.bodyY + g.listH - 1
		if g.bodyH >= 3 && len(m.dex.rows) > m.dexIndexHeight() {
			add(g.listX+1, bottom, 10, 19, "Scroll")
			add(g.listX+g.listW-12, bottom, 5, 16, "↑")
			add(g.listX+g.listW-7, bottom, 5, 17, "↓")
		}
		if !g.wide && len(m.dex.rows) > 0 {
			add(g.w-16, 1, 14, 4, "Open entry")
		}
	}
	if g.wide || m.dex.detail {
		tabY := m.pokedexTabY()
		if g.short {
			add(2, tabY, g.w-4, 24, "View: "+dexTabNames[m.dex.tab]+" ▸")
		} else {
			x, w := g.entryX, g.entryW
			cols := 4
			if !g.wide && g.w < 64 {
				cols = 2
			}
			tw := w / cols
			for i, label := range dexTabNames {
				row := i / cols
				step := 1
				if m.pokedexOutlined() {
					step = 3
				}
				width := tw - 1
				if !m.pokedexOutlined() {
					width = min(width, ansi.StringWidth(label)+8)
				}
				add(x+(i%cols)*tw, tabY+row*step, width, 20+i, label)
			}
		}
		bx, by, bw, bh := g.entryX, g.bodyY, g.entryW, g.bodyH
		if m.dex.tab == 0 && (g.wide || !m.dex.overviewFacts) {
			bx, by, bw, bh = m.dexArtRect()
		}
		bottom := by + bh - 1
		vertical, pan := m.dexMaxScroll() > 0, m.dexMaxHorizontal() > 0
		if m.dex.filters && !g.wide && m.height < 16 {
			vertical, pan = false, false
		}
		arrowW := 5
		if bw < 34 {
			arrowW = 4
		}
		if (vertical || pan) && bh >= 3 {
			add(bx+1, bottom, 10, 18, "Scroll")
		}
		x := bx + bw - 1
		if pan && bh >= 3 {
			x -= arrowW
			add(x, bottom, arrowW, 14, "→")
			x -= arrowW
			add(x, bottom, arrowW, 13, "←")
		}
		if vertical && bh >= 3 {
			x -= arrowW
			add(x, bottom, arrowW, 12, "↓")
			x -= arrowW
			add(x, bottom, arrowW, 11, "↑")
		}
		if !g.wide && m.dex.tab == 0 {
			label := "Art"
			if m.dex.overviewFacts {
				label = "Details"
			}
			// Keep the view switch on the tab row, separate from frame controls.
			width := 15
			if g.short {
				for i := range cs {
					if cs[i].id == 24 {
						cs[i].w = g.w - 20
					}
				}
				add(g.w-16, tabY, width, 26, label)
			} else {
				add(g.w-16, 1, width, 26, label)
			}
		}
		if !g.wide && m.dex.tab == 2 && m.dex.entry.Seen && len(m.dex.entry.Evolution) > 0 {
			add(g.w-14, 1, 12, 27, "Full art")
		}
	}
	if m.dexWide() && m.dex.tab == 0 && m.dexFactsMaxScroll() > 0 {
		x, y, w, h := m.dexFactsRect()
		bottom := y + h - 1
		add(x+1, bottom, 10, 40, "Scroll")
		add(x+w-11, bottom, 5, 41, "↑")
		add(x+w-6, bottom, 5, 42, "↓")
	}
	labels := []string{"Back", "Theme", "Refresh", "Quit"}
	if m.dex.familyOrigin > 0 {
		labels[0] = "Family"
	}
	ids := []int{6, 7, 5, 8}
	x := 2
	gap := 1
	if g.w < 43 {
		gap = 0
	}
	for i, label := range labels {
		w := ansi.StringWidth(label) + 4
		if m.pokedexOutlined() {
			w = max(12, w)
		}
		add(x, m.pokedexFooterY(), w, ids[i], label)
		x += w + gap
	}
	return cs
}
func (m Model) paintPokedexControls(c *canvas) {
	for _, control := range m.dexControls() {
		selected := control.id >= 20 && control.id <= 23 && m.dex.tab == control.id-20 || control.id >= 31 && control.id <= 33 && m.dex.status == control.id-31 || control.id == 2 || control.id == 24
		outlined := m.pokedexOutlined() && (control.y == m.pokedexFooterY() || control.id >= 20 && control.id <= 23 || m.dexWide() && control.y == 3)
		copy := m
		if m.focus == 10 && control.id == 18 {
			copy.focus = 18
		}
		if control.id == 26 {
			selected = true
		}
		if outlined {
			c.framedControl(control.x, control.y-1, control.w, control.id, ansi.Truncate(control.label, max(1, control.w-4), "…"), copy, selected)
		} else {
			c.pokedexButton(control, copy, selected)
		}
	}
}
func (m Model) paintPokedexHints(c *canvas) {
	g := m.dexGeometry()
	first := "[↑↓←→] Move  [Tab] Focus"
	action := "Select"
	switch m.focus {
	case 0:
		first = "[↑↓] Choose  [←→] Move  [Tab] Focus"
		action = "Open"
	case 19:
		first = "[↑↓] Choose  [←→] Move  [Tab] Focus"
		action = "Scroll"
	case 18:
		if m.dex.tab == 1 || m.dex.tab == 2 {
			first = "[↑↓] Choose  [←→] Move  [Tab] Focus"
			action = "Inspect"
			break
		}
		fallthrough
	case 10, 40:
		action = "Scroll"
		first = "[↑↓] Scroll  [←→] Move  [Tab] Focus"
		if m.focus == 10 && m.dexMaxScroll() == 0 && m.dexMaxHorizontal() == 0 {
			first = "[↑↓←→] Move  [Tab] Focus"
		}
		if (m.focus == 10 || m.focus == 18) && m.dex.tab == 0 && (m.dexWide() || !m.dex.overviewFacts) && m.dexMaxHorizontal() > 0 {
			first = "[↑↓] Scroll  [←→] Pan  [Tab] Focus"
		}
	case 15:
		first = "[↑↓] Choose  [←→] Move  [Tab] Focus"
		action = "Inspect"
	case 24:
		action = "Next view"
	case 25:
		action = "Filters"
	case 26:
		action = "Switch"
	case 27:
		action = "Open art"
	case 4:
		action = "Open"
		if m.dex.detail {
			action = "Index"
		}
	case 5:
		action = "Refresh"
	case 6:
		action = "Back"
	case 7:
		action = "Theme"
	case 8:
		action = "Quit"
	case 11, 12, 16, 17, 41, 42:
		action = "Scroll"
	case 13, 14:
		action = "Pan"
	}
	if m.focus >= 2000 && m.focus < 3000 {
		action = "Open"
	}
	second := "[Enter] " + action + "  [Esc] Back"
	// One or two rows, always joined to the outer frame.
	m.paintSetupHints(c, g.x, g.y, g.w, g.h, first, second)
}
func (m Model) pokedexFrameGroup(id int) []int {
	ids := []int{}
	positions := map[int]int{}
	for _, c := range m.dexControls() {
		positions[c.id] = c.x
		if m.dexWide() {
			if c.id == 19 || c.id == 16 || c.id == 17 || c.id == 18 || c.id >= 11 && c.id <= 14 || c.id >= 40 && c.id <= 42 {
				ids = append(ids, c.id)
			}
		} else if id >= 40 && id <= 42 {
			if c.id >= 40 && c.id <= 42 {
				ids = append(ids, c.id)
			}
		} else if id == 0 || id == 19 || id == 16 || id == 17 {
			if c.id == 19 || c.id == 16 || c.id == 17 {
				ids = append(ids, c.id)
			}
		} else if c.id == 18 || c.id >= 11 && c.id <= 14 {
			ids = append(ids, c.id)
		}
	}
	// Controls are added from the right to keep compact geometry simple;
	// navigation follows their visible left-to-right order.
	for i := 0; i < len(ids); i++ {
		for j := i + 1; j < len(ids); j++ {
			xi, xj := positions[ids[i]], positions[ids[j]]
			if xj < xi {
				ids[i], ids[j] = ids[j], ids[i]
			}
		}
	}
	return ids
}

func (m Model) pokedexPageSize(focus int) int {
	if focus == 0 || focus == 19 {
		return m.dexIndexHeight()
	}
	if focus == 40 {
		_, _, _, h := m.dexFactsRect()
		return max(1, h-2)
	}
	if focus == 10 && m.dex.tab == 0 && (m.dexWide() || !m.dex.overviewFacts) {
		_, _, _, h := m.dexArtRect()
		return max(1, h-2)
	}
	return max(1, m.dexBodyHeight())
}
