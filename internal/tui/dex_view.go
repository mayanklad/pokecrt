package tui

import (
	"fmt"
	"github.com/charmbracelet/x/ansi"
	"strconv"
	"strings"
	"time"
)

var dexTabNames = []string{"Overview", "Variants", "Evolution", "Records"}

type dexLine struct {
	text   string
	art    bool
	target int
	nodes  []dexControl
}
type dexControl struct {
	x, y, w, id int
	label       string
}
type dexLayout struct {
	x, y, w, h, bodyY, bodyH, entryX, entryW, listX, listW, listH int
	wide, short                                                   bool
}

func (m Model) dexGeometry() dexLayout {
	if m.screen == dexScreen {
		return m.pokedexGeometry(m.dexWide())
	}

	if !m.dexWide() {
		return m.compactDexGeometry()
	}
	cw, ch := max(1, m.width), max(1, m.height)
	w, h := cw, ch
	g := dexLayout{x: (cw - w) / 2, y: (ch - h) / 2, w: w, h: h, wide: m.dexWide(), short: h < 20}
	g.listX = g.x + 2
	g.listW = w - 4
	g.entryX = g.x + 2
	g.entryW = w - 4
	if g.wide {
		g.bodyY = g.y + 5
		g.bodyH = h - 12
		g.listW = 31
		g.entryX = g.x + 35
		g.entryW = w - 37
	} else if g.short {
		g.bodyY = g.y + 4
		g.bodyH = h - 9
	} else {
		g.bodyY = g.y + 6
		g.bodyH = h - 11
		if m.dex.detail {
			g.bodyY = g.y + 4
			if m.dex.filters {
				g.bodyY = g.y + 6
			}
			g.bodyH = g.y + h - 8 - g.bodyY
		}
	}
	if h >= 24 {
		g.bodyH -= 3
	}
	g.bodyH = max(3, g.bodyH)
	return g
}
func (m Model) dexListWidth() int   { return m.dexGeometry().listW - 4 }
func (m Model) dexDetailWidth() int { return m.dexGeometry().entryW - 4 }
func (m Model) dexIndexHeight() int { return max(1, m.dexGeometry().listH-2) }
func (m Model) dexBodyHeight() int  { return max(1, m.dexGeometry().bodyH-2) }
func (m Model) dexArtRect() (x, y, w, h int) {
	g := m.dexGeometry()
	if !g.wide {
		return g.entryX, g.bodyY, g.entryW, g.bodyH
	}
	return g.entryX + 2, g.bodyY + 3, max(34, (g.entryW-6)/2), max(3, g.bodyH-4)
}
func (m Model) dexMaxScroll() int {
	if !m.dexWide() && m.dex.tab == 0 && m.dex.overviewFacts {
		return max(0, len(m.compactEntryFacts())-m.dexBodyHeight())
	}
	if m.dex.tab == 0 {
		_, _, _, h := m.dexArtRect()
		return max(0, len(m.dex.artRows)-max(1, h-2))
	}
	return max(0, len(m.dexLines())-m.dexBodyHeight())
}
func (m Model) dexMaxHorizontal() int {
	if m.dex.tab != 0 || !m.dexWide() && m.dex.overviewFacts {
		return 0
	}
	_, _, w, _ := m.dexArtRect()
	return max(0, m.dex.artWidth-max(1, w-2))
}
func (m Model) dexControls() []dexControl { return m.pokedexControls() }

func (m Model) dexTabLabel(i int, label string) string {
	if m.dex.tab == i {
		return "[" + label + "]"
	}
	return "[" + label + "]"
}
func (m Model) dexStatusLabel(status int) string {
	label := []string{"All", "Seen", "Unseen"}[status]
	if status == m.dex.status {
		return "[" + label + "]"
	}
	return "[" + label + "]"
}
func (m Model) dexGenLabel() string {
	if m.dex.generationFilter == 0 {
		return "Gen: All"
	}
	return "Gen: " + strconv.Itoa(m.dex.generationFilter)
}
func (m Model) dexSearchLabel() string {
	if m.dex.query != "" {
		return "Search: " + clean(m.dex.query)
	}
	return "Search /"
}
func (m Model) dexTargets() []dexControl {
	g := m.dexGeometry()
	cs := m.dexControls()
	if g.wide || !m.dex.detail {
		cs = append(cs, dexControl{g.listX + 2, g.bodyY + g.bodyH/2, g.listW - 4, 0, ""})
	}
	if g.wide || m.dex.detail {
		cs = append(cs, dexControl{g.entryX + 2, g.bodyY + g.bodyH/2, g.entryW - 4, m.dexContentFocus(), ""})
	}
	return cs
}
func (m Model) dexFocusOrder() []int { return m.pokedexFocusOrder() }
func (m Model) dexLines() []dexLine {
	out := []dexLine{}
	add := func(s string) { out = append(out, dexLine{text: s}) }
	if m.dex.loading {
		add("Reading your collection…")
	} else if m.dex.error != "" {
		add("Pokédex unavailable")
		add(m.dex.error)
		add("Refresh to retry; Back to choose a trainer.")
	} else if len(m.dex.rows) == 0 {
		add("No matching entries")
		add("Change search or filters.")
	} else if m.dex.entryLoading {
		add("Reading entry…")
	} else if !m.dex.entry.Seen {
		add("UNDISCOVERED")
		add("An encounter can reveal this entry.")
	} else {
		switch m.dex.tab {
		case 1:
			return m.variantLines()
		case 2:
			return m.evolutionLines()
		case 3:
			return m.recordLines()
		}
	}
	width := max(1, m.dexDetailWidth())
	wrapped := []dexLine{}
	for _, line := range out {
		if line.target > 0 {
			wrapped = append(wrapped, line)
			continue
		}
		for _, row := range strings.Split(ansi.Wrap(line.text, width, ""), "\n") {
			wrapped = append(wrapped, dexLine{text: row})
		}
	}
	return wrapped
}
func dexDate(ms int64) string { return time.UnixMilli(ms).Local().Format("02 Jan 2006 15:04") }
func (m Model) paintDex(c *canvas) {
	if m.screen == dexSearchScreen {
		m.paintCreate(c)
		return
	}
	if !m.dexWide() {
		m.paintCompactDex(c)
		return
	}
	g := m.dexGeometry()
	p := m.palette()
	c.box(g.x, g.y, g.w, g.h, "", p.accent)
	brand := "POKÉCRT / POKÉDEX"
	if !g.wide {
		brand = "POKÉDEX"
		if g.short && m.dex.detail {
			brand = m.dexEntryTitle()
			if !m.dex.entry.Seen {
				brand += "   LOCKED"
			}
		}
	}
	c.put(g.x+2, g.y+1, brand, p.accent)
	if g.wide {
		c.put(g.x+2, g.y+1, "POKÉCRT", m.brandStyle())
	}

	if g.wide {
		c.put(g.x+23, g.y+1, "● ● ●", p.muted)
	}
	s := m.dex.snapshot.summary
	trainerLabel := clean(m.dex.snapshot.name)
	if trainerLabel == "" {
		trainerLabel = clean(m.snapshot.Name)
		if trainerLabel == "" {
			trainerLabel = "Reading trainer…"
		}
	}
	if g.wide {

		summary := fmt.Sprintf("%d / %d DISCOVERED", s.Completion.Species, s.Completion.SpeciesTotal)
		if m.dex.snapshot.data == nil {
			summary = "READING COLLECTION…"
			if m.dex.error != "" {
				summary = "COLLECTION UNAVAILABLE"
			}
		}
		summaryX := g.x + g.w - 2 - ansi.StringWidth(summary)
		trainerX := g.x + 34
		c.put(trainerX, g.y+1, ansi.Truncate("TRAINER "+trainerLabel, max(0, summaryX-trainerX-3), "…"), p.muted)
		c.put(summaryX, g.y+1, summary, p.muted)
	} else if !g.short {
		label := fmt.Sprintf("%s   %d/%d discovered", trainerLabel, s.Completion.Species, s.Completion.SpeciesTotal)
		if m.dex.detail {
			label = m.dexEntryTitle()
			if m.dex.entry.Seen {
				label += fmt.Sprintf("   GEN %02d", m.dex.entry.Generation)
			}
		}
		c.put(g.x+2, g.y+2, ansi.Truncate(label, g.w-4, "…"), p.muted)
	}
	if g.wide || !m.dex.detail {
		m.paintDexIndex(c, g)
	}
	if g.wide || m.dex.detail {
		if m.dex.tab == 0 {
			m.paintDexOverview(c, g)
		} else {
			m.paintDexTab(c, g)
		}
	}
	if !g.wide && m.dex.detail && !g.short {
		label, types := m.dexOverviewLabels()
		if m.dex.tab != 0 {
			label = dexTabNames[m.dex.tab]
			types = "Only discovered facts are shown."
		}
		c.put(g.x+2, g.bodyY+g.bodyH-2, ansi.Truncate(label, g.w-4, "…"), p.muted)
		c.put(g.x+2, g.bodyY+g.bodyH-1, ansi.Truncate(types, g.w-4, "…"), p.accent)
	}
	m.paintPokedexControls(c)
	m.paintPokedexHints(c)
}
func (m Model) dexEntryTitle() string {
	if len(m.dex.rows) == 0 {
		return "NO MATCHING ENTRIES"
	}
	r := m.dex.rows[m.dex.selected]
	return fmt.Sprintf("#%03d %s", r.Number, strings.ToUpper(clean(r.Name)))
}
func (m Model) paintDexIndex(c *canvas, g dexLayout) {
	p := m.palette()
	title := "NATIONAL INDEX"
	c.box(g.listX, g.bodyY, g.listW, g.listH, title, p.accent)
	x, y, w, h := g.listX+2, g.bodyY+1, g.listW-4, g.listH-2
	if m.dex.loading {
		c.wrap(x, y, w, h, "Reading collection…", p.muted)
		return
	}
	if m.dex.error != "" {
		c.wrap(x, y, w, h, "Collection unavailable. Refresh to retry.", p.muted)
		return
	}
	if len(m.dex.rows) == 0 {
		c.wrap(x, y, w, h, "No matches. Change search or filters.", p.muted)
		return
	}
	first := max(0, min(m.dex.selected-h/2, len(m.dex.rows)-h))
	for i := first; i < len(m.dex.rows) && i < first+h; i++ {
		r := m.dex.rows[i]
		marker := "  "
		style := ""
		if i == m.dex.selected {
			marker = "● "
			style = m.controlStyle(false, true)
			if m.focus == 0 {
				marker = "▶ "
				style = m.listFocusStyle()
			}
		}
		suffix := ""
		if !r.Eligible {
			suffix = "   unavailable"
		}
		label := ansi.Truncate(fmt.Sprintf("%s#%03d %s%s", marker, r.Number, clean(r.Name), suffix), w, "…")
		label += strings.Repeat(" ", max(0, w-ansi.StringWidth(label)))
		row := y + i - first
		c.put(x, row, label, style)
		c.hits = append(c.hits, hit{x, row, w, 1000 + i})
	}
}
func (m Model) dexOverviewLabels() (string, string) {
	e := m.dex.entry
	if !e.Seen {
		return "UNDISCOVERED", "An encounter can reveal this entry."
	}
	if e.SelectedName != "" {
		types := []string{}
		for _, t := range e.SelectedTypes {
			types = append(types, "[ "+strings.ToUpper(t)+" ]")
		}
		return strings.ToUpper(clean(e.SelectedName)), strings.Join(types, " ")
	}
	notice := e.Notice
	if notice == "" {
		notice = "Appearance locked"
	}
	return notice, "Choose a collected variant."
}
func (m Model) paintDexOverview(c *canvas, g dexLayout) {
	p := m.palette()
	if g.wide {
		c.box(g.entryX, g.bodyY, g.entryW, g.bodyH, "ENTRY", p.accent)
		c.put(g.entryX+2, g.bodyY+1, ansi.Truncate(m.dexEntryTitle(), g.entryW-4, "…"), p.accent)
		if m.dex.entry.Seen {
			c.put(g.entryX+2, g.bodyY+2, fmt.Sprintf("GEN %02d   DISCOVERED", m.dex.entry.Generation), p.muted)
		}
	}
	x, y, w, h := m.dexArtRect()
	title := "ARTWORK"
	c.box(x, y, w, h, title, p.accent)
	vx, vy, vw, vh := x+1, y+1, w-2, h-2
	if m.dex.art != "" {
		horizontal := min(m.dex.horizontal, m.dexMaxHorizontal())
		vertical := min(m.dex.scroll, m.dexMaxScroll())
		offset := max(0, (vw-m.dex.artWidth)/2)
		verticalPad := max(0, (vh-len(m.dex.artRows))/2)
		for i := vertical; i < len(m.dex.artRows) && i < vertical+vh; i++ {
			c.putANSI(vx+offset, vy+verticalPad+i-vertical, ansi.Cut(m.dex.artRows[i], horizontal, horizontal+vw))
		}
	} else {
		message := "UNDISCOVERED"
		if m.dex.loading || m.dex.entryLoading {
			message = "READING…"
		} else if m.dex.error != "" {
			message = "UNAVAILABLE"
		} else if len(m.dex.rows) == 0 {
			message = "NO MATCHES"
		} else if m.dex.entry.Seen {
			message = "LOCKED APPEARANCE"
			if m.dex.entry.Selected.Count > 0 {
				message = "ARTWORK UNAVAILABLE"
			}
			if m.dex.entryError != "" {
				message = "ARTWORK ERROR"
			}
		}
		c.put(vx+max(0, (vw-1)/2), vy+max(0, (vh-2)/2), "?", p.accent)
		if vh > 2 {
			c.put(vx+max(0, (vw-ansi.StringWidth(message))/2), vy+vh/2+1, ansi.Truncate(message, vw, "…"), p.muted)
		}
	}
	for row := vy; row < vy+vh; row++ {
		c.hits = append(c.hits, hit{vx, row, vw, 10})
	}
	if g.wide {
		fx := x + w + 2
		fw := g.entryX + g.entryW - fx - 2
		m.paintEntryFacts(c, fx, y, fw, h)
	} else if g.bodyH >= 24 {
		m.paintEntryFacts(c, x, y+h+1, w, g.bodyY+g.bodyH-y-h-1)
	}

}
func (m Model) paintDexTab(c *canvas, g dexLayout) {
	p := m.palette()
	title := strings.ToUpper(dexTabNames[m.dex.tab]) + " / " + m.dexEntryTitle()
	c.box(g.entryX, g.bodyY, g.entryW, g.bodyH, title, p.accent)
	lines := m.dexLines()
	h := g.bodyH - 2
	offset := min(m.dex.scroll, max(0, len(lines)-h))
	x, y, w := g.entryX+2, g.bodyY+1, g.entryW-4
	for i := offset; i < len(lines) && i < offset+h; i++ {
		line := lines[i]
		row := y + i - offset
		if line.art {
			c.putANSI(x, row, ansi.Truncate(line.text, w, "…"))
			if len(line.nodes) == 0 {
				c.hits = append(c.hits, hit{x, row, w, m.dexContentFocus()})
			}
			for _, node := range line.nodes {
				c.hits = append(c.hits, hit{x + node.x, row, node.w, node.id})
			}
		} else if line.target > 0 {
			copy := m
			if line.target >= 3000 {
				copy.focus = -1
				if i == m.dex.optionIndex && m.focus == 15 {
					copy.focus = line.target
				}
			}
			c.button(x, row, w, line.target, line.text, copy)
		} else {
			c.put(x, row, ansi.Truncate(line.text, w, "…"), "")
			c.hits = append(c.hits, hit{x, row, w, m.dexContentFocus()})
		}
	}
}

// Reuse the core renderer's SGR stream; reset restores the frame palette.
func (c *canvas) putANSI(x, y int, s string) {
	style := ""
	for len(s) > 0 {
		if strings.HasPrefix(s, "\x1b[") {
			end := strings.IndexByte(s, 'm')
			if end < 0 {
				return
			}
			seq := s[:end+1]
			if seq == "\x1b[0m" {
				style = ""
			} else {
				style += seq
			}
			s = s[end+1:]
			continue
		}
		end := strings.IndexByte(s, '\x1b')
		if end < 0 {
			end = len(s)
		}
		if end == 0 {
			return
		}
		text := s[:end]
		c.put(x, y, text, style)
		x += ansi.StringWidth(text)
		s = s[end:]
	}
}

func dexEncounterLabel(count int64) string {
	if count == 1 {
		return "1 ENCOUNTER"
	}
	return fmt.Sprintf("%d ENCOUNTERS", count)
}
