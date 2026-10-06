package tui

import (
	"fmt"
	"github.com/charmbracelet/x/ansi"
	"strings"
)

func (m Model) activityBodyHeight() int {
	if m.section == 1 {
		return m.encounterGeometry().bodyH
	}
	if !m.dexWide() {
		if m.section != 1 {
			reserve := 6
			if m.height >= 24 && m.width >= 56 {
				reserve = 8
			}
			return max(3, m.height-reserve-m.compactActivityBodyY())
		}
		return max(3, m.height-4-m.compactActivityBodyY())
	}
	h := m.dexGeometry().h
	if h < 18 {
		return max(4, h-8)
	}
	if h >= 24 {
		height := h - 14
		if m.section == 1 {
			height--
		}
		return max(3, height)
	}
	height := h - 11
	if m.section != 1 {
		height -= 2
	}
	if m.section == 1 && h >= 20 {
		height--
	}
	return max(3, height)
}
func (m Model) activityLines() []string {
	a := m.activity
	d := a.data
	switch m.section {
	case 1:
		if m.activity.historyMode {
			lines := []string{}
			if len(d.history) == 0 {
				return append(lines, "Your journey starts with Encounter.")
			}
			for i, h := range d.history {
				marker := "  "
				if i == a.selected {
					marker = "› "
				}
				titleStyle := m.palette().gold
				if i == a.selected {
					titleStyle = m.palette().muted
				}
				lines = append(lines, styled(fmt.Sprintf("%s#%03d %s", marker, h.Key.SpeciesID, clean(h.Snapshot.SpeciesName)), titleStyle), fmt.Sprintf("  %s   %s   +%d XP", clean(h.Snapshot.FormName), h.Key.Palette, h.XP), styled("  "+dexDate(h.EncounteredAtMS), m.hintStyle()), "")
			}
			return lines
		}
		return m.encounterInformation()
	case 2:
		left, right := m.trainerInformationColumns()
		return append(append(left, ""), right...)
	default:
		left, right := m.achievementInformationColumns()
		return append(append(left, ""), right...)
	}

}
func (m Model) activityWrapped() []string {
	g := m.dexGeometry()
	w := g.w - 8
	if !g.wide && m.activity.error != "" {
		return wrapActivityLines([]string{m.activity.error}, w)
	}
	if g.wide && m.section != 1 {
		_, right := m.activityColumns()
		return wrapActivityLines(right, g.w/2-6)
	}
	if m.section == 1 && g.wide && !m.activity.historyMode && len(m.activity.art) > 0 && m.activity.error == "" {
		w = g.w/2 - 6
	}
	var lines []string
	source := m.activityLines()
	if m.section == 3 && !g.wide && m.height < 14 && len(source) > 2 {
		source = source[2:]
	}
	for _, line := range source {
		if m.section == 3 && !g.wide && m.height < 16 && strings.TrimSpace(ansi.Strip(line)) == "" {
			continue
		}
		lines = append(lines, strings.Split(ansi.Wrap(line, max(1, w), ""), "\n")...)
	}
	return lines
}
func (m Model) activityMaxScroll() int {
	n := len(m.activityWrapped())
	if m.dexGeometry().wide && m.section != 1 {
		left, _ := m.activityColumns()
		n = max(n, len(wrapActivityLines(left, m.dexGeometry().w/2-6)))
	}
	return max(0, n-(m.activityBodyHeight()-2))
}
func (m Model) activityControls() []dexControl {
	if m.section == 1 {
		return m.encounterControls()
	}
	if !m.dexWide() {
		return m.compactActivityControls()
	}
	g := m.dexGeometry()
	out := []dexControl{}
	add := func(x, y, w, id int, label string) {
		if g.h >= 24 && y >= g.y+g.h-6 && y < g.y+g.h-3 {
			y -= 3
		}
		if g.h >= 24 && y == g.y+g.h-3 {
			y -= 3
		}
		if m.section != 1 && g.h < 24 && (id == 12 || id == 13 || id == 11 || id == 14) {
			y -= 2
		}
		out = append(out, dexControl{x, y, w, id, label})
	}
	y := g.y + 3
	if m.section == 1 {
		label := "Encounter"
		if m.activity.recording {
			label = "Saving…"
		}
		if m.activity.loading {
			label = "Reading…"
		}
		add(g.x+2, y, 16, 10, label)
		add(g.x+18, y, g.w-20, 30, map[bool]string{true: "Show result", false: "History"}[m.activity.historyMode])
	} else if m.section == 2 {
		add(g.x+2, y, 29, 15, "Choose / create trainer")
	}
	if m.section != 1 && g.wide {
		for panel := 0; panel < 2; panel++ {
			px := g.x + 2
			if panel == 1 {
				px = g.x + g.w/2
			}
			pw := g.w/2 - 3
			if m.activityPanelMaxScroll(panel) > 0 {
				bottom := g.y + 5 + m.activityBodyHeight() - 1
				add(px+1, bottom, 10, 40+panel, "Scroll")
				upX := px + pw - 12
				if m.section != 1 {
					upX--
				}
				add(upX, bottom, 5, 42+panel*2, "↑")
				add(px+pw-7, bottom, 5, 43+panel*2, "↓")
			}
		}
	} else if m.activityMaxScroll() > 0 {
		if !(m.section == 1 && m.activity.historyMode && len(m.activity.data.history) > 0) {
			add(g.x+2, g.y+g.h-6, min(13, g.w-14), 22, "Scroll")
		}
		add(g.x+g.w-12, g.y+g.h-6, 5, 16, "↑")
		add(g.x+g.w-7, g.y+g.h-6, 5, 17, "↓")
	}
	if m.section == 1 {
		if m.activity.historyMode && len(m.activity.data.history) > 0 {
			add(g.x+2, g.y+g.h-6, min(26, g.w-14), 20, "Open selected entry")
			add(g.x+2, g.y+g.h-5, min(15, (g.w-4)/2), 31, "Previous")
			add(g.x+2+min(15, (g.w-4)/2), g.y+g.h-5, min(11, (g.w-4)/2), 32, "Next")
		} else if len(m.activity.art) > 0 {

			pan := m.activityMaxPan() > 0
			vertical := len(m.activity.art) > m.activityBodyHeight()-2
			if pan || vertical {
				add(g.x+2, g.y+g.h-5, 8, 21, "Art")
			}
			if pan {
				add(g.x+10, g.y+g.h-5, 5, 18, "←")
				add(g.x+15, g.y+g.h-5, 5, 19, "→")
			}
			if vertical {
				add(g.x+20, g.y+g.h-5, 5, 33, "↑")
				add(g.x+25, g.y+g.h-5, 5, 34, "↓")
			}
			if !g.wide {
				add(g.x+30, g.y+g.h-5, g.w-32, 35, "Info")
			}

		}
	}
	labels := []string{"Back", "Theme", "Refresh", "Quit"}
	ids := []int{12, 13, 11, 14}
	q := min(14, (g.w-4)/4)
	for i, id := range ids {
		add(g.x+2+i*q, g.y+g.h-3, q, id, labels[i])
	}
	return out
}
func (m Model) paintActivity(c *canvas) {
	if m.section == 1 {
		m.paintEncounter(c)
		return
	}
	if !m.dexWide() {
		m.paintCompactActivity(c)
		return
	}
	g := m.dexGeometry()
	p := m.palette()
	a := m.activity
	c.box(g.x, g.y, g.w, g.h, "", p.accent)
	title := strings.ToUpper(sections[m.section])
	c.put(g.x+2, g.y+1, "POKÉCRT / "+title, p.accent)
	c.put(g.x+2, g.y+1, "POKÉCRT", m.brandStyle())
	if g.wide {
		c.put(g.x+g.w/2, g.y+1, "TRAINER "+clean(a.data.profile.Name), p.muted)
	}

	statusY := g.y + 4
	if m.section == 1 && !g.short {
		statusY++
	}
	if m.section == 1 && a.result != nil && g.h >= 18 {
		k := a.result.Choice.Key()
		c.put(g.x+2, statusY, ansi.Truncate(fmt.Sprintf("#%03d %s   +%d XP", k.SpeciesID, strings.ToUpper(clean(a.result.Choice.Snapshot().SpeciesName)), a.result.XPAwarded), g.w-4, "…"), p.muted)
	} else if m.section == 1 && g.h >= 18 && m.encounterPending {
		c.put(g.x+2, statusY, "Saving encounter…", p.muted)
	} else if m.section == 1 && g.h >= 18 && m.encounterNotice != "" {
		c.put(g.x+2, statusY, m.encounterNotice, p.muted)
	}
	bodyY, bodyH := g.y+5, m.activityBodyHeight()
	if m.section == 1 && !g.short {
		bodyY++
	}
	bodyW := g.w - 4
	if g.h < 18 {
		bodyY = g.y + 3
	}
	if g.h >= 18 && !(g.wide && (m.section != 1 || !a.historyMode && len(a.art) > 0 && a.error == "")) {
		c.box(g.x+2, bodyY, bodyW, bodyH, " "+map[int]string{1: "ENCOUNTER LOG", 2: "TRAINER CARD", 3: "ACHIEVEMENT JOURNAL"}[m.section]+" ", m.achievementFrameStyle(p.accent))
	}
	textX, textW := g.x+4, bodyW-4

	if g.wide && m.section != 1 {
		half := g.w/2 - 3
		c.box(g.x+2, bodyY, half, bodyH, " "+map[int]string{2: "TRAINER CARD", 3: "EARNED BADGES"}[m.section]+" ", m.informationFrameStyle(0, p.accent))
		c.box(g.x+g.w/2, bodyY, g.w/2-2, bodyH, " "+map[int]string{2: "GENERATION PROGRESS", 3: "NEXT GOALS"}[m.section]+" ", m.informationFrameStyle(1, p.accent))
		left, _ := m.activityColumns()
		rows := wrapActivityLines(left, half-4)
		leftScroll := a.scroll
		if m.section != 1 {
			leftScroll = min(a.panelScroll[0], m.activityPanelMaxScroll(0))
		}
		for i := 0; i < bodyH-2 && i+leftScroll < len(rows); i++ {
			c.putANSI(g.x+4, bodyY+1+i, rows[i+leftScroll])
			if m.section != 1 && m.activityPanelMaxScroll(0) > 0 {
				c.hits = append(c.hits, hit{g.x + 4, bodyY + 1 + i, half - 4, 40})
			}
		}
		textX = g.x + g.w/2 + 2
		textW = g.w/2 - 6
	}
	if m.section == 1 && !a.historyMode && len(a.art) > 0 && a.error == "" && g.wide {
		artW := g.w/2 - 4
		c.box(g.x+2, bodyY, artW, bodyH, " DEVICE DISPLAY ", p.accent)
		m.paintActivityArt(c, g.x+3, bodyY+1, artW-2, bodyH-2)
		detailsX := g.x + g.w/2
		c.box(detailsX, bodyY, g.w/2-2, bodyH, "ENCOUNTER DETAILS", p.accent)
		textX = detailsX + 2
		textW = g.w/2 - 6

	} else if m.section == 1 && !a.historyMode && len(a.art) > 0 && a.error == "" {
		// Compact results retain the full natural sprite in a pannable viewport.
		if !a.resultDetails {
			m.paintActivityArt(c, g.x+3, bodyY+1, bodyW-2, bodyH-2)
			textW = 0
		}
	}
	lines := m.activityWrapped()
	if a.loading && a.data.profile.ID == 0 {
		lines = []string{"Reading trainer records…"}
	} else if a.error != "" {
		lines = strings.Split(ansi.Wrap(a.error, max(1, textW), ""), "\n")
	}
	var historyTargets []int
	if m.section == 1 && a.historyMode && !a.loading && a.error == "" {
		historyTargets = m.activityHistoryTargets()
	}
	scroll := a.scroll
	if m.section != 1 && g.wide {
		scroll = min(a.panelScroll[1], m.activityPanelMaxScroll(1))
	}
	if textW > 0 {
		for i := 0; i < bodyH-2 && i+scroll < len(lines); i++ {
			c.putANSI(textX, bodyY+1+i, ansi.Truncate(lines[i+scroll], textW, "…"))
			if m.section != 1 && g.wide && m.activityPanelMaxScroll(1) > 0 {
				c.hits = append(c.hits, hit{textX, bodyY + 1 + i, textW, 41})
			}
			if m.section == 1 && a.historyMode && !a.loading && a.error == "" {
				targets := historyTargets
				if i+a.scroll < len(targets) && targets[i+a.scroll] >= 0 {
					c.hits = append(c.hits, hit{textX, bodyY + 1 + i, textW, 1000 + targets[i+a.scroll]})
				}
			}
		}
	}
	for _, control := range m.activityControls() {
		label := control.label
		if label != "" {
			label = "[" + label + "]"
		}
		if control.id >= 40 && control.id <= 45 {
			c.button(control.x, control.y, control.w, control.id, control.label, m)
		} else if !g.short && control.y == g.y+3 {
			width := max(3, control.w-1)
			if m.section == 2 && control.id == 15 {
				width = control.w
			}
			c.framedControl(control.x, control.y-1, width, control.id, label, m, control.id == 30 && m.activity.historyMode)
		} else if !g.short && (control.y == g.y+g.h-3 || g.h >= 24 && control.y == g.y+g.h-6) {
			if m.section == 2 {
				c.trainerOutlinedButton(control.x, control.y-1, max(3, control.w-1), control.id, label, m)
			} else {
				c.outlinedButton(control.x, control.y-1, max(3, control.w-1), control.id, label, m)
			}
		} else {
			width := max(3, control.w-1)
			if control.label == "↑" || control.label == "↓" || control.label == "←" || control.label == "→" {
				width = control.w
			}
			c.button(control.x, control.y, width, control.id, label, m)
		}
	}

	if m.section != 1 {
		hintY := g.y + g.h - 3
		if g.h >= 24 {
			hintY = g.y + g.h - 4
		}
		m.paintAchievementHints(c, g.x+2, hintY, g.w-4)
	} else if g.h >= 24 {
		c.navigationHints(g.x+2, g.y+g.h-4, g.w-4, m, false, "Select")
	} else {
		c.navigationHints(g.x+2, g.y+g.h-2, g.w-4, m, true, "Select")
	}
}

func (m Model) activityMaxPan() int {
	w := m.dexGeometry().w - 6
	if m.dexGeometry().wide {
		w = m.dexGeometry().w/2 - 5
	}
	maxWidth := 0
	for _, line := range m.activity.art {
		maxWidth = max(maxWidth, ansi.StringWidth(line))
	}
	return max(0, maxWidth-w)
}

func (m Model) activityHistoryTargets() []int {
	g := m.dexGeometry()
	w := g.w - 8
	var out []int
	for i, line := range m.activityLines() {
		owner := -1
		if len(m.activity.data.history) > 0 {
			owner = i / 4
		}
		for range strings.Split(ansi.Wrap(line, max(1, w), ""), "\n") {
			out = append(out, owner)
		}
	}
	return out
}

func wrapActivityLines(lines []string, w int) []string {
	var out []string
	for _, line := range lines {
		out = append(out, strings.Split(ansi.Wrap(line, max(1, w), ""), "\n")...)
	}
	return out
}
func (m Model) activityColumns() (left, right []string) {
	if m.section == 2 {
		return m.trainerInformationColumns()
	}
	return m.achievementInformationColumns()
}
func (m Model) paintActivityArt(c *canvas, x, y, w, h int) {
	a := m.activity
	maxW := 0
	for _, line := range a.art {
		maxW = max(maxW, ansi.StringWidth(line))
	}
	dx, dy := max(0, (w-maxW)/2), max(0, (h-len(a.art))/2)
	for i := 0; i < h-dy && i+a.artScroll < len(a.art); i++ {
		c.putANSI(x+dx, y+dy+i, ansi.Cut(a.art[i+a.artScroll], a.artPan, a.artPan+w-dx))
	}
}

func (m Model) achievementFrameStyle(fallback string) string {
	if m.section != 1 {
		return m.achievementPanelStyle(0)
	}
	return fallback
}
func (m Model) informationFrameStyle(panel int, fallback string) string {
	if m.section != 1 {
		return m.achievementPanelStyle(panel)
	}
	return fallback
}
