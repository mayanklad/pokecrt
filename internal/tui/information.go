package tui

import (
	"fmt"
	"github.com/mayanklad/pokecrt/internal/trainer"
	"sort"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

func styled(text, style string) string {
	if style == "" {
		return text
	}
	return style + text + "\x1b[0m"
}
func (m Model) informationHeading(title string, width int) string {
	title = ansi.Truncate(title, max(1, width-3), "…")
	return styled(title, m.palette().gold) + styled("  "+strings.Repeat("─", max(0, width-ansi.StringWidth(title)-2)), m.hintStyle())
}
func (m Model) informationFact(label, value string, width int) string {
	if width < 30 || !m.dexWide() {
		return styled(label+": ", m.hintStyle()) + value
	}
	column := max(ansi.StringWidth(label)+2, min(18, max(10, width/3)))
	return styled(label+strings.Repeat(" ", column-ansi.StringWidth(label)), m.hintStyle()) + value
}
func (m Model) informationBar(label string, current, total int64, width int) string {
	if total <= 0 {
		return m.informationFact(label, "Unavailable", width)
	}
	barW := max(4, min(28, width-24))
	fill := int(float64(max(0, min(current, total))) / float64(total) * float64(barW))
	return styled(fmt.Sprintf("%-9s", label), m.hintStyle()) + styled(strings.Repeat("━", fill), m.palette().muted) + styled(strings.Repeat("─", barW-fill), m.hintStyle()) + fmt.Sprintf("  %d / %d", current, total)
}
func (m Model) typeBadge(kind string) string {
	colours := map[string][3]int{"grass": {95, 185, 115}, "poison": {190, 120, 205}, "fire": {230, 125, 75}, "water": {90, 165, 225}, "electric": {210, 175, 50}, "ice": {70, 180, 190}, "fighting": {205, 100, 90}, "ground": {185, 150, 90}, "flying": {140, 160, 215}, "psychic": {220, 115, 155}, "bug": {150, 175, 60}, "rock": {165, 150, 90}, "ghost": {145, 120, 190}, "dragon": {130, 125, 220}, "dark": {150, 135, 130}, "steel": {135, 160, 170}, "fairy": {205, 135, 190}, "normal": {155, 165, 145}}
	colour := m.palette().muted
	if rgb, ok := colours[kind]; ok && !m.noColor {
		colour = fg(rgb[0], rgb[1], rgb[2])
		if m.lightPalette() {
			colour = fg(rgb[0]*2/3, rgb[1]*2/3, rgb[2]*2/3)
		}
	}
	return styled("[ "+strings.ToUpper(kind)+" ]", colour)
}

// evolutionLines lays out whole clickable cards, with explicit child links so
// branches never imply an evolution merely because two cards are adjacent.
func (m Model) evolutionLines() []dexLine {
	e := m.dex.entry
	nodes := orderedEvolution(e.Evolution)
	w := m.dexDetailWidth()
	minCardWidth := 44
	for _, art := range m.dex.family {
		if !m.dexWide() {
			art = trimSpriteMargins(art)
		}
		for _, row := range strings.Split(art, "\n") {
			minCardWidth = max(minCardWidth, ansi.StringWidth(row)+4)
		}
	}
	columns := max(1, min(3, (w+2)/(minCardWidth+2)))
	cardW := (w - (columns-1)*2) / columns
	relationshipHeight := 1
	for _, node := range nodes {
		relationshipHeight = max(relationshipHeight, len(strings.Split(ansi.Wrap(evolutionCaption(node), max(1, cardW-4), ""), "\n")))
	}
	artHeight := 12
	for _, art := range m.dex.family {
		if !m.dexWide() {
			art = trimSpriteMargins(art)
		}
		artHeight = max(artHeight, len(strings.Split(strings.TrimSuffix(art, "\n"), "\n")))
	}
	out := []dexLine{{text: m.informationHeading("EVOLUTION FAMILY", w), art: true}, {text: "Collected appearances are shown. Undiscovered identities stay hidden."}, {}}
	for first := 0; first < len(nodes); first += columns {
		count := min(columns, len(nodes)-first)
		cards := make([][]string, count)
		height := 0
		for j := 0; j < count; j++ {
			node := nodes[first+j]
			border := m.hintStyle()
			marker := ""
			if m.focus == 18 && m.dex.tab == 2 && m.dex.cardNode == 2000+node.Number {
				border = m.palette().gold
				marker = "● "
			}
			if m.focus == 2000+node.Number {
				border = m.palette().muted
				marker = "▶ "
			}
			label := fmt.Sprintf("%s#%03d %s", marker, node.Number, clean(node.Name))
			inner := cardW - 2
			contentW := inner - 2
			card := []string{styled("╭"+strings.Repeat("─", inner)+"╮", border), styled("│ ", border) + styled(ansi.Truncate(label, contentW, "…"), m.palette().gold)}
			familyArt := m.dex.family[node.Number]
			if !m.dexWide() {
				familyArt = trimSpriteMargins(familyArt)
			}
			art := strings.Split(strings.TrimSuffix(familyArt, "\n"), "\n")
			if m.dex.family[node.Number] == "" {
				art = []string{"", "?", "", "Appearance not collected", ""}
			}
			placeholder := m.dex.family[node.Number] == ""
			artW := 0
			for _, row := range art {
				artW = max(artW, ansi.StringWidth(row))
			}
			pad := max(0, (artHeight-len(art))/2)
			for row := 0; row < artHeight; row++ {
				content := ""
				if index := row - pad; index >= 0 && index < len(art) {
					rowWidth := artW
					if placeholder {
						rowWidth = ansi.StringWidth(art[index])
					}
					content = strings.Repeat(" ", max(0, (contentW-rowWidth)/2)) + ansi.Cut(art[index], 0, contentW)
				}
				card = append(card, styled("│ ", border)+content)
			}
			if !m.dexWide() && artW > contentW {
				card = append(card, styled("│ Full art: Enter or button", m.hintStyle()))
			}
			card = append(card, styled("│", border))
			relationshipRows := strings.Split(ansi.Wrap(evolutionCaption(node), contentW, ""), "\n")
			for k := 0; k < relationshipHeight; k++ {
				caption := ""
				if k < len(relationshipRows) {
					caption = relationshipRows[k]
				}
				card = append(card, styled("│ ", border)+styled(caption, m.palette().muted))
			}
			for k, row := range card {
				card[k] = row + strings.Repeat(" ", max(0, cardW-1-ansi.StringWidth(row))) + styled("│", border)
			}
			// The top border is already complete.
			card[0] = styled("╭"+strings.Repeat("─", inner)+"╮", border)
			card = append(card, styled("╰"+strings.Repeat("─", inner)+"╯", border))
			cards[j] = card
			height = max(height, len(card))
		}
		for row := 0; row < height; row++ {
			line := dexLine{art: true}
			for j, card := range cards {
				if j > 0 {
					connector := "  "
					if row == 2+artHeight/2 {
						for _, child := range nodes[first+j-1].Children {
							if child == nodes[first+j].Number {
								connector = styled("→ ", m.palette().muted)
							}
						}
					}
					line.text += connector
				}
				if row < len(card) {
					line.text += card[row]
					line.nodes = append(line.nodes, dexControl{x: j * (cardW + 2), w: cardW, id: 2000 + nodes[first+j].Number})
				} else {
					line.text += strings.Repeat(" ", cardW)
				}
			}
			out = append(out, line)
		}
		out = append(out, dexLine{})
	}
	return out
}

func (m Model) recordLines() []dexLine {
	e := m.dex.entry
	w := m.dexDetailWidth()
	out := []dexLine{}
	add := func(text string) { out = append(out, dexLine{text: text, art: true}) }
	fact := func(label, value string) { add(m.informationFact(label, value, w)) }
	heading := func(title string) { add(m.informationHeading(title, w)) }
	heading("DISCOVERY RECORD")
	fact("Encounters", fmt.Sprint(e.Discovery.Count))
	fact("First discovered", dexDate(e.Discovery.FirstMS))
	fact("Last encountered", dexDate(e.Discovery.LastMS))
	add("")
	heading("SPECIES PROFILE")
	fact("Generation", fmt.Sprint(e.Generation))
	fact("Evolution stage", fmt.Sprint(e.Stage))
	fact("Colour", strings.ToUpper(e.Color))
	flags := []string{}
	if e.Baby {
		flags = append(flags, "Baby")
	}
	if e.Legendary {
		flags = append(flags, "Legendary")
	}
	if e.Mythical {
		flags = append(flags, "Mythical")
	}
	if len(flags) > 0 {
		fact("Classification", strings.Join(flags, " / "))
	}
	if e.SelectedName != "" {
		add("")
		heading("SELECTED APPEARANCE")
		add(clean(e.SelectedName))
		fact("Encounters", fmt.Sprint(e.Selected.Count))
		fact("First collected", dexDate(e.Selected.FirstMS))
		fact("Last encountered", dexDate(e.Selected.LastMS))
	}
	add("")
	heading("OBSERVED FORMS")
	for _, f := range e.Forms {
		add(styled(clean(f.Name), m.palette().gold))
		fact("Encounters", fmt.Sprint(f.Count))
		badges := []string{}
		for _, kind := range f.Types {
			badges = append(badges, m.typeBadge(kind))
		}
		add(strings.Join(badges, " "))
		for _, gender := range f.Genders {
			regular, shiny := "Locked", "Locked"
			if gender.Regular {
				regular = "Collected"
			}
			if gender.Shiny {
				shiny = "Collected"
			}
			fact(gender.Name, "Regular: "+regular+"   Shiny: "+shiny)
		}
		fact("Unknown genders", fmt.Sprint(f.UnknownGenders))
		add("")
	}
	fact("Unknown forms", fmt.Sprint(e.UnknownForms))
	add("")
	heading("YOUR COLLECTION")
	s := m.dex.snapshot.summary
	add(m.informationBar("Species", s.Completion.Species, s.Completion.SpeciesTotal, w))
	add(m.informationBar("Variants", s.Completion.Variants, s.Completion.VariantsTotal, w))
	fact("Encounters", fmt.Sprint(s.Encounters))
	fact("Shiny collections", fmt.Sprint(s.ShinyCollections))
	add("")
	heading("GENERATION PROGRESS")
	for _, g := range s.Generations {
		add(m.informationBar(fmt.Sprintf("Gen %02d", g.Generation), g.Seen, g.Total, w))
	}
	return wrapDexInformation(out, w)
}
func wrapDexInformation(lines []dexLine, width int) []dexLine {
	out := []dexLine{}
	for _, line := range lines {
		for _, row := range strings.Split(ansi.Wrap(line.text, max(1, width), ""), "\n") {
			out = append(out, dexLine{text: row, art: line.art, target: line.target})
		}
	}
	return out
}

func (m Model) entryFactRows(width int) []string {
	lines := []string{}
	e := m.dex.entry
	if !e.Seen {
		lines = []string{m.informationHeading("UNDISCOVERED", width), "An encounter reveals this entry.", "Identity and artwork remain hidden."}
	} else {
		lines = append(lines, m.informationHeading("APPEARANCE", width), strings.ReplaceAll(clean(e.SelectedName), " · ", " / "))
		badges := []string{}
		for _, kind := range e.SelectedTypes {
			badges = append(badges, m.typeBadge(kind))
		}
		lines = append(lines, strings.Join(badges, " "), "", m.informationHeading("SPECIES PROFILE", width), m.informationFact("Generation", fmt.Sprint(e.Generation), width), m.informationFact("Evolution stage", fmt.Sprint(e.Stage), width), m.informationFact("Colour", strings.ToUpper(e.Color), width), "", m.informationHeading("YOUR DISCOVERY", width), m.informationFact("Encounters", fmt.Sprint(e.Discovery.Count), width), m.informationFact("First discovered", dexDate(e.Discovery.FirstMS), width), m.informationFact("Last encountered", dexDate(e.Discovery.LastMS), width))
		if e.Notice != "" {
			lines = append(lines, "", e.Notice)
		}
		if m.dex.entryError != "" {
			lines = append(lines, "", m.dex.entryError)
		}
	}
	return wrapActivityLines(lines, width)
}
func (m Model) dexFactsRect() (x, y, w, h int) {
	g := m.dexGeometry()
	ax, ay, aw, ah := m.dexArtRect()
	x = ax + aw + 2
	return x, ay, g.entryX + g.entryW - x - 2, ah
}
func (m Model) dexFactsMaxScroll() int {
	if !m.dexWide() || m.dex.tab != 0 {
		return 0
	}
	_, _, w, h := m.dexFactsRect()
	return max(0, len(m.entryFactRows(w-4))-max(1, h-2))
}
func (m Model) paintEntryFacts(c *canvas, x, y, w, h int) {
	if w < 12 || h < 3 {
		return
	}
	c.box(x, y, w, h, "POKÉMON DETAILS", m.palette().accent)
	rows := m.entryFactRows(w - 4)
	offset := 0
	if m.dexWide() {
		offset = min(m.dex.factsScroll, m.dexFactsMaxScroll())
	}
	for i := 0; i < h-2 && i+offset < len(rows); i++ {
		c.putANSI(x+2, y+1+i, rows[i+offset])
		if m.dexFactsMaxScroll() > 0 {
			c.hits = append(c.hits, hit{x + 1, y + 1 + i, w - 2, 40})
		}
	}
}

func (m Model) trainerInformationColumns() (left, right []string) {
	d := m.activity.data
	s := d.stats
	p := s.Progress
	w := m.dexGeometry().w - 8
	if m.dexGeometry().wide {
		w = m.dexGeometry().w/2 - 7
	}
	fact := func(label, value string) string {
		if !m.dexWide() {
			return m.informationFact(label, value, w)
		}
		column := max(19, ansi.StringWidth(label)+2)
		return styled(label+strings.Repeat(" ", column-ansi.StringWidth(label)), m.hintStyle()) + value
	}
	left = []string{m.informationHeading(strings.ToUpper(clean(d.profile.Name)), w), fact("Trainer since", dexDate(d.profile.CreatedAtMS)), "", m.informationHeading("TRAINER LEVEL", w), fact("Level", fmt.Sprint(p.Level)), fact("Total XP", fmt.Sprint(p.Total)), m.informationBar("XP", p.InLevel, 1000, w), fact("Next level", fmt.Sprintf("%d XP remaining", p.ToNext)), "", m.informationHeading("JOURNEY TOTALS", w), fact("Encounters", fmt.Sprint(s.Encounters)), fact("Species", fmt.Sprint(s.Species)), fact("Appearances", fmt.Sprint(s.Variants)), fact("Shiny collections", fmt.Sprint(s.ShinyCollections)), fact("Shiny encounters", fmt.Sprint(s.ShinyEncounters)), "", m.informationHeading("COLLECTION", w), m.informationBar("Species", s.Completion.Species, s.Completion.SpeciesTotal, w), m.informationBar("Variants", s.Completion.Variants, s.Completion.VariantsTotal, w)}
	if s.FirstEncounterMS != nil {
		left = append(left, "", fact("First encounter", dexDate(*s.FirstEncounterMS)))
	}
	if s.LastEncounterMS != nil {
		left = append(left, fact("Last encounter", dexDate(*s.LastEncounterMS)))
	}
	right = []string{m.informationHeading("GENERATION PROGRESS", w), "Eligible discoveries in the current catalog.", ""}
	for _, g := range s.Generations {
		right = append(right, m.informationBar(fmt.Sprintf("Gen %02d", g.Generation), g.Eligible, g.EligibleTotal, w), fact("All discovered", fmt.Sprint(g.Discovered)), "")
	}
	if s.UnclassifiedSpecies > 0 {
		right = append(right, fact("Retained records", fmt.Sprintf("%d outside current catalog", s.UnclassifiedSpecies)))
	}
	return
}
func (m Model) achievementInformationColumns() (left, right []string) {
	d := m.activity.data
	w := m.dexGeometry().w - 8
	if m.dexGeometry().wide {
		w = m.dexGeometry().w/2 - 7
	}
	left = []string{m.informationHeading(fmt.Sprintf("%d BADGES EARNED", len(d.achievements.Unlocked)), w), ""}
	for _, g := range d.achievements.Unlocked {
		left = append(left, styled("● "+clean(g.Name), m.palette().gold), clean(g.Description), m.informationFact("Earned", dexDate(g.EarnedAtMS), w), "")
	}
	if len(d.achievements.Unlocked) == 0 {
		left = append(left, "Your first badge is waiting.", "Choose Encounter to begin your journey.")
	}
	right = []string{m.informationHeading(fmt.Sprintf("%d NEXT GOALS", len(d.achievements.Locked)), w), ""}
	for _, g := range d.achievements.Locked {
		right = append(right, styled("◇ "+clean(g.Name), m.palette().gold), clean(g.Description))
		if g.HasTarget {
			right = append(right, m.informationBar("Progress", g.Current, g.Target, w))
		} else {
			right = append(right, styled("Not yet earned", m.hintStyle()))
		}
		right = append(right, "")
	}
	return
}
func (m Model) encounterInformation() []string {
	w := m.dexGeometry().w - 8
	if m.dexGeometry().wide && len(m.activity.art) > 0 && m.activity.error == "" {
		w = m.dexGeometry().w/2 - 6
	}
	if m.activity.result == nil {
		return []string{m.informationHeading("READY TO EXPLORE", w), "", "Choose Encounter to discover a Pokémon.", "Your discoveries will be saved to this trainer."}
	}
	r := m.activity.result
	s := r.Choice.Snapshot()
	k := r.Choice.Key()
	fact := func(label, value string) string { return m.informationFact(label, value, w) }
	lines := []string{m.informationHeading(fmt.Sprintf("#%03d %s", k.SpeciesID, strings.ToUpper(clean(s.SpeciesName))), w), fact("Form", clean(s.FormName)), fact("Palette", strings.Title(k.Palette)), "", m.informationHeading("ENCOUNTER REWARDS", w), fact("XP gained", fmt.Sprintf("+%d XP", r.XPAwarded)), fact("Trainer level", fmt.Sprint(r.After.Level))}
	if r.FirstSpecies {
		lines = append(lines, styled("✦ NEW SPECIES DISCOVERED", m.palette().gold))
	}
	if r.FirstVariant {
		lines = append(lines, styled("✦ NEW APPEARANCE COLLECTED", m.palette().gold))
	}
	lines = append(lines, "", m.informationHeading("SAVED DISCOVERY", w), fact("Recorded", dexDate(r.EncounteredAtMS)), "Saved to your trainer.")
	for _, u := range r.NewUnlocks() {
		lines = append(lines, "", styled("● "+clean(u.Name), m.palette().gold))
	}
	return lines
}

// Family order follows parent links, including babies whose National Dex
// number is greater than that of their descendants.
func orderedEvolution(nodes []trainer.DexNode) []trainer.DexNode {
	out := append([]trainer.DexNode(nil), nodes...)
	depth := map[int]int{}
	for pass := 0; pass < len(nodes); pass++ {
		changed := false
		for _, node := range nodes {
			for _, child := range node.Children {
				if depth[child] < depth[node.Number]+1 {
					depth[child] = depth[node.Number] + 1
					changed = true
				}
			}
		}
		if !changed {
			break
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return depth[out[i].Number] < depth[out[j].Number] })
	return out
}

func evolutionCaption(node trainer.DexNode) string {
	if len(node.Children) == 0 {
		return "Final family stage"
	}
	children := make([]string, 0, len(node.Children))
	for _, id := range node.Children {
		children = append(children, fmt.Sprintf("#%03d", id))
	}
	return "→ " + strings.Join(children, ", ")
}

func (m Model) variantLines() []dexLine {
	w := m.dexDetailWidth()
	out := []dexLine{{text: m.informationHeading("APPEARANCE COLLECTION", w), art: true}, {text: "Choose an appearance. Collected artwork opens in Overview."}, {}}
	if len(m.dex.options) == 0 {
		return append(out, dexLine{text: "No observed appearances."})
	}
	for i, option := range m.dex.options {
		border := m.hintStyle()
		marker := "  "
		selected := option.Selection.Form == m.dex.selection.Form && option.Selection.Gender == m.dex.selection.Gender && option.Selection.Shiny == m.dex.selection.Shiny
		if key := m.dex.entry.ArtworkKey; key != nil {
			form, gender := option.Selection.Form, option.Selection.Gender
			if form == "" {
				form = "standard"
			}
			if gender == "" {
				gender = "default"
			}
			selected = form == key.FormID && gender == key.Gender && option.Selection.Shiny == (key.Palette == "shiny")
		}
		if selected {
			marker = "● "
			border = m.palette().gold
		}
		if m.focus == 18 && m.dex.tab == 1 && i == m.dex.optionIndex {
			marker = "● "
			border = m.palette().muted
		}
		if m.focus == 15 && i == m.dex.optionIndex {
			marker = "▶ "
			if selected {
				marker = "▶● "
			}
			border = m.palette().muted
		}
		status := "LOCKED"
		if option.Collected {
			status = "COLLECTED"
		}
		title := marker + strings.ReplaceAll(clean(option.Label), " · ", " / ")
		rows := strings.Split(ansi.Wrap(title, max(1, w-4), ""), "\n")
		lines := []string{styled("╭"+strings.Repeat("─", w-2)+"╮", border)}
		for _, row := range rows {
			lines = append(lines, styled("│ ", border)+styled(row, m.palette().foreground)+strings.Repeat(" ", max(0, w-3-ansi.StringWidth(row)))+styled("│", border))
		}
		detail := status + "   Enter to inspect"
		if selected {
			detail = status + "   Current appearance"
		}
		detail = ansi.Truncate(detail, w-4, "…")
		lines = append(lines, styled("│ ", border)+styled(detail, border)+strings.Repeat(" ", max(0, w-3-ansi.StringWidth(detail)))+styled("│", border), styled("╰"+strings.Repeat("─", w-2)+"╯", border))
		for _, row := range lines {
			out = append(out, dexLine{text: row, art: true, nodes: []dexControl{{w: w, id: 3000 + i}}})
		}
		out = append(out, dexLine{})
	}
	out = append(out, dexLine{text: fmt.Sprintf("Undiscovered forms: %d", m.dex.entry.UnknownForms)})
	return out
}
