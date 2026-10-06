package tui

import (
	"fmt"
	"github.com/charmbracelet/x/ansi"
	"sort"
	"strings"
)

type encounterLayout struct {
	bodyY, bodyH, footerY int
	outlined              bool
}

func (m Model) encounterGeometry() encounterLayout {
	g := encounterLayout{bodyY: 4, footerY: m.height - 5}
	g.outlined = m.height >= 24 && m.width >= 56
	if g.outlined {
		g.bodyY = 6
		g.footerY = m.height - 6
	}
	g.bodyH = max(3, g.footerY-2-g.bodyY)
	return g
}
func (m Model) encounterArtView() bool {
	return !m.activity.historyMode && len(m.activity.art) > 0 && m.activity.error == "" && (!m.activity.resultDetails || m.dexWide())
}
func (m Model) encounterControls() []dexControl {
	if m.width < 40 || m.height < 12 {
		return []dexControl{{0, max(0, m.height-1), min(12, m.width), 14, "Quit"}}
	}
	g := m.encounterGeometry()
	w := m.width
	cs := []dexControl{}
	add := func(x, y, width, id int, label string) { cs = append(cs, dexControl{x, y, width, id, label}) }
	topY := 2
	if g.outlined {
		topY = 3
	}
	label := "Encounter"
	if m.activity.recording {
		label = "Saving…"
	} else if m.activity.loading {
		label = "Reading…"
	}
	add(2, topY, 13, 10, label)
	if w >= 56 {
		add(16, topY, 12, 36, "Result")
		add(29, topY, 13, 30, "History")
	} else {
		add(16, topY, 10, 36, "Result")
		add(27, topY, 11, 30, "History")
	}
	bottom := g.bodyY + g.bodyH - 1
	scroll := func(x, width, id int, label string) { add(x+1, bottom, ansi.StringWidth(label)+4, id, label) }
	arrows := func(x, width int, ids []int, labels []string) {
		start := x + width - 1 - (len(ids)*5 + len(ids) - 1)
		for i, id := range ids {
			add(start+i*6, bottom, 5, id, labels[i])
		}
	}
	if m.activity.historyMode {
		if len(m.activity.data.history) > 0 {
			scroll(2, w-4, 20, "Scroll")
			add(14, bottom, 8, 37, "Open")
			if len(m.activity.data.history) > 1 {
				arrows(2, w-4, []int{31, 32}, []string{"↑", "↓"})
			}
		}
	} else if m.encounterArtView() {
		artX, artW := 2, w-4
		if m.dexWide() {
			artW = w/2 - 3
		}
		vertical := len(m.activity.art) > g.bodyH-2
		pan := m.activityMaxPan() > 0
		if vertical || pan {
			scroll(artX, artW, 21, "Art")
			ids := []int{}
			labels := []string{}
			if vertical {
				ids = append(ids, 33, 34)
				labels = append(labels, "↑", "↓")
			}
			if pan {
				ids = append(ids, 18, 19)
				labels = append(labels, "←", "→")
			}
			arrows(artX, artW, ids, labels)
		}
		if !m.dexWide() {
			add(w-13, g.bodyY-1, 11, 35, "Details")
		}
		if m.dexWide() && m.activityMaxScroll() > 0 {
			scroll(w/2, w/2-2, 22, "Scroll")
			arrows(w/2, w/2-2, []int{16, 17}, []string{"↑", "↓"})
		}
	} else {
		if m.activityMaxScroll() > 0 {
			scroll(2, w-4, 22, "Scroll")
			arrows(2, w-4, []int{16, 17}, []string{"↑", "↓"})
		}
		if len(m.activity.art) > 0 && m.activity.error == "" && !m.dexWide() {
			add(w-13, g.bodyY-1, 11, 35, "Art")
		}
	}
	if g.outlined {
		x := 2
		for i, label := range []string{"Back", "Theme", "Refresh", "Quit"} {
			width := 12
			if label == "Theme" {
				width = 11
			}
			if label == "Refresh" {
				width = 13
			}
			add(x, g.footerY, width, []int{12, 13, 11, 14}[i], label)
			x += width + 1
		}
	} else {
		x := 2
		gap := 1
		if w < 43 {
			gap = 0
		}
		for i, label := range []string{"Back", "Theme", "Refresh", "Quit"} {
			width := ansi.StringWidth(label) + 4
			add(x, g.footerY, width, []int{12, 13, 11, 14}[i], label)
			x += width + gap
		}
	}
	return cs
}
func (m Model) paintEncounter(c *canvas) {
	g, p := m.encounterGeometry(), m.palette()
	w, h := m.width, m.height
	c.box(0, 0, w, h, "", p.accent)
	c.put(2, 1, "POKÉCRT / ENCOUNTERS", p.accent)
	c.put(2, 1, "POKÉCRT", m.brandStyle())
	if m.dexWide() {
		c.put(w/2, 1, "TRAINER "+clean(m.activity.data.profile.Name), p.muted)
	}
	title := "ENCOUNTER DETAILS"
	if m.activity.historyMode {
		title = "RECENT DISCOVERIES"
	}
	if m.encounterArtView() {
		title = "DEVICE DISPLAY"
	}
	if m.encounterArtView() && m.dexWide() {
		artW := w/2 - 3
		c.box(2, g.bodyY, artW, g.bodyH, "DEVICE DISPLAY", p.accent)
		m.paintActivityArt(c, 3, g.bodyY+1, artW-2, g.bodyH-2)
		c.box(w/2, g.bodyY, w/2-2, g.bodyH, "ENCOUNTER DETAILS", p.accent)
	} else {
		c.box(2, g.bodyY, w-4, g.bodyH, title, p.accent)
		if m.encounterArtView() {
			m.paintActivityArt(c, 3, g.bodyY+1, w-6, g.bodyH-2)
		}
	}
	if !m.activity.historyMode && m.activity.result != nil {
		y := g.bodyY - 1
		result := m.activity.result
		headingW := w - 4
		if !m.dexWide() && len(m.activity.art) > 0 {
			headingW = w - 17
		}
		c.put(2, y, ansi.Truncate(fmt.Sprintf("#%03d %s  +%d XP", result.Choice.Key().SpeciesID, strings.ToUpper(clean(result.Choice.Snapshot().SpeciesName)), result.XPAwarded), headingW, "…"), p.muted)
	}
	if !m.encounterArtView() || m.dexWide() {
		x, width := 4, w-8
		if m.encounterArtView() && m.dexWide() {
			x = w/2 + 2
			width = w/2 - 6
		}
		rows := m.activityWrapped()
		if m.activity.loading && m.activity.data.profile.ID == 0 {
			rows = []string{"Reading trainer records…"}
		}
		if m.activity.error != "" {
			rows = wrapActivityLines([]string{m.activity.error}, width)
		}
		targets := m.activityHistoryTargets()
		for i := 0; i < g.bodyH-2 && i+m.activity.scroll < len(rows); i++ {
			row := i + m.activity.scroll
			c.putANSI(x, g.bodyY+1+i, ansi.Truncate(rows[row], width, "…"))
			if m.activity.historyMode && row < len(targets) && targets[row] >= 0 {
				c.hits = append(c.hits, hit{x, g.bodyY + 1 + i, width, 1000 + targets[row]})
			} else if m.activityMaxScroll() > 0 {
				c.hits = append(c.hits, hit{x, g.bodyY + 1 + i, width, 22})
			}
		}
	}
	for _, control := range m.encounterControls() {
		if g.outlined && (control.y == 3 || control.y == g.footerY) {
			selected := control.id == 30 && m.activity.historyMode || control.id == 36 && !m.activity.historyMode
			c.framedControl(control.x, control.y-1, control.w, control.id, control.label, m, selected)
		} else {
			if control.y == g.bodyY+g.bodyH-1 {
				for _, padX := range []int{control.x - 1, control.x + control.w} {
					if padX >= 0 && padX < c.width && c.rows[control.y][padX].text == "─" {
						c.put(padX, control.y, " ", p.foreground)
					}
				}
			}
			c.button(control.x, control.y, control.w, control.id, control.label, m)
		}
	}
	m.paintEncounterHints(c)
}
func (m Model) paintEncounterHints(c *canvas) {
	first := "[↑↓←→] Move  [Tab] Focus"
	action := "Select"
	if m.focus == 21 {
		first = "[↑↓] Scroll  [←→] Pan  [Tab] Focus"
	}
	if m.focus == 22 {
		first = "[↑↓] Scroll  [←→] Move  [Tab] Focus"
	}
	if m.focus == 20 {
		first = "[↑↓] Choose  [←→] Move  [Tab] Focus"
		action = "Open"
	}
	second := "[Enter] " + action + "  [Esc] Back  [Q] Quit"
	full := first + "  " + second
	y := c.height - 3
	if ansi.StringWidth(full) <= c.width-4 {
		y = c.height - 2
	}
	c.put(0, y-1, "├"+strings.Repeat("─", max(0, c.width-2))+"┤", m.footerDividerStyle())
	if ansi.StringWidth(full) <= c.width-4 {
		c.put(2, y, full, m.hintStyle())
		return
	}
	c.put(2, y, first, m.hintStyle())
	c.put(2, y+1, second, m.hintStyle())
}
func (m *Model) moveEncounterControl(key string) {
	controls := m.encounterControls()
	current := dexControl{}
	for _, c := range controls {
		if c.id == m.focus {
			current = c
		}
	}
	if key == "down" && (m.focus == 10 || m.focus == 36 || m.focus == 30 || m.focus == 35) {
		preferred := 22
		if m.activity.historyMode {
			preferred = 20
		} else if m.encounterArtView() {
			preferred = 21
		}
		for _, control := range controls {
			if control.id == preferred {
				m.focus = preferred
				return
			}
		}
		m.focus = 12
		return
	}
	if key == "down" && current.y == m.encounterGeometry().footerY {
		return
	}
	if key == "up" && (m.focus == 10 || m.focus == 36 || m.focus == 30) {
		return
	}
	if key == "left" || key == "right" {
		row := []dexControl{}
		for _, c := range controls {
			if c.y == current.y {
				row = append(row, c)
			}
		}
		sort.Slice(row, func(i, j int) bool { return row[i].x < row[j].x })
		for i, c := range row {
			if c.id == m.focus {
				next := i + 1
				if key == "left" {
					next = i - 1
				}
				if next >= 0 && next < len(row) {
					m.focus = row[next].id
					return
				}
				if current.y != m.encounterGeometry().footerY {
					m.focus = 14
					if key == "left" {
						m.focus = 12
					}
				}
				return
			}
		}
	}
	m.focus = directionalTarget(controls, m.focus, key)
}
