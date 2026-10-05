package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

func (m Model) paintSetup(c *canvas) {
	if m.screen == createScreen {
		m.paintCreate(c)
	} else {
		m.paintProfiles(c)
	}
}
func (m Model) paintCreate(c *canvas) {
	p := m.palette()
	w, h := min(c.width, 88), min(c.height, 28)
	x, y := (c.width-w)/2, (c.height-h)/2
	title, subtitle := "NEW TRAINER", "Name   1–32 Unicode characters"
	if m.screen == dexSearchScreen {
		title = "POKÉDEX SEARCH"
		subtitle = "Visible name or National number"
	}
	c.box(x, y, w, h, title, p.accent)
	c.put(x+2, y+1, ansi.Truncate(subtitle, w-14, "…"), p.muted)
	c.button(x+w-10, y+1, 8, 7, "Quit", m)
	prefix, suffix := string(m.name[:m.cursor]), string(m.name[m.cursor:])
	fieldBoxW := min(w-4, 52)
	fieldW := fieldBoxW - 2
	offset := max(0, ansi.StringWidth(prefix)-fieldW+2)
	caret := " "
	if m.focus == 0 {
		caret = "█"
	}
	field := ansi.Cut(prefix, offset, ansi.StringWidth(prefix)) + caret + suffix
	fieldText := ansi.Truncate(field, fieldW, "…")
	fieldStyle := m.controlStyle(m.focus == 0, false)
	c.put(x+2, y+2, strings.Repeat(" ", fieldBoxW), "")
	c.put(x+3, y+2, fieldText, fieldStyle)
	if m.focus == 0 {
		c.put(x+3+ansi.StringWidth(ansi.Cut(prefix, offset, ansi.StringWidth(prefix))), y+2, "█", p.foreground)
	}
	c.put(x+2, y+3, strings.Repeat("─", fieldBoxW), p.muted)
	c.hits = append(c.hits, hit{x + 2, y + 2, fieldBoxW, 0})
	status := m.formError
	if status == "" {
		status = m.notice
	}
	if status != "" {
		c.wrap(x+2, y+4, w-4, 1, status, p.accent)
	}
	columns := min(10, (w-4)/3)
	keysX := x + (w-columns*3)/2
	for i, r := range m.keyboardLetters() {
		c.button(keysX+(i%columns)*3, y+5+i/columns, 3, 100+i, string(r), m)
	}
	page := []string{"abc → ABC", "ABC → 123", "123 → abc"}[m.keyboardPage]
	if h >= 24 {
		c.outlinedButton(x+2, y+8, 13, 3, page, m)
		c.outlinedButton(x+15, y+8, 13, 4, "Backspace", m)
		c.outlinedButton(x+28, y+8, w-30, 5, "Space", m)
	} else {
		c.button(x+2, y+8, 12, 3, page, m)
		c.button(x+15, y+8, 12, 4, "Backspace", m)
		c.button(x+28, y+8, w-30, 5, "Space", m)
	}
	if h >= 16 {
		message := "The first trainer becomes active. No encounter is recorded."
		if m.snapshot.Profiles > 0 {
			message = "Additional trainers need a separate Use action to become active."
		}
		if m.screen == dexSearchScreen {
			message = "Search only revealed names and National numbers. An empty search restores the filtered list."
		}
		messageY := y + 10
		if h >= 24 {
			messageY = y + 12
		}
		c.wrap(x+2, messageY, w-4, min(3, max(0, y+h-11-messageY)), message, p.muted)
	}
	create := "Create trainer"
	if m.screen == dexSearchScreen {
		create = "Apply search"
	}
	if m.busy {
		create = "Working…"
	}
	// The tall form reserves two framed action rows below the explanation.
	cancel := "Cancel"
	if m.busy {
		cancel = "Cancel request"
	}
	if h >= 24 {
		actionW := min((w-4)/2-1, 22)
		c.outlinedButton(x+2, y+h-11, actionW, 1, create, m)
		c.outlinedButton(x+3+actionW, y+h-11, actionW, 2, cancel, m)
		c.outlinedButton(x+2, y+h-8, 16, 6, "Appearance", m)
	} else {
		c.button(x+2, y+h-3, (w-4)/2-1, 1, create, m)
		c.button(x+2+(w-4)/2, y+h-3, (w-4)/2-1, 2, cancel, m)
		c.button(x+2, y+h-2, 16, 6, "Appearance", m)
	}
	if h >= 24 {
		c.navigationHints(x+2, y+h-5, w-4, m, false, "Select")
	} else {
		c.put(x+19, y+h-2, ansi.Truncate(m.createHelp(w-21), w-21, "…"), p.muted)
	}
}
func (m Model) paintProfiles(c *canvas) {
	p := m.palette()
	w, h := min(c.width, 88), min(c.height, max(28, min(36, len(m.snapshot.Entries)+16)))
	x, y := (c.width-w)/2, (c.height-h)/2
	c.box(x, y, w, h, "TRAINERS", p.accent)
	label := "Choose a trainer, then Use trainer."
	if m.busy {
		label = "Working… navigation remains available."
	} else if m.formError != "" {
		label = m.formError
	} else if m.notice != "" {
		label = m.notice
	}
	c.wrap(x+2, y+1, w-4, 2, label, p.muted)
	count := len(m.snapshot.Entries)
	reserve := 10
	if h >= 24 {
		reserve = 19
	}
	capacity := max(1, h-reserve)
	first := max(0, min(m.selected-capacity/2, count-capacity))
	for i := first; i < count && i < first+capacity; i++ {
		profile := m.snapshot.Entries[i]
		marker := "  "
		style := ""
		if i == m.selected {
			marker = "● "
			style = m.controlStyle(false, true)
			if m.focus == 0 {
				marker = "▶ "
				style = m.controlStyle(true, true)
			}
		}
		active := ""
		if profile.Active {
			active = "   ACTIVE"
		}
		label := ansi.Truncate(marker+clean(profile.Name)+active, w-4, "…")
		label += strings.Repeat(" ", max(0, w-4-ansi.StringWidth(label)))
		row := y + 4 + i - first
		c.put(x+2, row, label, style)
		c.hits = append(c.hits, hit{x + 2, row, w - 4, 1000 + i})
	}
	if count == 0 {
		c.button(x+2, y+4, w-4, 0, "No trainers yet.", m)
	}
	if count > capacity {
		c.put(x+2, y+4+capacity, fmt.Sprintf("%d–%d of %d   arrows/wheel scroll", first+1, min(count, first+capacity), count), p.muted)
	}
	use := "Use trainer"
	if m.busy {
		use = "Working…"
	}
	half := (w - 4) / 2
	if h >= 24 {
		ids := [3][2]int{{1, 2}, {3, 4}, {6, 5}}
		labels := [3][2]string{{"New trainer", use}, {"Back / Cancel", "Refresh"}, {"Appearance", "Quit"}}
		for row := 0; row < 3; row++ {
			for col := 0; col < 2; col++ {
				c.outlinedButton(x+2+col*min(half, 22), y+h-14+row*3, min(half-1, 21), ids[row][col], labels[row][col], m)
			}
		}
	} else {
		c.button(x+2, y+h-5, half, 1, "New trainer", m)
		c.button(x+2+half, y+h-5, half, 2, use, m)
		c.button(x+2, y+h-4, half, 3, "Back / Cancel", m)
		c.button(x+2+half, y+h-4, half, 4, "Refresh", m)
		c.button(x+2, y+h-3, half, 6, "Appearance", m)
		c.button(x+2+half, y+h-3, half, 5, "Quit", m)
	}
	if h >= 24 {
		c.navigationHints(x+2, y+h-5, w-4, m, false, "Select")
	} else {
		c.put(x+2, y+h-2, ansi.Truncate(m.profilesHelp(), w-4, "…"), p.muted)
	}
}

func (m Model) createHelp(width int) string {
	if m.focus == 0 {
		if m.busy {
			return "↓ Keys   Tab Focus"
		}
		if width < 35 {
			return "↓ Keys   Enter"
		}
		if m.screen == dexSearchScreen {
			return "↓ Keyboard   ←→ Cursor   Enter Search   Tab Focus"
		}
		return "↓ Keyboard   ←→ Cursor   Enter Create   Tab Focus"
	}
	if m.busy {
		return "↑↓←→ Move   Tab"
	}
	if m.focus >= 100 {
		if width < 35 {
			return "↑↓←→   Enter Type"
		}
		return "↑↓←→ Move   Enter Type   Tab Focus"
	}
	if width < 35 {
		return "↑↓←→   Enter"
	}
	return "↑↓←→ Move   Enter Select   Tab Focus"
}
func (m Model) profilesHelp() string {
	if m.focus == 0 {
		return "↑↓ Choose / leave   ←→ Buttons   Enter Use"
	}
	return "↑↓ Column   ←→ Row   Enter Select   Tab Focus"
}
