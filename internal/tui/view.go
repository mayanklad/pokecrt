package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/rivo/uniseg"
)

var sections = []string{"Pokédex", "Encounters", "Trainer", "Achievements"}
var descriptions = []string{
	"Your discoveries, one page at a time. Undiscovered entries keep their secrets.",
	"Every encounter adds to your journey. Only an explicit encounter action records a discovery.",
	"Your profile, collection and progress belong to your trainer.",
	"Celebrate discoveries and explore your next collection goal.",
}
var cliActions = []string{"pokecrt dex", "pokecrt encounter", "pokecrt trainer", "pokecrt trainer achievements"}

type hit struct{ x, y, width, id int }
type cell struct{ text, style string }
type canvas struct {
	width, height int
	rows          [][]cell
	hits          []hit
}

func newCanvas(w, h int) *canvas {
	c := &canvas{width: w, height: h, rows: make([][]cell, h)}
	for y := range c.rows {
		c.rows[y] = make([]cell, w)
		for x := range c.rows[y] {
			c.rows[y][x].text = " "
		}
	}
	return c
}
func (c *canvas) put(x, y int, s, style string) {
	if y < 0 || y >= c.height || x >= c.width {
		return
	}
	g := uniseg.NewGraphemes(s)
	for g.Next() {
		glyph := g.Str()
		width := ansi.StringWidth(glyph)
		if width < 1 {
			continue
		}
		if x+width > c.width {
			break
		}
		if x >= 0 {
			c.rows[y][x] = cell{glyph, style}
			for i := 1; i < width; i++ {
				c.rows[y][x+i] = cell{"", style}
			}
		}
		x += width
	}
}
func (c *canvas) wrap(x, y, w, limit int, s, style string) {
	if w < 1 || limit < 1 {
		return
	}
	lines := strings.Split(ansi.Wrap(s, w, ""), "\n")
	for i, line := range lines {
		if i >= limit {
			break
		}
		c.put(x, y+i, line, style)
	}
}
func (c *canvas) button(x, y, w, id int, label string, m Model) {
	if y < 0 || y >= c.height || w < 1 {
		return
	}
	indicator := "  "
	style := ""
	if m.focus == id {
		indicator = "▶ "
		style = m.palette().accent
		if !m.noColor {
			style = fg(6, 25, 36) + bg(87, 221, 233) + "\x1b[1m"
		}
	}
	label = ansi.Truncate(indicator+label, max(0, min(w, c.width-x)), "…")
	if m.focus == id {
		label += strings.Repeat(" ", max(0, min(w, c.width-x)-ansi.StringWidth(label)))
	}
	c.put(x, y, label, style)
	c.hits = append(c.hits, hit{x, y, min(w, c.width-x), id})
}
func (c *canvas) box(x, y, w, h int, title, style string) {
	if w < 2 || h < 2 {
		return
	}
	c.put(x, y, "╔"+strings.Repeat("═", w-2)+"╗", style)
	c.put(x, y+h-1, "╚"+strings.Repeat("═", w-2)+"╝", style)
	for row := y + 1; row < y+h-1; row++ {
		c.put(x, row, "║", style)
		c.put(x+w-1, row, "║", style)
	}
	if title != "" {
		c.put(x+2, y, ansi.Truncate(" "+title+" ", w-4, "…"), style)
	}
}
func (c *canvas) content(p palette) string {
	var out strings.Builder
	base := p.foreground + p.background
	for y, row := range c.rows {
		current := base
		if base != "" {
			out.WriteString("\x1b[0m" + base)
		}
		for _, cell := range row {
			target := base + cell.style
			if target != current {
				out.WriteString("\x1b[0m" + target)
				current = target
			}
			out.WriteString(cell.text)
		}
		if current != "" {
			out.WriteString("\x1b[0m")
		}
		if y+1 < c.height {
			out.WriteByte('\n')
		}
	}
	return out.String()
}

func (m Model) View() tea.View {
	// Bound work for enormous reported windows; normal terminals use every cell.
	w, h := max(1, min(m.width, 240)), max(1, min(m.height, 100))
	c := newCanvas(w, h)
	p := m.palette()
	if m.width < 40 || m.height < 12 {
		c.wrap(0, 0, w, min(3, h), "Resize to at least 40 columns × 12 rows.", p.accent)
		quitID := 6
		if m.settings {
			quitID = 6
		}
		if !m.settings {
			if m.screen == createScreen {
				quitID = 7
			} else if m.screen == profilesScreen {
				quitID = 5
			}
		}
		m.focus = quitID
		c.button(0, h-1, min(w, 12), quitID, "Quit", m)
	} else if m.settings {
		m.paintSettings(c)
	} else if m.screen != mainScreen {
		m.paintSetup(c)
	} else {
		m.paintMain(c)
	}
	content := c.content(p)
	v := tea.NewView(content)
	v.AltScreen = true
	v.ReportFocus = true
	v.MouseMode = tea.MouseModeCellMotion
	// Hit targets come from exactly the frame being displayed, including resize
	// and settings state. No mutable coordinates are stored during View.
	v.OnMouse = func(msg tea.MouseMsg) tea.Cmd {
		click, ok := msg.(tea.MouseClickMsg)
		if !ok || click.Button != tea.MouseLeft {
			return nil
		}
		for _, target := range c.hits {
			if click.Y == target.y && click.X >= target.x && click.X < target.x+target.width {
				id := target.id
				if id == 0 && !m.settings && m.screen == createScreen {
					cursor := m.clickedNameCursor(click.X-target.x-4, min(w, 88)-8)
					return func() tea.Msg { return nameCursorMsg(cursor) }
				}
				return func() tea.Msg { return activateMsg(id) }
			}
		}
		return nil
	}
	return v
}

func (m Model) paintMain(c *canvas) {
	w, h := c.width, c.height
	p := m.palette()
	// Keep the adventure stage cohesive on large windows rather than stretching
	// dialogue across the entire screen. All hit coordinates remain frame-derived.
	frameW, frameH := min(w, 120), min(h, 40)
	left, top := (w-frameW)/2, (h-frameH)/2
	c.box(left, top, frameW, frameH, "", p.accent)
	c.put(left+3, top+1, "◈ POKÉCRT ◈", p.accent)
	c.put(left+3, top+2, "ADVENTURE MENU", p.muted)
	profileRow := 3
	if frameW >= 90 && frameH >= 28 {
		m.paintLogo(c, left+(frameW-41)/2, top+1)
		profileRow = 4
	}
	c.put(left+3, top+profileRow, ansi.Truncate(m.profileLabel(), frameW-6, "…"), p.muted)
	c.button(left+frameW-11, top+1, 8, 6, "Quit", m)
	wide := frameW >= 90 && frameH >= 28
	if frameH >= 28 {
		sceneH := frameH - 20
		if wide {
			sceneH = frameH - 17
		}
		c.box(left+2, top+5, frameW-4, sceneH, "POKÉDEX DEVICE", p.accent)
		m.paintScene(c, left+3, top+6, frameW-6, sceneH-2)
		dy := top + 5 + sceneH
		if wide {
			menuW := 30
			textW := frameW - menuW - 5
			c.box(left+2, dy, textW, 9, "Adventure", p.accent)
			c.wrap(left+4, dy+2, textW-4, 5, m.dialogue(), "")
			mx := left + frameW - menuW - 2
			c.box(mx, dy, menuW, 9, "Choose", p.accent)
			for i, label := range sections {
				c.button(mx+2, dy+1+i, menuW-4, i, label, m)
			}
			c.button(mx+2, dy+5, menuW-4, 4, "Appearance", m)
			c.button(mx+2, dy+6, menuW-4, 5, "Refresh trainer", m)
		} else {
			c.box(left+2, dy, frameW-4, 4, "Adventure", p.accent)
			c.wrap(left+4, dy+1, frameW-8, 2, m.dialogue(), "")
			c.box(left+2, dy+4, frameW-4, 8, "Choose", p.accent)
			for i, label := range sections {
				c.button(left+4, dy+5+i, frameW-8, i, label, m)
			}
			c.button(left+4, dy+9, frameW-8, 4, "Appearance", m)
			c.button(left+4, dy+10, frameW-8, 5, "Refresh trainer", m)
		}
	} else {
		for i, label := range sections {
			c.button(left+3, top+4+i, frameW-6, i, label, m)
		}
		c.button(left+3, top+8, frameW-6, 4, "Appearance", m)
		c.button(left+3, top+9, frameW-6, 5, "Refresh trainer", m)
		if frameH >= 16 {
			c.box(left+2, top+11, frameW-4, frameH-14, "Adventure", p.accent)
			c.wrap(left+4, top+12, frameW-8, frameH-16, m.dialogue(), "")
		}
	}
	c.put(left+2, top+frameH-2, ansi.Truncate("↑↓ Choose · Enter Open · Q Quit", frameW-4, "…"), p.muted)
}

func (m Model) profileLabel() string {
	if m.loading {
		return "Reading trainer…"
	}
	if m.loadError != "" {
		return "Trainer unavailable · refresh to retry"
	}
	if m.snapshot.Active {
		return "Trainer: " + clean(m.snapshot.Name)
	}
	if m.snapshot.Profiles > 0 {
		return "Choose an active trainer with 'pokecrt trainer use <name>'."
	}
	return "Welcome, future trainer."
}
func (m Model) dialogue() string {
	if m.loadError != "" {
		return "Trainer data could not be read: " + m.loadError + ". Your data has not been changed."
	}
	if m.loading {
		return "Reading your trainer profile. You can still navigate or change appearance."
	}
	if !m.snapshot.Active {
		if m.snapshot.Profiles > 0 {
			return "Open Trainer to select or create a profile."
		}
		return "Welcome, future trainer.\nOpen Trainer to create or select a profile."
	}
	return descriptions[m.section] + " Browse this section with '" + cliActions[m.section] + "'."
}

func (m Model) paintSettings(c *canvas) {
	w, h := min(c.width, 88), min(c.height, 20)
	x, y := (c.width-w)/2, (c.height-h)/2
	p := m.palette()
	c.box(x, y, w, h, "APPEARANCE", p.accent)
	c.button(x+w-11, y+1, 8, 6, "Quit", m)
	for i, label := range appearanceNames {
		marker := "○ "
		if appearances[i] == m.appearance {
			marker = "● "
		}
		c.button(x+2, y+2+i, w-4, i, marker+label, m)
	}
	message := "Switch with Enter or a click. Terminal Native preserves your terminal background and configured transparency."
	if m.appearance == FollowTerminal {
		message = "Follow Terminal checks background replies about every 2 seconds while focused. Without replies, terminal defaults are used."
	}
	if m.noColor {
		message = "NO_COLOR is enabled. Focus and selection remain visible. Color choices apply when color is enabled."
	}
	c.put(x+2, y+6, ansi.Truncate("↑↓ Choose · Enter Apply · Tab Focus", w-4, "…"), p.muted)
	c.wrap(x+2, y+7, w-4, max(0, h-13), message, "")
	c.wrap(x+2, y+h-5, w-4, 1, m.configStatus(), p.accent)
	label := "Save appearance"
	if m.savingConfig {
		label = "Saving…"
	}
	c.button(x+2, y+h-4, w-4, 4, label, m)
	c.button(x+2, y+h-3, w-4, 5, "Back", m)
	c.put(x+2, y+h-2, ansi.Truncate(m.appearanceStatus(), w-4, "…"), p.muted)
}
