package tui

import (
	"github.com/charmbracelet/x/ansi"
	"strings"
)

type appearanceLayout struct {
	x, y, w, h, modeY, footerY int
	outlined                   bool
}

func (m Model) appearanceGeometry() appearanceLayout {
	w, h := min(m.width, 88), min(m.height, 28)
	g := appearanceLayout{x: (m.width - w) / 2, y: (m.height - h) / 2, w: w, h: h, modeY: 2, footerY: h - 5}
	g.outlined = w >= 56 && h >= 24
	if g.outlined {
		g.modeY = 5
		g.footerY = h - 6
	}
	return g
}
func (m Model) appearanceControls() []dexControl {
	g := m.appearanceGeometry()
	cs := []dexControl{}
	quitY, quitW := g.y+1, 8
	if g.outlined {
		quitY++
		quitW = 12
	}
	cs = append(cs, dexControl{g.x + g.w - quitW - 2, quitY, quitW, 6, "Quit"})
	for i, label := range appearanceNames {
		row := g.modeY + i
		if g.outlined {
			row = g.modeY + i*3
		}
		cs = append(cs, dexControl{g.x + 2, g.y + row, 23, i, label})
	}
	x := g.x + 2
	save := "Save default"
	if m.savingConfig {
		save = "Saving…"
	}
	for i, label := range []string{"Back", save, "Help"} {
		width := []int{8, 16, 8}[i]
		if g.outlined {
			width += 2
		}
		cs = append(cs, dexControl{x, g.y + g.footerY, width, []int{5, 4, 90}[i], label})
		x += width + 1
	}
	return cs
}
func appearanceLabel(a Appearance) string {
	for i, mode := range appearances {
		if a == mode {
			return appearanceNames[i]
		}
	}
	return string(a)
}
func (m Model) appearanceSettingsStatus() string {
	if m.savingConfig {
		return "Saving default…"
	}
	if m.configError != "" {
		return "Settings error. See Help for details."
	}
	if m.savedAppearance == m.appearance {
		return "Default: " + appearanceLabel(m.appearance)
	}
	return "Preview: " + appearanceLabel(m.appearance) + "  not saved"
}
func (m Model) appearanceDescription() string {
	a := m.appearance
	if m.focus >= 0 && m.focus < 4 {
		a = appearances[m.focus]
	}
	text := map[Appearance]string{
		Dark:           "Uses the dark color palette.",
		Light:          "Uses the light color palette.",
		FollowTerminal: "Matches terminal background replies. Uses terminal defaults when replies are unavailable.",
		TerminalNative: "Keeps your terminal background, colors and transparency.",
	}[a]
	if m.noColor {
		text = "NO_COLOR keeps output uncolored. Focus and selection remain visible. " + text
	}
	return text
}
func (m Model) paintSettings(c *canvas) {
	g, p := m.appearanceGeometry(), m.palette()
	c.pageBox(g.x, g.y, g.w, g.h, "APPEARANCE", m)
	c.put(g.x+2, g.y+1, "Changes apply immediately", p.muted)
	statusY := g.y + g.footerY - 1
	if g.outlined {
		statusY--
		selected := m.appearance
		if m.focus >= 0 && m.focus < 4 {
			selected = appearances[m.focus]
		}
		c.putANSI(g.x+28, g.y+4, m.informationHeading(strings.ToUpper(appearanceLabel(selected)), g.w-30))
		c.wrap(g.x+28, g.y+6, g.w-30, 8, m.appearanceDescription(), p.foreground)
	} else {
		c.wrap(g.x+2, g.y+6, g.w-4, max(0, statusY-(g.y+6)), m.appearanceDescription(), p.foreground)
	}
	c.put(g.x+2, statusY, ansi.Truncate(m.appearanceSettingsStatus(), g.w-4, "…"), p.accent)
	for _, control := range m.appearanceControls() {
		selected := control.id >= 0 && control.id < 4 && appearances[control.id] == m.appearance
		if g.outlined {
			c.framedControl(control.x, control.y-1, control.w, control.id, control.label, m, selected)
		} else {
			c.button(control.x, control.y, control.w, control.id, control.label, m)
		}
		if selected {
			offset := 0
			if m.focus == control.id {
				offset = 1
			}
			c.put(control.x+1+offset, control.y, "●", p.gold)
		}
	}
	action := "Apply"
	switch m.focus {
	case 4:
		action = "Save"
	case 5:
		action = "Back"
	case 6:
		action = "Quit"
	case 90:
		action = "Help"
	}
	m.paintSetupHints(c, g.x, g.y, g.w, g.h, "[↑↓←→] Move  [Tab] Focus", "[Enter] "+action+"  [Esc] Back  [Q] Quit")
}

func (m Model) paintAppearanceHelp(c *canvas) {
	g, p := m.appearanceGeometry(), m.palette()
	c.pageBox(g.x, g.y, g.w, g.h, "APPEARANCE / HELP", m)
	bodyH := g.h - 7
	c.box(g.x+2, g.y+2, g.w-4, bodyH, "DETAILS", p.accent)
	rows := m.compactMessageRows()
	offset := min(m.compactMessageScroll, m.compactMessageMaxScroll())
	for i := 0; i < bodyH-2 && i+offset < len(rows); i++ {
		c.putANSI(g.x+4, g.y+3+i, rows[i+offset])
	}
	if m.compactMessageMaxScroll() > 0 {
		bottom := g.y + 2 + bodyH - 1
		c.button(g.x+3, bottom, 10, 95, "Scroll", m)
		c.button(g.x+g.w-14, bottom, 5, 93, "↑", m)
		c.button(g.x+g.w-8, bottom, 5, 94, "↓", m)
	}
	c.button(g.x+2, g.y+g.h-5, 8, 92, "Back", m)
	first := "[↑↓←→] Move  [Tab] Focus"
	if m.compactMessageMaxScroll() > 0 {
		first = "[↑↓] Scroll [←→] Move [Tab] Focus"
	}
	action := "Back"
	if m.focus == 93 || m.focus == 94 {
		action = "Scroll"
	}
	if m.focus == 95 {
		action = "Select"
	}
	m.paintSetupHints(c, g.x, g.y, g.w, g.h, first, "[Enter] "+action+"  [Esc] Back  [Q] Quit")
}
