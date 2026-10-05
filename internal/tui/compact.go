package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/mayanklad/pokecrt/internal/catalog"
)

// Compact pages reserve actual rows instead of shifting individual controls.
func (m Model) compactDexGeometry() dexLayout {
	w, h := max(1, m.width), max(1, m.height)
	g := dexLayout{w: w, h: h, short: h < 20, listX: 2, listW: w - 4, entryX: 2, entryW: w - 4}
	if m.dex.detail {
		g.bodyY = 4
		if g.short {
			g.bodyY = 3
		}
		if m.dex.filters {
			g.bodyY += 2
		}
		bottom := h - 5
		if h >= 24 {
			bottom--
		}
		g.bodyH = max(3, bottom-g.bodyY+1)
	} else {
		g.bodyY = 5
		if g.short {
			g.bodyY = 4
		}
		g.bodyH = max(3, h-4-g.bodyY)
	}
	return g
}

func compactActions(x, y, w int, ids []int, labels []string) []dexControl {
	out := []dexControl{}
	for i, label := range labels {
		width := ansi.StringWidth(label) + 3
		out = append(out, dexControl{x: x, y: y, w: width, id: ids[i], label: label})
		x += width + 1
	}
	return out
}

func (m Model) compactDexControls() []dexControl {
	g := m.dexGeometry()
	if m.width < 40 || m.height < 12 {
		return []dexControl{{0, max(0, m.height-1), min(12, m.width), 8, "Quit"}}
	}
	cs := []dexControl{}
	add := func(x, y, w, id int, label string) { cs = append(cs, dexControl{x, y, w, id, label}) }
	filters := func(y int) {
		if g.w >= 64 {
			add(2, y, 12, 31, "All")
			add(14, y, 13, 32, "Seen")
			add(27, y, 15, 33, "Unseen")
			add(42, y, g.w-44, 3, m.dexGenLabel())
		} else {
			add(2, y, 15, 2, []string{"All", "Seen", "Unseen"}[m.dex.status])
			add(17, y, g.w-19, 3, m.dexGenLabel())
		}
		add(2, y+1, g.w-4, 1, m.dexSearchLabel())
	}
	if !m.dex.detail {
		y := 3
		if g.short {
			y = 2
		}
		filters(y)
		if len(m.dex.rows) > 0 {
			add(2, g.h-4, g.w-4, 4, "Open entry")
			if len(m.dex.rows) > m.dexBodyHeight() {
				add(g.listX+g.listW-12, g.bodyY+g.bodyH-1, 5, 16, "↑")
				add(g.listX+g.listW-7, g.bodyY+g.bodyH-1, 5, 17, "↓")
			}
		}
	} else {
		y := 3
		if g.short {
			y = 2
		}
		third := (g.w - 4) / 3
		add(2, y, third, 4, "Index")
		if !m.dex.filters {
			add(2+third, y, third, 1, "Search /")
		}
		filterLabel := "Filters"
		if m.dex.filters {
			filterLabel = "Close"
		}
		add(2+2*third, y, g.w-4-2*third, 25, filterLabel)
		if m.dex.filters {
			filters(y + 1)
		}
		if g.h >= 24 {
			half := (g.w - 4) / 2
			for i, label := range dexTabNames {
				add(2+(i%2)*half, g.h-5+i/2, half, 20+i, label)
			}
		} else {
			add(2, g.h-4, g.w-4, 24, "View: "+dexTabNames[m.dex.tab]+" ▸")
		}
		y = g.bodyY + g.bodyH - 1
		if m.dexMaxScroll() > 0 {
			add(4, y, 5, 11, "↑")
			add(9, y, 5, 12, "↓")
		}
		if m.dexMaxHorizontal() > 0 {
			add(14, y, 5, 13, "←")
			add(19, y, 5, 14, "→")
		}
		if m.dex.tab == 0 {
			label := "Facts"
			if m.dex.overviewFacts {
				label = "Art"
			}
			add(g.entryX+g.entryW-10, y, 8, 26, label)
		}
		if m.dex.tab == 2 && m.dex.entry.Seen && len(m.dex.entry.Evolution) > 0 {
			add(g.entryX+g.entryW-12, y, 10, 27, "Full art")
		}
	}
	back, theme := "Back", "Appearance"
	if m.dex.familyOrigin > 0 {
		back = "Family"
	}
	if g.w < 48 {
		theme = "Theme"
	}
	return append(cs, compactActions(2, g.h-3, g.w-4, []int{6, 7, 5, 8}, []string{back, theme, "Reload", "Quit"})...)
}

func (m Model) compactEntryFacts() []dexLine {
	w := m.dexDetailWidth()
	e := m.dex.entry
	out := []dexLine{}
	add := func(s string) { out = append(out, dexLine{text: s, art: true}) }
	if m.dex.loading || m.dex.entryLoading {
		add("Reading entry…")
		return out
	}
	if m.dex.error != "" {
		add(m.dex.error)
		return wrapDexInformation(out, w)
	}
	if !e.Seen {
		add(m.informationHeading("UNDISCOVERED", w))
		add("An encounter reveals this entry.")
		add("Identity and artwork remain hidden.")
		return wrapDexInformation(out, w)
	}
	add(m.informationHeading("APPEARANCE", w))
	add(strings.ReplaceAll(clean(e.SelectedName), " · ", " / "))
	badges := []string{}
	for _, kind := range e.SelectedTypes {
		badges = append(badges, m.typeBadge(kind))
	}
	add(strings.Join(badges, " "))
	add("")
	add(m.informationHeading("SPECIES PROFILE", w))
	add(m.informationFact("Generation", fmt.Sprint(e.Generation), w))
	add(m.informationFact("Evolution stage", fmt.Sprint(e.Stage), w))
	add(m.informationFact("Colour", strings.ToUpper(e.Color), w))
	add("")
	add(m.informationHeading("YOUR DISCOVERY", w))
	add(m.informationFact("Encounters", fmt.Sprint(e.Discovery.Count), w))
	add(m.informationFact("First discovered", dexDate(e.Discovery.FirstMS), w))
	add(m.informationFact("Last encountered", dexDate(e.Discovery.LastMS), w))
	if e.Notice != "" {
		add("")
		add(e.Notice)
	}
	if m.dex.entryError != "" {
		add("")
		add(m.dex.entryError)
	}
	return wrapDexInformation(out, w)
}

func (m Model) paintCompactDex(c *canvas) {
	g, p := m.dexGeometry(), m.palette()
	c.box(0, 0, g.w, g.h, "", p.accent)
	brand := "POKÉCRT / POKÉDEX"
	if g.short && m.dex.detail {
		brand = m.dexEntryTitle()
		if !m.dex.entry.Seen {
			brand += " LOCKED"
		}
	}
	c.put(2, 1, ansi.Truncate(brand, g.w-4, "…"), p.accent)
	if strings.HasPrefix(brand, "POKÉCRT") {
		c.put(2, 1, "POKÉCRT", m.brandStyle())
	}
	if !g.short {
		label := m.dexEntryTitle()
		if !m.dex.detail {
			s := m.dex.snapshot.summary
			label = fmt.Sprintf("%s  %d/%d discovered", clean(m.dex.snapshot.name), s.Completion.Species, s.Completion.SpeciesTotal)
		}
		c.put(2, 2, ansi.Truncate(label, g.w-4, "…"), p.muted)
	}
	if !m.dex.detail {
		m.paintDexIndex(c, g)
	} else if m.dex.tab == 0 && !m.dex.overviewFacts {
		m.paintDexOverview(c, g)
	} else if m.dex.tab == 0 {
		c.box(g.entryX, g.bodyY, g.entryW, g.bodyH, "FIELD NOTES", p.accent)
		rows := m.compactEntryFacts()
		offset := min(m.dex.scroll, max(0, len(rows)-m.dexBodyHeight()))
		for i := 0; i < m.dexBodyHeight() && i+offset < len(rows); i++ {
			c.putANSI(g.entryX+2, g.bodyY+1+i, rows[i+offset].text)
		}
	} else {
		m.paintDexTab(c, g)
	}
	for _, control := range m.dexControls() {
		copy := m
		c.button(control.x, control.y, control.w, control.id, control.label, copy)
	}
	hint := "Arrows Focus  Enter Use  Tab Next"
	if m.focus == 0 {
		hint = "↑↓ Choose  Enter Open  / Search"
	}
	if m.focus == 10 && m.dex.tab == 0 {
		hint = "↑↓ Scroll  ←→ Pan  Tab Focus"
		if m.dex.overviewFacts {
			hint = "↑↓ Scroll  Tab Focus  Esc Index"
		}
	}
	if m.focus == 15 {
		hint = "↑↓ Choose  Enter Inspect  Tab Focus"
	}
	if m.focus >= 2000 && m.focus < 3000 {
		hint = "↑↓ Family  Enter Open  Tab Focus"
	}
	c.put(2, g.h-2, ansi.Truncate(hint, g.w-4, "…"), m.hintStyle())
}

func (m *Model) reconcileLayout() {
	if m.width < 40 || m.height < 12 {
		return
	}
	popupFocus := m.focus
	if m.compactMessage != "" {
		m.focus = m.compactMessageReturnFocus
		defer func() {
			m.compactMessageReturnFocus = m.focus
			m.focus = popupFocus
			m.compactMessageScroll = min(m.compactMessageScroll, m.compactMessageMaxScroll())
		}()
	}
	if m.screen == dexScreen {
		settingsFocus := m.focus
		if m.settings {
			m.focus = m.settingsReturnFocus
		}
		if !m.dexWide() {
			if m.focus == 10 || m.focus == 15 || m.focus >= 2000 {
				m.dex.detail = true
			}
			if m.width < 64 && m.focus >= 31 && m.focus <= 33 {
				m.focus = 2
			}
		}
		m.dex.scroll = max(0, min(m.dex.scroll, m.dexMaxScroll()))
		m.dex.horizontal = max(0, min(m.dex.horizontal, m.dexMaxHorizontal()))
		m.ensureDexFocus()
		if m.settings {
			m.settingsReturnFocus = m.focus
			m.focus = settingsFocus
		}
	}
	if m.screen == activityScreen {
		m.activity.scroll = max(0, min(m.activity.scroll, m.activityMaxScroll()))
		m.activity.artPan = max(0, min(m.activity.artPan, m.activityMaxPan()))
		m.activity.artScroll = max(0, min(m.activity.artScroll, max(0, len(m.activity.art)-m.activityBodyHeight()+2)))
		for panel := 0; panel < 2; panel++ {
			m.activity.panelScroll[panel] = max(0, min(m.activity.panelScroll[panel], m.achievementMaxScroll(panel)))
		}
		controls := m.activityControls()
		if len(controls) > 0 {
			found := false
			id := m.focus
			if m.settings {
				id = m.settingsReturnFocus
			}
			for _, control := range controls {
				found = found || control.id == id
			}
			if !found {
				if m.settings {
					m.settingsReturnFocus = controls[0].id
				} else {
					m.focus = controls[0].id
				}
			}
		}
	}
}

func (m *Model) openFamilyArtwork() tea.Cmd {
	if len(m.dex.entry.Evolution) == 0 {
		return nil
	}
	id := m.focus
	if id < 2000 || id >= 3000 {
		id = m.dex.cardNode
	}
	valid := false
	for _, node := range m.dex.entry.Evolution {
		valid = valid || id == 2000+node.Number
	}
	if !valid {
		id = 2000 + orderedEvolution(m.dex.entry.Evolution)[0].Number
	}
	q := catalog.Selection{}
	if m.dex.snapshot.data != nil {
		for _, option := range m.dex.snapshot.data.Appearances(id - 2000) {
			if option.Collected {
				q = option.Selection
				break
			}
		}
	}
	m.dex.familyOrigin, m.dex.familyNode, m.dex.familySelection = m.dex.entry.Number, id, m.dex.selection
	m.dex.status, m.dex.generationFilter, m.dex.query = 0, 0, ""
	m.filterDex()
	for i, row := range m.dex.rows {
		if row.Number == id-2000 {
			m.dex.selected = i
			break
		}
	}
	m.dex.detail, m.dex.tab, m.focus = true, 0, 10
	return m.loadDexEntry(q)
}

func trimSpriteMargins(art string) string {
	if art == "" {
		return ""
	}
	rows := strings.Split(strings.TrimSuffix(art, "\n"), "\n")
	first, last := 0, len(rows)
	for first < last && strings.TrimSpace(ansi.Strip(rows[first])) == "" {
		first++
	}
	for last > first && strings.TrimSpace(ansi.Strip(rows[last-1])) == "" {
		last--
	}
	if first == last {
		return ""
	}
	left, right := int(^uint(0)>>1), 0
	for _, row := range rows[first:last] {
		plain := ansi.Strip(row)
		if strings.TrimSpace(plain) == "" {
			continue
		}
		left = min(left, ansi.StringWidth(plain)-ansi.StringWidth(strings.TrimLeft(plain, " ")))
		right = max(right, ansi.StringWidth(strings.TrimRight(plain, " ")))
	}
	out := []string{}
	for _, row := range rows[first:last] {
		out = append(out, ansi.Cut(row, left, right))
	}
	return strings.Join(out, "\n")
}

func (m Model) compactActivityBodyY() int {
	if m.section == 1 {
		return 4
	}
	return 3
}
func (m Model) compactActivityControls() []dexControl {
	if m.width < 40 || m.height < 12 {
		return []dexControl{{0, max(0, m.height-1), min(12, m.width), 14, "Quit"}}
	}
	g := m.dexGeometry()
	out := []dexControl{}
	add := func(x, y, w, id int, label string) { out = append(out, dexControl{x, y, w, id, label}) }
	if m.section == 1 {
		label := "Encounter"
		if m.activity.recording {
			label = "Saving…"
		}
		if m.activity.loading {
			label = "Reading…"
		}
		add(2, 2, 16, 10, label)
		label = "History"
		if m.activity.historyMode {
			label = "Show result"
		}
		add(18, 2, g.w-20, 30, label)
	} else if m.section == 2 {
		add(2, 2, g.w-4, 15, "Choose / create trainer")
	}
	artView := m.section == 1 && !m.activity.historyMode && len(m.activity.art) > 0 && !m.activity.resultDetails && m.activity.error == ""
	bottom := m.compactActivityBodyY() + m.activityBodyHeight() - 1
	if artView {
		vertical := len(m.activity.art) > m.activityBodyHeight()-2
		pan := m.activityMaxPan() > 0
		if vertical || pan {
			add(4, bottom, 7, 21, "Art")
		}
		if vertical {
			add(11, bottom, 5, 33, "↑")
			add(16, bottom, 5, 34, "↓")
		}
		if pan {
			add(21, bottom, 5, 18, "←")
			add(26, bottom, 5, 19, "→")
		}
	} else if m.activityMaxScroll() > 0 {
		if !(m.section == 1 && m.activity.historyMode && len(m.activity.data.history) > 0) {
			add(4, bottom, 12, 22, "Scroll")
		}
		add(g.w-14, bottom, 5, 16, "↑")
		add(g.w-9, bottom, 5, 17, "↓")
	}
	if m.section == 1 && m.activity.historyMode && len(m.activity.data.history) > 0 {
		add(2, g.h-4, 8, 20, "Open")
		add(11, g.h-4, 12, 31, "Previous")
		add(24, g.h-4, 8, 32, "Next")
	} else if m.section == 1 && len(m.activity.art) > 0 && m.activity.error == "" {
		label := "Details"
		if m.activity.resultDetails {
			label = "Art"
		}
		add(2, g.h-4, 16, 35, label)
	}
	return append(out, compactActions(2, g.h-3, g.w-4, []int{12, 13, 11, 14}, []string{"Back", "Theme", "Reload", "Quit"})...)
}
func (m Model) paintCompactActivity(c *canvas) {
	g, p, a := m.dexGeometry(), m.palette(), m.activity
	c.box(0, 0, g.w, g.h, "", p.accent)
	c.put(2, 1, ansi.Truncate("POKÉCRT / "+strings.ToUpper(sections[m.section]), g.w-4, "…"), p.accent)
	c.put(2, 1, "POKÉCRT", m.brandStyle())
	y, h := m.compactActivityBodyY(), m.activityBodyHeight()
	artView := m.section == 1 && !a.historyMode && len(a.art) > 0 && !a.resultDetails && a.error == ""
	title := map[int]string{1: "ENCOUNTER DETAILS", 2: "TRAINER CARD", 3: "ACHIEVEMENT JOURNAL"}[m.section]
	if artView {
		title = "DEVICE DISPLAY"
	}
	if m.section == 1 && a.historyMode {
		title = "RECENT DISCOVERIES"
	}
	if m.focus == 21 || m.focus == 22 || m.focus == 20 {
		title = "▶ " + title
	}
	c.box(2, y, g.w-4, h, title, p.accent)
	if m.section == 1 && a.result != nil && !a.historyMode {
		c.put(2, 3, ansi.Truncate(fmt.Sprintf("#%03d %s  +%d XP", a.result.Choice.Key().SpeciesID, strings.ToUpper(clean(a.result.Choice.Snapshot().SpeciesName)), a.result.XPAwarded), g.w-4, "…"), p.muted)
	}
	if artView {
		m.paintActivityArt(c, 3, y+1, g.w-6, h-2)
		if len(a.art) > h-2 || m.activityMaxPan() > 0 {
			for row := y + 1; row < y+h-1; row++ {
				c.hits = append(c.hits, hit{3, row, g.w - 6, 21})
			}
		}
	} else {
		rows := m.activityWrapped()
		if a.loading && a.data.profile.ID == 0 {
			rows = []string{"Reading trainer records…"}
		}
		offset := min(a.scroll, max(0, len(rows)-h+2))
		targets := m.activityHistoryTargets()
		for i := 0; i < h-2 && i+offset < len(rows); i++ {
			c.putANSI(4, y+1+i, rows[i+offset])
			if m.section == 1 && a.historyMode && !a.loading && a.error == "" {
				if i+offset < len(targets) && targets[i+offset] >= 0 {
					c.hits = append(c.hits, hit{4, y + 1 + i, g.w - 8, 1000 + targets[i+offset]})
				}
			} else if m.activityMaxScroll() > 0 {
				c.hits = append(c.hits, hit{4, y + 1 + i, g.w - 8, 22})
			}
		}
	}
	for _, control := range m.activityControls() {
		c.button(control.x, control.y, control.w, control.id, control.label, m)
	}
	hint := "Arrows Focus  Enter Use  Tab Next"
	if m.focus == 21 {
		hint = "↑↓ Scroll  ←→ Pan  Tab Focus"
	}
	if m.focus == 22 {
		hint = "↑↓ Scroll  Tab Focus  Esc Back"
	}
	if m.focus == 20 {
		hint = "↑↓ History  Enter Open  Tab Focus"
	}
	c.put(2, g.h-2, ansi.Truncate(hint, g.w-4, "…"), m.hintStyle())
}

func (m Model) compactExplanation() string {
	if m.settings {
		message := "Appearance changes preview immediately. Save as default remembers the selected mode. Back returns without changing the saved default. Terminal Native preserves the terminal background and transparency. Follow Terminal checks background replies while focused and uses terminal defaults when replies are unavailable."
		if m.noColor {
			message = "NO_COLOR is enabled. Focus and selection use visible characters. Color choices apply when color is enabled.\n\n" + message
		}
		return m.configStatus() + "\n\n" + message
	}
	message := "Choose a trainer, then Use trainer to activate it. Creating an additional trainer does not switch the current trainer automatically."
	if m.screen == createScreen {
		message = "Use 1–32 Unicode characters for the trainer name. The first trainer becomes active. Additional trainers require Use trainer to become active. Creating a trainer does not record an encounter."
	}
	if m.screen == dexSearchScreen {
		message = "Search revealed names or National numbers. An empty search restores the filtered list. Left/Right edits the input cursor; Down enters the onscreen keyboard. Cancel or Esc returns without applying the search."
	}
	if m.formError != "" {
		message = m.formError + "\n\n" + message
	}
	if m.notice != "" {
		message = m.notice + "\n\n" + message
	}
	return message
}
func (m *Model) openCompactMessage() tea.Cmd {
	m.compactMessage = m.compactExplanation()
	m.compactMessageScroll = 0
	m.compactMessageReturnFocus = m.focus
	m.focus = 92
	return nil
}
func (m Model) compactMessageRows() []string {
	return wrapActivityLines(strings.Split(m.compactMessage, "\n"), min(m.width, 88)-4)
}
func (m Model) compactMessageMaxScroll() int {
	return max(0, len(m.compactMessageRows())-max(1, min(m.height, 28)-6))
}
func (m *Model) closeCompactMessage() {
	m.compactMessage = ""
	m.focus = m.compactMessageReturnFocus
	m.reconcileLayout()
}
func (m *Model) activateCompactMessage(id int) tea.Cmd {
	switch id {
	case 92:
		m.closeCompactMessage()
	case 93:
		m.compactMessageScroll = max(0, m.compactMessageScroll-1)
	case 94:
		m.compactMessageScroll = min(m.compactMessageMaxScroll(), m.compactMessageScroll+1)
	}
	return nil
}
func (m *Model) handleCompactMessage(msg tea.Msg) (tea.Cmd, bool) {
	if m.compactMessage == "" {
		return nil, false
	}
	switch k := msg.(type) {
	case activateMsg:
		return m.activateCompactMessage(int(k)), true
	case tea.KeyPressMsg:
		switch k.String() {
		case "ctrl+c", "q":
			return tea.Quit, true
		case "esc":
			m.closeCompactMessage()
		case "enter", "space":
			return m.activateCompactMessage(m.focus), true
		case "up", "pgup":
			delta := 1
			if k.String() == "pgup" {
				delta = max(1, min(m.height, 28)-6)
			}
			m.compactMessageScroll = max(0, m.compactMessageScroll-delta)
		case "down", "pgdown":
			delta := 1
			if k.String() == "pgdown" {
				delta = max(1, min(m.height, 28)-6)
			}
			m.compactMessageScroll = min(m.compactMessageMaxScroll(), m.compactMessageScroll+delta)
		case "tab", "shift+tab", "left", "right":
			order := []int{92}
			if m.compactMessageMaxScroll() > 0 {
				order = []int{92, 93, 94}
			}
			delta := 1
			if k.String() == "shift+tab" || k.String() == "left" {
				delta = -1
			}
			for i, id := range order {
				if id == m.focus {
					m.focus = order[(i+delta+len(order))%len(order)]
					break
				}
			}
		}
		return nil, true
	case tea.MouseWheelMsg:
		delta := 1
		if k.Button == tea.MouseWheelUp || k.Button == tea.MouseWheelLeft {
			delta = -1
		}
		m.compactMessageScroll = max(0, min(m.compactMessageMaxScroll(), m.compactMessageScroll+delta))
		return nil, true
	}
	return nil, false
}
func (m Model) paintCompactMessage(c *canvas) {
	w, h := min(c.width, 88), min(c.height, 28)
	x, y := (c.width-w)/2, (c.height-h)/2
	c.pageBox(x, y, w, h, "HELP / STATUS", m)
	rows := m.compactMessageRows()
	offset := min(m.compactMessageScroll, m.compactMessageMaxScroll())
	for i := 0; i < h-6 && i+offset < len(rows); i++ {
		c.putANSI(x+2, y+2+i, rows[i+offset])
	}
	c.button(x+2, y+h-3, 10, 92, "Back", m)
	if m.compactMessageMaxScroll() > 0 {
		c.button(x+w-14, y+h-3, 5, 93, "↑", m)
		c.button(x+w-8, y+h-3, 5, 94, "↓", m)
	}
	c.put(x+2, y+h-2, ansi.Truncate("↑↓ Scroll  Enter Back  Esc Close", w-4, "…"), m.hintStyle())
}

func (m Model) paintCompactCreate(c *canvas) {
	p := m.palette()
	w, h := min(c.width, 88), min(c.height, 28)
	x, y := (c.width-w)/2, (c.height-h)/2
	title, subtitle := "NEW TRAINER", "Name (1–32 characters)"
	if m.screen == dexSearchScreen {
		title, subtitle = "POKÉDEX SEARCH", "Name or National number"
	}
	c.pageBox(x, y, w, h, title, m)
	c.put(x+2, y+1, ansi.Truncate(subtitle, w-4, "…"), p.muted)
	prefix, suffix := string(m.name[:m.cursor]), string(m.name[m.cursor:])
	fieldBoxW := min(w-4, 52)
	fieldW := fieldBoxW - 2
	offset := max(0, ansi.StringWidth(prefix)-fieldW+2)
	caret := " "
	if m.focus == 0 {
		caret = "█"
	}
	field := ansi.Cut(prefix, offset, ansi.StringWidth(prefix)) + caret + suffix
	c.put(x+3, y+2, ansi.Truncate(field, fieldW, "…"), m.controlStyle(m.focus == 0, false))
	c.put(x+2, y+3, strings.Repeat("─", fieldBoxW), p.muted)
	c.hits = append(c.hits, hit{x + 2, y + 2, fieldBoxW, 0})
	status := m.formError
	if status == "" {
		status = m.notice
	}
	c.put(x+2, y+4, ansi.Truncate(status, w-4, "…"), p.accent)
	keysX := x + (w-30)/2
	for i, r := range m.keyboardLetters() {
		c.button(keysX+(i%10)*3, y+5+i/10, 3, 100+i, string(r), m)
	}
	page := []string{"abc→ABC", "ABC→123", "123→abc"}[m.keyboardPage]
	c.button(x+2, y+8, 11, 3, page, m)
	c.button(x+13, y+8, 11, 4, "Backspace", m)
	c.button(x+24, y+8, 7, 5, "Space", m)
	c.button(x+31, y+8, 7, 90, "Help", m)
	if h >= 16 {
		message := "Creating a trainer does not record an encounter. Help explains activation and input."
		if m.screen == dexSearchScreen {
			message = "Search revealed names or National numbers. Help explains search and navigation."
		}
		c.wrap(x+2, y+10, w-4, max(0, h-14), message, p.muted)
	}
	label := "Create"
	if m.screen == dexSearchScreen {
		label = "Search"
	}
	if m.busy {
		label = "Saving"
	}
	for _, control := range compactActions(x+2, y+h-3, w-4, []int{1, 2, 6, 7}, []string{label, "Cancel", "Theme", "Quit"}) {
		c.button(control.x, control.y, control.w, control.id, control.label, m)
	}
	hint := "Arrows Focus  Enter Use  Tab Next"
	if m.focus == 0 {
		hint = "←→ Cursor  ↓ Keys  Enter " + label
	}
	if m.focus >= 100 {
		hint = "↑↓←→ Keys  Enter Type  Tab Next"
	}
	c.put(x+2, y+h-2, ansi.Truncate(hint, w-4, "…"), m.hintStyle())
}

func (m Model) paintCompactSettings(c *canvas) {
	w, h := min(c.width, 88), min(c.height, 28)
	x, y := (c.width-w)/2, (c.height-h)/2
	p := m.palette()
	c.pageBox(x, y, w, h, "APPEARANCE", m)
	c.button(x+w-10, y+1, 8, 6, "Quit", m)
	for i, label := range appearanceNames {
		if h >= 26 {
			c.framedControl(x+2, y+3+i*3, w-4, i, label, m, appearances[i] == m.appearance)
		} else {
			c.button(x+2, y+2+i, w-4, i, label, m)
		}
	}
	helpY := y + 6
	if h >= 26 {
		helpY = y + 15
	}
	c.put(x+2, helpY, ansi.Truncate("Live preview. Help shows full details.", w-4, "…"), p.muted)
	c.wrap(x+2, helpY+1, w-4, max(0, y+h-5-helpY-1), m.configStatus(), p.accent)
	c.button(x+2, y+h-4, 12, 90, "Help", m)
	label := "Save default"
	if m.savingConfig {
		label = "Saving…"
	}
	c.button(x+2, y+h-3, 18, 4, label, m)
	c.button(x+21, y+h-3, 10, 5, "Back", m)
	c.put(x+2, y+h-2, ansi.Truncate("↑↓ Mode  Enter Apply  Tab Focus", w-4, "…"), m.hintStyle())
}
