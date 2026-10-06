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
	if m.screen == dexScreen {
		return m.pokedexGeometry(false)
	}

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

func (m Model) compactDexControls() []dexControl { return m.pokedexControls() }

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
	brandW := g.w - 4
	if !m.dex.detail && len(m.dex.rows) > 0 {
		brandW = g.w - 20
	} else if m.dex.detail && m.dex.tab == 0 {
		brandW = g.w - 20
	} else if m.dex.detail && m.dex.tab == 2 {
		brandW = g.w - 16
	}
	if m.dex.detail && compactActionRows(g.w) && m.height < 16 {
		brandW = g.w - 4
	}
	c.put(2, 1, ansi.Truncate(brand, brandW, "…"), p.accent)
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
	if m.dex.detail && g.bodyH < 3 {
		c.put(2, 5, "Choose filters above.", p.muted)
	} else if !m.dex.detail {
		m.paintDexIndex(c, g)
	} else if m.dex.tab == 0 && !m.dex.overviewFacts {
		m.paintDexOverview(c, g)
	} else if m.dex.tab == 0 {
		c.box(g.entryX, g.bodyY, g.entryW, g.bodyH, "POKÉMON DETAILS", p.accent)
		rows := m.compactEntryFacts()
		offset := min(m.dex.scroll, max(0, len(rows)-m.dexBodyHeight()))
		for i := 0; i < m.dexBodyHeight() && i+offset < len(rows); i++ {
			c.putANSI(g.entryX+2, g.bodyY+1+i, rows[i+offset].text)
		}
	} else {
		m.paintDexTab(c, g)
	}
	m.paintPokedexControls(c)
	m.paintPokedexHints(c)
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
			if m.settings && m.compactMessageMaxScroll() == 0 && m.focus >= 93 && m.focus <= 95 {
				m.focus = 92
			}
		}()
	}
	if m.screen == dexScreen {
		settingsFocus := m.focus
		if m.settings {
			m.focus = m.settingsReturnFocus
		}
		if !m.dexWide() {
			if pokedexEntryFocus(m.focus) {
				m.dex.detail = true
			}
			if m.focus >= 40 && m.focus <= 42 {
				m.dex.tab = 0
				m.dex.overviewFacts = true
				m.dex.scroll = m.dex.factsScroll
				m.focus = 18
			}
			if m.width < 64 && m.focus >= 31 && m.focus <= 33 {
				m.focus = 2
			}
		}
		m.dex.scroll = max(0, min(m.dex.scroll, m.dexMaxScroll()))
		m.dex.horizontal = max(0, min(m.dex.horizontal, m.dexMaxHorizontal()))
		m.dex.factsScroll = max(0, min(m.dex.factsScroll, m.dexFactsMaxScroll()))
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
			m.activity.panelScroll[panel] = max(0, min(m.activity.panelScroll[panel], m.activityPanelMaxScroll(panel)))
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
	if m.section == 2 && m.height >= 24 && m.width >= 56 {
		return 5
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
		y := 2
		if g.h >= 24 && g.w >= 56 {
			y = 3
		}
		add(2, y, 29, 15, "Choose / create trainer")
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
			add(3, bottom, 10, 22, "Scroll")
		}
		upX := g.w - 14
		if m.section != 1 {
			upX--
		}
		add(upX, bottom, 5, 16, "↑")
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
	if m.section != 1 {
		y := g.h - 5
		if g.h >= 24 && g.w >= 56 {
			y = g.h - 6
		}
		if compactActionRows(g.w) && m.height >= 16 {
			y--
		}
		return append(out, pageFooterControls(g.w, m.height, y, []int{12, 13, 11, 14}, []string{"Back", "Appearance", "Refresh", "Quit"}, g.h >= 24 && g.w >= 56)...)

	}
	return append(out, compactActions(2, g.h-3, g.w-4, []int{12, 13, 11, 14}, []string{"Back", "Appearance", "Reload", "Quit"})...)
}
func (m Model) paintCompactActivity(c *canvas) {
	g, p, a := m.dexGeometry(), m.palette(), m.activity
	c.box(0, 0, g.w, g.h, "", p.accent)
	titleW := g.w - 4
	c.put(2, 1, ansi.Truncate("POKÉCRT / "+strings.ToUpper(sections[m.section]), titleW, "…"), p.accent)
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
	if m.section == 1 && (m.focus == 21 || m.focus == 22 || m.focus == 20) {
		title = "▶ " + title
	}
	c.box(2, y, g.w-4, h, title, m.achievementFrameStyle(p.accent))
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
		if m.section != 1 && m.height >= 24 && m.width >= 56 && (control.id == 15 || control.id == 12 || control.id == 13 || control.id == 11 || control.id == 14) {
			if m.section == 2 {
				c.trainerOutlinedButton(control.x, control.y-1, control.w, control.id, control.label, m)
			} else {
				c.outlinedButton(control.x, control.y-1, control.w, control.id, control.label, m)
			}
		} else {

			c.button(control.x, control.y, control.w, control.id, control.label, m)
		}
	}
	if m.section != 1 {
		m.paintAchievementHints(c, 2, g.h-3, g.w-4)
		return
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
	width := min(m.width, 88) - 4
	if m.settings {
		width -= 4
	}
	return wrapActivityLines(strings.Split(m.compactMessage, "\n"), width)
}
func (m Model) compactMessagePageSize() int {
	reserve := 6
	if m.settings {
		reserve = 9
	}
	return max(1, min(m.height, 28)-reserve)
}
func (m Model) compactMessageMaxScroll() int {
	return max(0, len(m.compactMessageRows())-m.compactMessagePageSize())
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
	case 95:
		if m.settings {
			m.focus = 95
		}
	case 93:
		if m.settings {
			m.focus = 93
		}
		m.compactMessageScroll = max(0, m.compactMessageScroll-1)
	case 94:
		if m.settings {
			m.focus = 94
		}
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
				delta = m.compactMessagePageSize()
			}
			m.compactMessageScroll = max(0, m.compactMessageScroll-delta)
		case "down", "pgdown":
			delta := 1
			if k.String() == "pgdown" {
				delta = m.compactMessagePageSize()
			}
			m.compactMessageScroll = min(m.compactMessageMaxScroll(), m.compactMessageScroll+delta)
		case "tab", "shift+tab", "left", "right":
			order := []int{92}
			if m.compactMessageMaxScroll() > 0 {
				order = []int{92, 93, 94}
				if m.settings {
					order = []int{95, 93, 94, 92}
				}
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
	if m.settings {
		m.paintAppearanceHelp(c)
		return
	}
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
	subtitleW := w - 4

	c.put(x+2, y+1, ansi.Truncate(subtitle, subtitleW, "…"), p.muted)
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
	if !(compactActionRows(w) && h < 16) {
		c.put(x+2, y+3, strings.Repeat("─", fieldBoxW), p.muted)
	}
	c.hits = append(c.hits, hit{x + 2, y + 2, fieldBoxW, 0})
	status := m.formError
	if status == "" {
		status = m.notice
	}
	keysY, editY, statusY := y+5, y+8, y+4
	if compactActionRows(w) && h < 16 {
		keysY = y + 3
		editY = y + 6
		statusY = y + 1
	} else if h == 12 {
		keysY--
		editY--
		statusY = y + 3
	}
	if status != "" || statusY == y+4 {
		c.put(x+2, statusY, ansi.Truncate(status, w-4, "…"), p.accent)
	}
	keysX := x + (w-30)/2
	for i, r := range m.keyboardLetters() {
		c.button(keysX+(i%10)*3, keysY+i/10, 3, 100+i, string(r), m)
	}
	page := []string{"abc→ABC", "ABC→123", "123→abc"}[m.keyboardPage]
	c.button(x+2, editY, 11, 3, page, m)
	if m.screen == createScreen || m.screen == dexSearchScreen {
		c.button(x+13, editY, 13, 4, "Backspace", m)
		c.button(x+26, editY, 9, 5, "Space", m)
	} else {
		c.button(x+13, editY, 11, 4, "Backspace", m)
		c.button(x+24, editY, 7, 5, "Space", m)
	}
	if h >= 16 {
		message := "The first trainer becomes active. Additional trainers require Use trainer. No encounter is recorded."
		if m.screen == dexSearchScreen {
			message = "Search revealed names or National numbers. An empty search clears the query."
		}
		messageH := max(0, h-14)
		if compactActionRows(w) {
			messageH = max(0, h-17)
		}
		c.wrap(x+2, y+10, w-4, messageH, message, p.muted)
	}
	label := "Create"
	if m.screen == dexSearchScreen {
		label = "Search"
	}
	if m.busy {
		label = "Saving"
	}
	if m.screen == createScreen || m.screen == dexSearchScreen {
		actionY := y + h - 4
		if h >= 16 {
			actionY = y + h - 5
		}
		if h >= 24 {
			actionY = y + h - 6
		}
		if compactActionRows(w) {
			actionY--
		}
		buttons := pageFooterControls(w, h, actionY-y, []int{1, 2, 6, 7}, []string{label, "Cancel", "Appearance", "Quit"}, h >= 24 && w >= 56)
		for _, control := range buttons {
			if h >= 24 && w >= 56 {
				c.framedControl(x+control.x, y+control.y-1, control.w, control.id, control.label, m, false)
			} else {
				c.button(x+control.x, y+control.y, control.w, control.id, control.label, m)
			}
		}

		first := "[↑↓←→] Move  [Tab] Focus"
		second := "[Enter] Select  [Esc] Back"
		if m.focus == 0 {
			first = "↓ Keys  [←→] Cursor  [Tab] Focus"
			second = "[Enter] Create  [Esc] Back"
			if m.screen == dexSearchScreen {
				second = "[Enter] Search  [Esc] Back"
			}
		}
		if m.focus >= 100 {
			second = "[Enter] Type  [Esc] Back"
		}
		if h >= 16 {
			m.paintSetupHints(c, x, y, w, h, first, second)
		} else {
			hint := "↓ Keys  [Tab] Focus  [Enter] Create"
			if m.screen == dexSearchScreen {
				hint = "↓ Keys  [Tab] Focus  [Enter] Search"
			}
			if m.focus >= 100 {
				hint = "[↑↓←→] [Tab] Focus [Enter] Type"
			} else if m.focus != 0 {
				hint = "[↑↓←→] [Tab] Focus [Enter] Select"
			}
			c.put(x, y+h-3, "├"+strings.Repeat("─", w-2)+"┤", m.footerDividerStyle())
			c.put(x+2, y+h-2, ansi.Truncate(hint, w-4, "…"), m.hintStyle())
		}
	} else {
		for _, control := range compactActions(x+2, y+h-3, w-4, []int{1, 2, 6, 7}, []string{label, "Cancel", "Appearance", "Quit"}) {
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
}
