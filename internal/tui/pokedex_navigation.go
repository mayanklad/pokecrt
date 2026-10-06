package tui

import (
	"sort"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

func pokedexFooter(id int) bool { return id == 6 || id == 7 || id == 5 || id == 8 }
func pokedexTab(id int) bool    { return id >= 20 && id <= 24 }
func pokedexFrameControl(id int) bool {
	return id == 19 || id == 18 || id >= 11 && id <= 14 || id == 16 || id == 17 || id >= 40 && id <= 42
}
func (m Model) pokedexIsTab(id int) bool {
	return pokedexTab(id) || id == 26 && m.height < 20 && m.dex.detail && m.dex.tab == 0
}
func (m Model) pokedexRegionControls(region string) []dexControl {
	out := []dexControl{}
	for _, c := range m.dexControls() {
		match := false
		switch region {
		case "footer":
			match = pokedexFooter(c.id)
		case "tabs":
			match = m.pokedexIsTab(c.id)
		case "frame":
			match = pokedexFrameControl(c.id)
		case "header":
			match = !pokedexFooter(c.id) && !m.pokedexIsTab(c.id) && !pokedexFrameControl(c.id)
		}
		if match {
			out = append(out, c)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if region != "frame" && out[i].y != out[j].y {
			return out[i].y < out[j].y
		}
		return out[i].x < out[j].x
	})
	return out
}
func (m Model) pokedexFocusOrder() []int {
	order := []int{}
	seen := map[int]bool{}
	add := func(id int) {
		if !seen[id] {
			seen[id] = true
			order = append(order, id)
		}
	}
	for _, c := range m.pokedexRegionControls("header") {
		add(c.id)
	}
	g := m.dexGeometry()
	if g.bodyH >= 3 {
		if (g.wide || !m.dex.detail) && len(m.dex.rows) > 0 {
			add(0)
		}
		if g.wide || m.dex.detail {
			if m.dex.tab == 1 && m.dex.entry.Seen && len(m.dex.options) > 0 {
				add(15)
			}
			if m.dex.tab == 2 && m.dex.entry.Seen {
				for _, node := range orderedEvolution(m.dex.entry.Evolution) {
					add(2000 + node.Number)
				}
			}
		}
	}
	for _, region := range []string{"frame", "tabs", "footer"} {
		for _, c := range m.pokedexRegionControls(region) {
			add(c.id)
		}
	}
	return order
}
func (m Model) pokedexAnchorX(id int) int {
	for _, c := range m.dexControls() {
		if c.id == id {
			return c.x + c.w/2
		}
	}
	g := m.dexGeometry()
	if id == 0 || id == 19 {
		return g.listX + g.listW/2
	}
	if id >= 2000 && id < 3000 {
		for _, line := range m.evolutionLines() {
			for _, node := range line.nodes {
				if node.id == id {
					return g.entryX + 2 + node.x + node.w/2
				}
			}
		}
	}
	return g.entryX + g.entryW/2
}
func pokedexClosest(cs []dexControl, x int) int {
	best, distance := -1, int(^uint(0)>>1)
	for _, c := range cs {
		d := abs(c.x + c.w/2 - x)
		if d < distance {
			best, distance = c.id, d
		}
	}
	return best
}
func (m Model) pokedexRow(region string, id int, direction string) int {
	cs := m.pokedexRegionControls(region)
	var current dexControl
	found := false
	for _, c := range cs {
		if c.id == id {
			current = c
			found = true
			break
		}
	}
	if !found {
		return -1
	}
	candidates := []dexControl{}
	if direction == "left" || direction == "right" {
		for _, c := range cs {
			if c.y == current.y && (direction == "left" && c.x < current.x || direction == "right" && c.x > current.x) {
				candidates = append(candidates, c)
			}
		}
		if len(candidates) == 0 {
			return -1
		}
		if direction == "left" {
			return candidates[len(candidates)-1].id
		}
		return candidates[0].id
	}
	row := -1
	for _, c := range cs {
		if direction == "up" && c.y < current.y && (row < 0 || c.y > row) {
			row = c.y
		}
		if direction == "down" && c.y > current.y && (row < 0 || c.y < row) {
			row = c.y
		}
	}
	for _, c := range cs {
		if c.y == row {
			candidates = append(candidates, c)
		}
	}
	return pokedexClosest(candidates, current.x+current.w/2)
}
func (m Model) pokedexEdge(region string, last bool, x int) int {
	cs := m.pokedexRegionControls(region)
	if len(cs) == 0 {
		return -1
	}
	row := cs[0].y
	if last {
		row = cs[len(cs)-1].y
	}
	candidates := []dexControl{}
	for _, c := range cs {
		if c.y == row {
			candidates = append(candidates, c)
		}
	}
	return pokedexClosest(candidates, x)
}
func (m *Model) pokedexTabsOrFooter(x int) {
	id := m.pokedexEdge("tabs", false, x)
	if id < 0 {
		id = pokedexClosest(m.pokedexRegionControls("footer"), x)
	}
	if id >= 0 {
		m.focus = id
	}
}
func (m *Model) pokedexContentExit(direction string) {
	x := m.pokedexAnchorX(m.focus)
	switch direction {
	case "up":
		id := m.pokedexEdge("header", true, x)
		if id >= 0 {
			m.focus = id
		}
	case "down":
		m.pokedexTabsOrFooter(x)
	case "left":
		if m.dexWide() && len(m.dex.rows) > 0 {
			m.focus = 0
		} else {
			m.focus = 6
		}
	case "right":
		// Enter the controls of this pane before leaving it for the footer.
		for _, c := range m.pokedexRegionControls("frame") {
			if c.id == 18 || c.id == 11 || c.id == 12 {
				m.focus = c.id
				return
			}
		}
		m.focus = 8
	}
}
func (m *Model) navigatePokedex(direction string) {
	id := m.focus
	x := m.pokedexAnchorX(id)
	if pokedexFooter(id) {
		if direction == "down" {
			return
		}
		if next := m.pokedexRow("footer", id, direction); next >= 0 {
			m.focus = next
			return
		}
		if direction == "up" {
			next := m.pokedexEdge("tabs", true, x)
			if next < 0 {
				next = pokedexClosest(m.pokedexRegionControls("frame"), x)
			}
			if next < 0 {
				if m.dexWide() || m.dex.detail {
					next = m.dexContentFocus()
				} else if len(m.dex.rows) > 0 {
					next = 0
				} else {
					next = m.pokedexEdge("header", true, x)
				}
			}
			if next >= 0 {
				m.focus = next
			}
			return
		}
		frames := m.pokedexRegionControls("frame")
		if len(frames) > 0 {
			if direction == "left" {
				m.focus = frames[0].id
			} else {
				m.focus = frames[len(frames)-1].id
			}
		} else {
			if m.dexWide() || m.dex.detail {
				m.focus = m.dexContentFocus()
			} else if len(m.dex.rows) > 0 {
				m.focus = 0
			}
		}
		return
	}
	if m.pokedexIsTab(id) {
		if next := m.pokedexRow("tabs", id, direction); next >= 0 {
			m.focus = next
			return
		}
		switch direction {
		case "left":
			m.focus = 6
		case "right":
			m.focus = 8
		case "down":
			m.focus = pokedexClosest(m.pokedexRegionControls("footer"), x)
		case "up":
			next := pokedexClosest(m.pokedexRegionControls("frame"), x)
			if next < 0 {
				next = m.dexContentFocus()
			}
			if next == id {
				next = m.pokedexEdge("header", true, x)
			}
			if next >= 0 {
				m.focus = next
			}
		}
		return
	}
	if pokedexFrameControl(id) {
		if direction == "left" || direction == "right" {
			group := m.pokedexFrameGroup(id)
			for i, v := range group {
				if v == id {
					if direction == "left" {
						if i > 0 {
							m.focus = group[i-1]
						} else {
							m.focus = 6
						}
					} else {
						if i+1 < len(group) {
							m.focus = group[i+1]
						} else {
							m.focus = 8
						}
					}
					return
				}
			}
		}
		if direction == "down" {
			m.pokedexTabsOrFooter(x)
			return
		}
		if direction == "up" {
			switch id {
			case 16, 17, 19:
				m.focus = 0
			case 41, 42:
				m.focus = 40
			default:
				m.focus = m.dexContentFocus()
			}
			if m.focus == id {
				m.pokedexContentExit("up")
			}
		}
		return
	}
	if id == 0 || id == 15 || id == 10 || id >= 2000 && id < 3000 {
		m.pokedexContentExit(direction)
		return
	}
	if next := m.pokedexRow("header", id, direction); next >= 0 {
		m.focus = next
		return
	}
	if direction == "down" {
		g := m.dexGeometry()
		if g.bodyH >= 3 {
			if (g.wide || !m.dex.detail) && len(m.dex.rows) > 0 && (!g.wide || x < g.entryX) {
				m.focus = 0
			} else {
				m.focus = m.dexContentFocus()
			}
		} else {
			m.pokedexTabsOrFooter(x)
		}
	}
}

func (c *canvas) pokedexButton(control dexControl, m Model, selected bool) {
	x, y, w, id := control.x, control.y, control.w, control.id
	if w < 3 || y < 0 || y >= c.height {
		return
	}
	focused := m.focus == id
	style, _, _ := m.controlColours(id, control.label)
	// Reserve enough room for a pointer or dot before truncating long labels.
	limit := max(1, w-4)
	label := ansi.Truncate(control.label, limit, "…")
	left := max(1, (w-2-ansi.StringWidth(label))/2)
	c.put(x, y, "❨"+strings.Repeat(" ", w-2)+"❩", style)
	c.put(x+1+left, y, label, style)
	if focused {
		c.put(x+1, y, "▶", m.palette().muted)
	}
	if selected {
		dot := x + 1
		if focused {
			dot++
		}
		if dot >= x+1+left {
			dot = x + w - 2
		}
		c.put(dot, y, "●", m.palette().gold)
	}
	c.hits = append(c.hits, hit{x, y, w, id})
}

func (m Model) pokedexFocusedFrame() (x, y, w, h int, ok bool) {
	g := m.dexGeometry()
	if g.bodyH < 3 {
		return
	}
	id := m.focus
	switch {
	case id == 0 || id == 19 || id == 16 || id == 17:
		if g.wide || !m.dex.detail {
			return g.listX, g.bodyY, g.listW, g.listH, true
		}
	case id >= 40 && id <= 42:
		if g.wide && m.dex.tab == 0 {
			x, y, w, h = m.dexFactsRect()
			return x, y, w, h, true
		}
	case id == 10 || id == 18 || id >= 11 && id <= 14 || id == 15 || id >= 2000:
		if g.wide || m.dex.detail {
			if m.dex.tab == 0 && (g.wide || !m.dex.overviewFacts) {
				x, y, w, h = m.dexArtRect()
				return x, y, w, h, true
			}
			return g.entryX, g.bodyY, g.entryW, g.bodyH, true
		}
	}
	return
}

func pokedexEntryFocus(id int) bool {
	return id == 10 || id == 15 || id == 18 || id >= 11 && id <= 14 || id >= 20 && id <= 27 || id >= 40 && id <= 42 || id >= 2000 && id < 4000
}
