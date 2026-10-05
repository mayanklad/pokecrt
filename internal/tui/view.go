package tui

import (
	"fmt"
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
	// Text fields keep their editing delimiters and cursor mapping.
	if strings.HasPrefix(label, "[") && strings.HasSuffix(label, "]") && !strings.Contains(label, "▏") {
		label = strings.TrimSpace(label[1 : len(label)-1])
	}
	menuIcon := ""
	menu := m.screen == mainScreen && !m.settings && id >= 0 && id <= 5
	if menu {
		icons := []string{"▤", "◆", "♟", "★", "◇", "↻"}
		menuIcon = icons[id]
		style, _, _ = m.controlColours(id, label)
		label = "[" + menuIcon + "]  " + label
	} else if w >= 3 && !strings.Contains(label, "▏") && id < 1000 && !(id >= 100 && id < 200) {
		selected := m.settings && id < 4 && appearances[id] == m.appearance
		if m.screen == dexScreen && !m.settings {
			selected = id >= 20 && id <= 23 && m.dex.tab == id-20 || id >= 31 && id <= 33 && m.dex.status == id-31
		}
		marker := ""
		if selected {
			marker = "● "
		}
		if m.focus == id {
			marker = "▶ "
			if selected {
				marker = "▶● "
			}
		}
		label = strings.TrimPrefix(strings.TrimPrefix(label, "○ "), "● ")
		text := ansi.Truncate(marker+label, w-2, "…")
		left := max(0, (w-2-ansi.StringWidth(text))/2)
		style, _, _ = m.controlColours(id, label)
		if m.focus == id {
			style = m.controlStyle(true, selected)
		}
		if selected && m.focus != id {
			style += m.palette().gold
		}
		c.put(x, y, "❨"+strings.Repeat(" ", left)+text+strings.Repeat(" ", max(0, w-2-left-ansi.StringWidth(text)))+"❩", style)
		c.hits = append(c.hits, hit{x, y, w, id})
		return
	}
	if m.focus == id {
		indicator = "▶ "
		style = m.palette().accent
		if !m.noColor {
			style = m.controlStyle(true, false)
		}
	}
	label = ansi.Truncate(indicator+label, max(0, min(w, c.width-x)), "…")
	if m.focus == id || menu {
		label += strings.Repeat(" ", max(0, min(w, c.width-x)-ansi.StringWidth(label)))
	}
	c.put(x, y, label, style)
	if menu && m.focus != id {
		c.put(x+3, y, menuIcon, m.palette().gold)
	}
	c.hits = append(c.hits, hit{x, y, min(w, c.width-x), id})
}
func (c *canvas) box(x, y, w, h int, title, style string) {
	if w < 2 || h < 2 {
		return
	}
	c.put(x, y, "╭"+strings.Repeat("─", w-2)+"╮", style)
	c.put(x, y+h-1, "╰"+strings.Repeat("─", w-2)+"╯", style)
	for row := y + 1; row < y+h-1; row++ {
		c.put(x, row, "│", style)
		c.put(x+w-1, row, "│", style)
	}

	if title != "" {
		c.put(x+2, y, ansi.Truncate(" "+strings.TrimSpace(title)+" ", w-4, "…"), style)
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
			if m.screen == activityScreen {
				quitID = 14
			} else if m.screen == dexScreen {
				quitID = 8
			} else if m.screen == dexSearchScreen {
				quitID = 7
			} else if m.screen == createScreen {
				quitID = 7
			} else if m.screen == profilesScreen {
				quitID = 5
			}
		}
		m.focus = quitID
		c.button(0, h-1, min(w, 12), quitID, "Quit", m)
	} else if m.settings {
		m.paintSettings(c)
	} else if m.screen == activityScreen {
		m.paintActivity(c)
	} else if m.screen == dexScreen || m.screen == dexSearchScreen {
		m.paintDex(c)
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
				if id == 0 && !m.settings && (m.screen == createScreen || m.screen == dexSearchScreen) {
					cursor := m.clickedNameCursor(click.X-target.x-1, min(min(w, 88)-4, 52)-2)
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
	wide := frameW >= 90 && frameH >= 28
	if wide {
		lx := left + (frameW-41)/2
		m.paintLogo(c, lx, top+1)
		c.put(lx+14, top+4, "TRAINER HUB", p.gold)
		px := left + frameW - 30
		c.box(px, top+1, 28, 4, "", p.accent)
		name := clean(m.snapshot.Name)
		if name == "" {
			name = "No trainer yet"
		}
		if m.loading {
			name = "Reading trainer…"
		}
		c.put(px+2, top+2, ansi.Truncate("♟ "+name, 24, "…"), p.foreground)
		status := "Create or choose a trainer"
		if m.snapshot.Active {
			status = fmt.Sprintf("Level %d", max(1, m.snapshot.Level))
		}
		c.put(px+2, top+3, ansi.Truncate(status, 24, "…"), p.muted)
	} else {
		c.put(left+3, top+1, "◈ POKÉCRT ◈", p.accent)
		c.put(left+3, top+2, "TRAINER HUB", p.gold)
		c.put(left+3, top+3, ansi.Truncate(m.profileLabel(), frameW-6, "…"), p.muted)
	}
	if frameH < 28 {
		c.button(left+frameW-11, top+1, 8, 6, "Quit", m)
	}
	if frameH >= 28 {
		sceneH := frameH - 24
		if wide {
			sceneH = frameH - 20
		}
		c.box(left+2, top+5, frameW-4, sceneH, "TOWN MAP", p.accent)
		m.paintScene(c, left+3, top+6, frameW-6, sceneH-2)
		dy := top + 5 + sceneH
		if wide {
			menuW := 30
			textW := frameW - menuW - 5
			c.box(left+2, dy, textW, 8, m.homeTitle(), p.accent)
			c.wrap(left+4, dy+2, textW-4, 5, m.dialogue(), "")
			mx := left + frameW - menuW - 2
			c.box(mx, dy, menuW, 8, "Choose", p.accent)
			for i, label := range sections {
				c.button(mx+2, dy+1+i, menuW-4, i, label, m)
			}
			c.button(mx+2, dy+5, menuW-4, 4, "Appearance", m)
		} else {
			c.box(left+2, dy, frameW-4, 4, m.homeTitle(), p.accent)
			c.wrap(left+4, dy+1, frameW-8, 2, m.dialogue(), "")
			c.box(left+2, dy+4, frameW-4, 8, "Choose", p.accent)
			for i, label := range sections {
				c.button(left+4, dy+5+i, frameW-8, i, label, m)
			}
			c.button(left+4, dy+9, frameW-8, 4, "Appearance", m)
		}
	} else {
		for i, label := range sections {
			c.button(left+3, top+4+i, frameW-6, i, label, m)
		}
		c.button(left+3, top+8, frameW-6, 4, "Appearance", m)
		c.button(left+3, top+9, frameW-6, 5, "Refresh", m)
		if frameH >= 16 {
			c.box(left+2, top+11, frameW-4, frameH-14, m.homeTitle(), p.accent)
			c.wrap(left+4, top+12, frameW-8, frameH-16, m.dialogue(), "")
		}
	}
	if frameH >= 28 {
		c.outlinedButton(left+3, top+frameH-7, 14, 5, "Refresh", m)
		c.outlinedButton(left+18, top+frameH-7, 12, 6, "Quit", m)
		c.navigationHints(left+3, top+frameH-4, frameW-6, m, false, "Open")

	} else {
		c.put(left+2, top+frameH-2, ansi.Truncate("↑↓ Choose   Enter Open   Q Quit", frameW-4, "…"), p.muted)
	}
}

func (m Model) homeTitle() string {
	if m.focus >= 0 && m.focus < len(sections) {
		return sections[m.focus]
	}
	if m.focus == 4 {
		return "Appearance"
	}
	if m.focus == 5 {
		return "Refresh"
	}
	return "Quit"
}

func (m Model) profileLabel() string {
	if m.loading {
		return "Reading trainer…"
	}
	if m.loadError != "" {
		return "Trainer unavailable   refresh to retry"
	}
	if m.snapshot.Active {
		return "Trainer: " + clean(m.snapshot.Name) + fmt.Sprintf("   Level %d", max(1, m.snapshot.Level))
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
	if m.focus >= 0 && m.focus < len(descriptions) {
		return descriptions[m.focus]
	}
	if m.focus == 4 {
		return "Choose Dark, Light, Follow Terminal or Terminal Native. Changes preview immediately."
	}
	if m.focus == 5 {
		return "Reload this trainer’s latest collection and progress."
	}
	return "Close PokéCRT and return to your terminal."
}

func (m Model) paintSettings(c *canvas) {
	w, h := min(c.width, 88), min(c.height, 28)
	x, y := (c.width-w)/2, (c.height-h)/2
	p := m.palette()
	c.box(x, y, w, h, "APPEARANCE", p.accent)
	c.button(x+w-11, y+1, 8, 6, "Quit", m)
	for i, label := range appearanceNames {
		marker := "○ "
		if appearances[i] == m.appearance {
			marker = "● "
		}
		if h >= 24 {
			c.framedControl(x+2, y+2+i*3, w-4, i, label, m, appearances[i] == m.appearance)
		} else {
			c.button(x+2, y+2+i, w-4, i, marker+label, m)
		}
	}
	message := "Switch with Enter or a click. Terminal Native preserves your terminal background and configured transparency."
	if m.appearance == FollowTerminal {
		message = "Follow Terminal checks background replies about every 2 seconds while focused. Without replies, terminal defaults are used."
	}
	if m.noColor {
		message = "NO_COLOR is enabled. Focus and selection remain visible. Color choices apply when color is enabled."
	}
	helpY := y + 6
	if h >= 24 {
		helpY = y + 14
	}
	c.put(x+2, helpY, ansi.Truncate("Changes preview immediately.", w-4, "…"), p.muted)
	c.wrap(x+2, helpY+1, w-4, max(0, y+h-10-helpY-1), message, "")
	statusY := y + h - 5
	if h >= 24 {
		statusY = y + h - 9
	}
	c.wrap(x+2, statusY, w-4, 1, m.configStatus(), p.accent)
	label := "Save as default"
	if m.savingConfig {
		label = "Saving…"
	}
	if h >= 24 {
		c.outlinedButton(x+2, y+h-8, (w-4)/2-1, 4, label, m)
		c.outlinedButton(x+3+min((w-4)/2-1, 21), y+h-8, min((w-4)/2-1, 21), 5, "Back", m)
		c.navigationHints(x+2, y+h-5, w-4, m, false, "Apply")
	} else {
		c.button(x+2, y+h-3, (w-4)/2-1, 4, label, m)
		c.button(x+2+(w-4)/2, y+h-3, (w-4)/2-1, 5, "Back", m)
		c.navigationHints(x+2, y+h-2, w-4, m, true, "Apply")
	}
}
