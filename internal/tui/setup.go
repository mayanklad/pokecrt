package tui

import (
	"context"
	"errors"
	"github.com/charmbracelet/x/ansi"
	"unicode"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"github.com/mayanklad/pokecrt/internal/trainer"
)

type setupScreen uint8

const (
	mainScreen setupScreen = iota
	createScreen
	profilesScreen
	dexScreen
	dexSearchScreen
	activityScreen
)

type profileResult struct {
	id       uint64
	created  bool
	profile  trainer.Profile
	err      error
	snapshot Snapshot
	readErr  error
}
type configResult struct {
	appearance Appearance
	err        error
}
type savedConfigMsg struct {
	appearance Appearance
	err        error
}

func (m *Model) openProfiles() {
	m.screen = profilesScreen
	m.focus = 0
	m.formError = ""
	m.syncSelected("")
}
func (m *Model) openCreate() {
	m.screen = createScreen
	m.focus = 0
	m.name = nil
	m.cursor = 0
	m.formError = ""
	m.notice = ""
}
func (m *Model) syncSelected(name string) {
	m.selected = min(m.selected, max(0, len(m.snapshot.Entries)-1))
	for i, p := range m.snapshot.Entries {
		if p.Name == name || (name == "" && p.Active) {
			m.selected = i
			return
		}
	}
}
func (m *Model) setupBack() tea.Cmd {
	if m.busy {
		if m.opCancel != nil {
			m.opCancel()
		}
		m.notice = "Cancelling request…"
		return nil
	}
	if m.screen == createScreen && m.snapshot.Profiles > 0 {
		m.openProfiles()
		return nil
	}
	if !m.snapshot.Active {
		return tea.Quit
	}
	m.screen = mainScreen
	m.focus = 2
	m.formError = ""
	return nil
}
func (m *Model) insertName(s string) {
	if !utf8.ValidString(s) {
		m.formError = "Use valid UTF-8."
		return
	}
	for _, r := range s {
		if unicode.IsControl(r) || r == '\u2028' || r == '\u2029' {
			m.formError = "Names cannot contain control characters or newlines."
			return
		}
	}
	runes := []rune(s)
	if len(m.name)+len(runes) > 128 {
		m.formError = "Name input is too long; use 1–32 characters."
		return
	}
	tail := append([]rune(nil), m.name[m.cursor:]...)
	m.name = append(m.name[:m.cursor], runes...)
	m.name = append(m.name, tail...)
	m.cursor += len(runes)
	m.formError = ""
}
func (m *Model) editName(k tea.KeyPressMsg) tea.Cmd {
	switch k.String() {
	case "enter":
		return m.startProfile(true)
	case "left":
		m.cursor = max(0, m.cursor-1)
	case "right":
		m.cursor = min(len(m.name), m.cursor+1)
	case "home", "ctrl+a":
		m.cursor = 0
	case "end", "ctrl+e":
		m.cursor = len(m.name)
	case "backspace":
		if m.cursor > 0 {
			m.name = append(m.name[:m.cursor-1], m.name[m.cursor:]...)
			m.cursor--
			m.formError = ""
		}
	case "delete":
		if m.cursor < len(m.name) {
			m.name = append(m.name[:m.cursor], m.name[m.cursor+1:]...)
			m.formError = ""
		}
	case "ctrl+u":
		m.name = nil
		m.cursor = 0
		m.formError = ""
	case "down":
		m.focus = 100
	case "up":
		m.focus = 7
	default:
		if k.Text != "" {
			m.insertName(k.Text)
		} else if k.Code == ' ' {
			m.insertName(" ")
		} else if k.Mod == 0 && k.Code >= 32 && k.Code <= unicode.MaxRune {
			m.insertName(string(k.Code))
		}
	}
	return nil
}
func (m *Model) handleSetup(msg tea.Msg) (tea.Cmd, bool) {
	if m.settings || (m.screen != createScreen && m.screen != profilesScreen) {
		return nil, false
	}
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if m.width < 40 || m.height < 12 {
			if msg.String() == "q" || msg.String() == "esc" {
				return tea.Quit, true
			}
		}
	}
	switch msg := msg.(type) {
	case tea.PasteMsg:
		if m.screen == createScreen && m.focus == 0 && !m.busy {
			m.insertName(msg.Content)
		}
		return nil, true
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			return tea.Quit, true
		case "esc":
			return m.setupBack(), true
		case "tab":
			m.moveFocus(1)
			return nil, true
		case "shift+tab":
			m.moveFocus(-1)
			return nil, true
		}
		if m.screen == createScreen && m.focus == 0 {
			if m.busy {
				if msg.String() == "up" || msg.String() == "down" {
					m.navigateSetup(msg.String())
				}
				return nil, true
			}
			return m.editName(msg), true
		}
		switch msg.String() {
		case "q":
			return tea.Quit, true
		case "a":
			m.openSettings()
			return nil, true
		case "up", "k", "down", "j", "right", "left":
			direction := msg.String()
			if direction == "k" {
				direction = "up"
			}
			if direction == "j" {
				direction = "down"
			}
			m.navigateSetup(direction)
		case "enter", "space":
			return m.activateSetup(m.focus), true
		}
		return nil, true
	case tea.MouseWheelMsg:
		if m.screen == profilesScreen && len(m.snapshot.Entries) > 0 {
			d := 1
			if msg.Button == tea.MouseWheelUp {
				d = -1
			}
			m.selected = max(0, min(m.selected+d, len(m.snapshot.Entries)-1))
		}
		return nil, true
	}
	return nil, false
}
func (m *Model) activateSetup(id int) tea.Cmd {
	if m.screen == createScreen && ((id >= 0 && id <= 7) || (id >= 100 && id < 100+len(m.keyboardLetters()))) {
		m.focus = id
	}
	if m.screen == profilesScreen && id >= 0 && id <= 6 {
		m.focus = id
	}
	if m.screen == createScreen {
		if id >= 100 {
			letters := m.keyboardLetters()
			i := id - 100
			if i < len(letters) && !m.busy {
				m.insertName(string(letters[i]))
			}
			return nil
		}
		switch id {
		case 0:
			m.focus = 0
		case 1:
			return m.startProfile(true)
		case 2:
			return m.setupBack()
		case 3:
			m.keyboardPage = (m.keyboardPage + 1) % 3
		case 4:
			if !m.busy {
				return m.editName(tea.KeyPressMsg{Code: tea.KeyBackspace})
			}
		case 5:
			if !m.busy {
				m.insertName(" ")
			}
		case 6:
			m.openSettings()
		case 7:
			return tea.Quit
		}
	} else {
		if id >= 1000 {
			m.selected = min(id-1000, max(0, len(m.snapshot.Entries)-1))
			m.focus = 0
			return nil
		}
		switch id {
		case 0, 2:
			return m.startProfile(false)
		case 1:
			if !m.busy {
				m.openCreate()
			}
		case 3:
			return m.setupBack()
		case 4:
			return m.reload()
		case 5:
			return tea.Quit
		case 6:
			m.openSettings()
		}
	}
	return nil
}
func (m Model) keyboardLetters() []rune {
	switch m.keyboardPage {
	case 1:
		return []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	case 2:
		return []rune("0123456789-_'.,!?@#+&()/:")
	}
	return []rune("abcdefghijklmnopqrstuvwxyz")
}
func (m *Model) startProfile(create bool) tea.Cmd {
	if m.busy || m.actions == nil {
		return nil
	}
	raw := string(m.name)
	action := m.actions.create
	if !create {
		if len(m.snapshot.Entries) == 0 {
			m.formError = "No trainers yet. Choose New trainer."
			return nil
		}
		m.selected = min(m.selected, len(m.snapshot.Entries)-1)
		raw = m.snapshot.Entries[m.selected].Name
		action = m.actions.use
	}
	if action == nil {
		return nil
	}
	if _, err := trainer.ParseName(raw); err != nil {
		m.formError = clean(err.Error())
		if create {
			m.focus = 0
		}
		return nil
	}
	ctx, cancel := context.WithCancel(m.ctx)
	m.opCancel = cancel
	m.opID++
	id := m.opID
	m.busy = true
	m.formError = ""
	m.notice = "Working…"
	m.loadID++
	m.loading = false
	load, parent := m.load, m.ctx
	return func() tea.Msg {
		defer cancel()
		p, err := action(ctx, raw)
		var s Snapshot
		var readErr error
		if load != nil {
			s, readErr = load(parent)
		}
		return profileResult{id, create, p, err, s, readErr}
	}
}
func (m *Model) profileDone(msg profileResult) {
	if msg.id != m.opID || !m.busy {
		return
	}
	m.busy = false
	m.loadID++
	m.loading = false
	m.opCancel = nil
	if msg.readErr == nil {
		m.snapshot = msg.snapshot
		m.loadError = ""
	} else {
		m.loadError = clean(msg.readErr.Error())
	}
	if msg.err != nil {
		m.notice = ""
		m.formError = clean(msg.err.Error())
		if errors.Is(msg.err, context.Canceled) {
			m.formError = "Request cancelled. Trainer list refreshed; check before retrying."
		}
		return
	}
	m.formError = ""
	if msg.created && !msg.profile.Active {
		m.openProfiles()
		m.syncSelected(msg.profile.Name)
		m.notice = "Created " + clean(msg.profile.Name) + ". Choose Use trainer to activate."
	} else {
		m.screen = mainScreen
		m.focus = 2
		m.notice = "Using trainer: " + clean(msg.profile.Name)
	}
	if msg.readErr != nil {
		m.notice += ". Refresh to confirm current trainer status."
	}
}
func (m *Model) saveSettings() tea.Cmd {
	if m.savingConfig || m.configSave == nil {
		return nil
	}
	m.savingConfig = true
	m.configWriteStarted = true
	m.appearanceRevision++
	m.configError = ""
	a, save, ctx := m.appearance, m.configSave, m.ctx
	return func() tea.Msg { return savedConfigMsg{a, save(ctx, a)} }
}
func (m *Model) applyConfig(msg configResult) tea.Cmd {
	m.configLoaded = true
	if m.configWriteStarted {
		return nil
	}
	if msg.err != nil {
		m.configError = clean(msg.err.Error())
		return nil
	}
	m.savedAppearance = msg.appearance
	if msg.appearance == "" || m.appearanceOverride || m.appearanceRevision > 0 {
		return nil
	}
	m.appearance = msg.appearance
	m.probeID++
	m.probing = false
	m.pollID++
	return m.probe()
}
func (m Model) configStatus() string {
	if m.savingConfig {
		return "Saving appearance…"
	}
	if m.configError != "" {
		return "Settings: " + m.configError + "   current mode remains usable."
	}
	if m.savedAppearance == "" {
		return "Live preview   Save as default to remember it."
	}
	if m.savedAppearance != m.appearance {
		return "Saved: " + string(m.savedAppearance) + "   current choice is not saved."
	}
	return "Saved: " + string(m.savedAppearance)
}

// Translate a click in the scrolled field to a Unicode rune cursor. Mouse-only
// users can correct the middle of a name without clearing the rest of it.
func (m Model) clickedNameCursor(column, fieldWidth int) int {
	prefixWidth := ansi.StringWidth(string(m.name[:m.cursor]))
	offset := max(0, prefixWidth-fieldWidth+2)
	target := offset + max(0, column)
	if target > prefixWidth {
		target--
	}
	width := 0
	for i, r := range m.name {
		rw := ansi.StringWidth(string(r))
		if width+rw > target {
			return i
		}
		width += rw
	}
	return len(m.name)
}

// navigateSetup follows the visible rows; the list and letter grid have exits.
func (m *Model) navigateSetup(direction string) {
	if m.screen == profilesScreen {
		if m.focus == 0 {
			switch direction {
			case "up":
				if m.selected > 0 {
					m.selected--
				}
			case "down":
				if m.selected+1 < len(m.snapshot.Entries) {
					m.selected++
				} else {
					m.focus = 1
				}
			case "left":
				m.focus = 1
			case "right":
				m.focus = 2
			}
			return
		}
		switch direction {
		case "up":
			switch m.focus {
			case 1, 2:
				m.focus = 0
			case 3:
				m.focus = 1
			case 4:
				m.focus = 2
			case 6:
				m.focus = 3
			case 5:
				m.focus = 4
			}
		case "down":
			switch m.focus {
			case 1:
				m.focus = 3
			case 2:
				m.focus = 4
			case 3:
				m.focus = 6
			case 4:
				m.focus = 5
			case 5, 6:
				// Bottom edge: stay in the same column.
			}
		case "left":
			switch m.focus {
			case 2:
				m.focus = 1
			case 4:
				m.focus = 3
			case 5:
				m.focus = 6
			}
		case "right":
			switch m.focus {
			case 1:
				m.focus = 2
			case 3:
				m.focus = 4
			case 6:
				m.focus = 5
			}
		}
		return
	}
	n := len(m.keyboardLetters())
	if m.focus >= 100 {
		i := m.focus - 100
		switch direction {
		case "left":
			if i%10 > 0 {
				m.focus--
			}
		case "right":
			if i%10 < 9 && i+1 < n {
				m.focus++
			}
		case "up":
			if i >= 10 {
				m.focus -= 10
			} else {
				m.focus = 0
			}
		case "down":
			if i+10 < n {
				m.focus += 10
			} else {
				targets := []dexControl{}
				for _, target := range m.createFocusTargets() {
					if target.id == m.focus || target.id >= 3 && target.id <= 5 {
						targets = append(targets, target)
					}
				}
				m.focus = directionalTarget(targets, m.focus, "down")
			}
		}
		return
	}
	m.focus = directionalTarget(m.createFocusTargets(), m.focus, direction)
}

func (m Model) createFocusTargets() []dexControl {
	c := newCanvas(max(40, min(m.width, 240)), max(12, min(m.height, 100)))
	m.paintCreate(c)
	targets := []dexControl{}
	positions := map[int]int{}
	for _, h := range c.hits {
		if m.busy && h.id != 0 && h.id != 2 && h.id != 6 && h.id != 7 {
			continue
		}
		if index, ok := positions[h.id]; ok {
			// Store the centre row of the whole framed control, not its top border.
			targets[index].y = (targets[index].y + h.y) / 2
		} else {
			positions[h.id] = len(targets)
			targets = append(targets, dexControl{h.x, h.y, h.width, h.id, ""})
		}
	}
	return targets
}
