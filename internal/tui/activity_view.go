package tui

import (
	"fmt"
	"github.com/charmbracelet/x/ansi"
	"strings"
)

func (m Model) activityBodyHeight() int {
	h := m.dexGeometry().h
	if h < 18 {
		return max(4, h-8)
	}
	return max(3, h-11)
}
func (m Model) activityLines() []string {
	a := m.activity
	d := a.data
	switch m.section {
	case 1:
		if m.activity.historyMode {
			lines := []string{"LATEST 50 · stored discoveries", ""}
			if len(d.history) == 0 {
				return append(lines, "Your journey starts with Encounter.")
			}
			for i, h := range d.history {
				marker := "  "
				if i == a.selected {
					marker = "▶ "
				}
				lines = append(lines, fmt.Sprintf("%s#%03d %s", marker, h.Key.SpeciesID, clean(h.Snapshot.SpeciesName)), fmt.Sprintf("  %s · %s · +%d XP", clean(h.Snapshot.FormName), h.Key.Palette, h.XP), "  "+dexDate(h.EncounteredAtMS))
			}
			return lines
		}
		if a.result == nil {
			return []string{"READY FOR AN ENCOUNTER", "", "Choose Encounter to discover a Pokémon.", "Browsing never records a discovery."}
		}
		r := a.result
		s := r.Choice.Snapshot()
		k := r.Choice.Key()
		lines := []string{fmt.Sprintf("#%03d %s", k.SpeciesID, strings.ToUpper(clean(s.SpeciesName))), strings.ToUpper(clean(s.FormName) + " · " + k.Palette), "", "COMMITTED · saved to your trainer", fmt.Sprintf("+%d XP · LEVEL %d", r.XPAwarded, r.After.Level)}
		if r.FirstSpecies {
			lines = append(lines, "NEW SPECIES DISCOVERED")
		}
		if r.FirstVariant {
			lines = append(lines, "NEW APPEARANCE COLLECTED")
		}
		lines = append(lines, dexDate(r.EncounteredAtMS))
		for _, u := range r.NewUnlocks() {
			lines = append(lines, "UNLOCKED · "+clean(u.Name))
		}
		return lines
	case 2:
		s := d.stats
		p := s.Progress
		filled := int(p.InLevel * 20 / 1000)
		lines := []string{strings.ToUpper(clean(d.profile.Name)), "Created " + dexDate(d.profile.CreatedAtMS), "", fmt.Sprintf("LEVEL %d · %d XP", p.Level, p.Total), "[" + strings.Repeat("━", filled) + strings.Repeat("·", 20-filled) + "]", fmt.Sprintf("%d / 1000 · %d XP to next level", p.InLevel, p.ToNext), "", fmt.Sprintf("%d ENCOUNTERS", s.Encounters), fmt.Sprintf("%d SPECIES · %d APPEARANCES", s.Species, s.Variants), fmt.Sprintf("%d SHINY COLLECTIONS · %d shiny encounters", s.ShinyCollections, s.ShinyEncounters), "", fmt.Sprintf("ELIGIBLE SPECIES %d / %d", s.Completion.Species, s.Completion.SpeciesTotal), fmt.Sprintf("ELIGIBLE APPEARANCES %d / %d", s.Completion.Variants, s.Completion.VariantsTotal), "", "GENERATION PROGRESS"}
		for _, g := range s.Generations {
			lines = append(lines, fmt.Sprintf("GEN %02d · %d / %d eligible · %d discovered", g.Generation, g.Eligible, g.EligibleTotal, g.Discovered))
		}
		if s.UnclassifiedSpecies > 0 {
			lines = append(lines, fmt.Sprintf("%d retained discoveries outside current catalog", s.UnclassifiedSpecies))
		}
		if s.FirstEncounterMS != nil {
			lines = append(lines, "", "First encounter "+dexDate(*s.FirstEncounterMS))
		}
		if s.LastEncounterMS != nil {
			lines = append(lines, "Last encounter "+dexDate(*s.LastEncounterMS))
		}
		return lines
	default:
		lines := []string{fmt.Sprintf("%d EARNED · %d LOCKED", len(d.achievements.Unlocked), len(d.achievements.Locked)), ""}
		for _, g := range d.achievements.Unlocked {
			lines = append(lines, "● "+clean(g.Name), "  "+clean(g.Description), "  Earned "+dexDate(g.EarnedAtMS), "")
		}
		for _, g := range d.achievements.Locked {
			progress := "Unavailable in current inventory"
			if g.HasTarget {
				progress = fmt.Sprintf("%d / %d", g.Current, g.Target)
			}
			lines = append(lines, "◇ "+clean(g.Name), "  "+clean(g.Description), "  "+progress, "")
		}
		return lines
	}
}
func (m Model) activityWrapped() []string {
	g := m.dexGeometry()
	w := g.w - 8
	if g.wide && m.section != 1 {
		_, right := m.activityColumns()
		return wrapActivityLines(right, g.w/2-6)
	}
	if m.section == 1 && g.wide && !m.activity.historyMode {
		w = g.w/2 - 6
	}
	var lines []string
	for _, line := range m.activityLines() {
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
	g := m.dexGeometry()
	out := []dexControl{}
	add := func(x, y, w, id int, label string) { out = append(out, dexControl{x, y, w, id, label}) }
	y := g.y + 3
	if m.section == 1 {
		label := "Encounter"
		if m.activity.recording {
			label = "Saving…"
		}
		if m.activity.loading {
			label = "Reading…"
		}
		add(g.x+2, y, 14, 10, label)
		add(g.x+16, y, g.w-18, 30, map[bool]string{true: "Show result", false: "History"}[m.activity.historyMode])
	} else if m.section == 2 {
		add(g.x+2, y, g.w-4, 15, "Choose / create trainer")
	}
	// Content focus and explicit mouse scroll controls stay available at 40×12.
	if !(m.section == 1 && m.activity.historyMode && len(m.activity.data.history) > 0) {
		add(g.x+2, g.y+g.h-6, min(18, g.w-14), 22, "Scroll")
	}
	add(g.x+g.w-12, g.y+g.h-6, 5, 16, "↑")
	add(g.x+g.w-7, g.y+g.h-6, 5, 17, "↓")
	if m.section == 1 {
		if m.activity.historyMode && len(m.activity.data.history) > 0 {
			add(g.x+2, g.y+g.h-6, g.w-14, 20, "Open selected entry")
			add(g.x+2, g.y+g.h-5, (g.w-4)/2, 31, "Previous")
			add(g.x+2+(g.w-4)/2, g.y+g.h-5, (g.w-4)/2, 32, "Next")
		} else if len(m.activity.art) > 0 {

			add(g.x+2, g.y+g.h-5, 8, 21, "Art")
			add(g.x+10, g.y+g.h-5, 5, 18, "←")
			add(g.x+15, g.y+g.h-5, 5, 19, "→")
			if !g.wide {
				add(g.x+30, g.y+g.h-5, g.w-32, 35, "Info")
			}
			add(g.x+20, g.y+g.h-5, 5, 33, "↑")
			add(g.x+25, g.y+g.h-5, 5, 34, "↓")
		}
	}
	labels := []string{"Back", "Theme", "Read", "Quit"}
	ids := []int{12, 13, 11, 14}
	q := (g.w - 4) / 4
	for i, id := range ids {
		add(g.x+2+i*q, g.y+g.h-3, q, id, labels[i])
	}
	return out
}
func (m Model) paintActivity(c *canvas) {
	g := m.dexGeometry()
	p := m.palette()
	a := m.activity
	c.box(g.x, g.y, g.w, g.h, "", p.accent)
	title := strings.ToUpper(sections[m.section])
	c.put(g.x+2, g.y+1, "POKÉCRT / "+title, p.accent)
	if g.wide {
		c.put(g.x+g.w/2, g.y+1, "TRAINER "+clean(a.data.profile.Name), p.muted)
	}
	c.put(g.x+1, g.y+2, strings.Repeat("─", g.w-2), p.accent)
	c.put(g.x+1, g.y+g.h-4, strings.Repeat("─", g.w-2), p.accent)
	if m.section == 1 && a.result != nil && g.h >= 18 {
		k := a.result.Choice.Key()
		c.put(g.x+2, g.y+4, ansi.Truncate(fmt.Sprintf("#%03d %s · +%d XP", k.SpeciesID, strings.ToUpper(clean(a.result.Choice.Snapshot().SpeciesName)), a.result.XPAwarded), g.w-4, "…"), p.muted)
	} else if g.h >= 18 && m.encounterPending {
		c.put(g.x+2, g.y+4, "Saving encounter…", p.muted)
	} else if g.h >= 18 && m.encounterNotice != "" {
		c.put(g.x+2, g.y+4, m.encounterNotice, p.muted)
	}
	bodyY, bodyH := g.y+5, m.activityBodyHeight()
	bodyW := g.w - 4
	if g.h < 18 {
		bodyY = g.y + 3
	}
	if g.h >= 18 {
		c.box(g.x+2, bodyY, bodyW, bodyH, " "+map[int]string{1: "ENCOUNTER LOG", 2: "TRAINER CARD", 3: "ACHIEVEMENT JOURNAL"}[m.section]+" ", p.accent)
	}
	textX, textW := g.x+4, bodyW-4
	if g.wide && m.section != 1 {
		half := g.w/2 - 3
		c.box(g.x+2, bodyY, half, bodyH, " "+map[int]string{2: "TRAINER CARD", 3: "EARNED BADGES"}[m.section]+" ", p.accent)
		c.box(g.x+g.w/2, bodyY, g.w/2-2, bodyH, " "+map[int]string{2: "GENERATION PROGRESS", 3: "NEXT GOALS"}[m.section]+" ", p.accent)
		left, _ := m.activityColumns()
		rows := wrapActivityLines(left, half-4)
		for i := 0; i < bodyH-2 && i+a.scroll < len(rows); i++ {
			c.put(g.x+4, bodyY+1+i, rows[i+a.scroll], p.foreground)
		}
		textX = g.x + g.w/2 + 2
		textW = g.w/2 - 6
	}
	if m.section == 1 && !a.historyMode && len(a.art) > 0 && a.error == "" && g.wide {
		artW := g.w/2 - 4
		c.box(g.x+2, bodyY, artW, bodyH, " DEVICE DISPLAY ", p.accent)
		m.paintActivityArt(c, g.x+3, bodyY+1, artW-2, bodyH-2)
		textX = g.x + g.w/2 + 1
		textW = g.w/2 - 5
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
	if textW > 0 {
		for i := 0; i < bodyH-2 && i+a.scroll < len(lines); i++ {
			c.put(textX, bodyY+1+i, ansi.Truncate(lines[i+a.scroll], textW, "…"), p.foreground)
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
		c.button(control.x, control.y, control.w, control.id, label, m)
	}

	c.put(g.x+2, g.y+g.h-2, "↑↓←→ Focus · Tab · Enter · Esc Back", p.muted)
}

func (m Model) activityMaxPan() int {
	w := m.dexGeometry().w - 6
	if m.dexGeometry().wide {
		w = m.dexGeometry().w/2 - 6
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
		if i >= 2 && len(m.activity.data.history) > 0 {
			owner = (i - 2) / 3
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
	lines := m.activityLines()
	if m.section == 2 {
		return lines[:14], lines[15:]
	}
	split := 2 + len(m.activity.data.achievements.Unlocked)*4
	left = append(left, lines[:split]...)
	if len(m.activity.data.achievements.Unlocked) == 0 {
		left = append(left, "Your first badge is waiting.", "Choose Encounter to start.")
	}
	return left, lines[split:]
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
