package tui

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"
)

type snapshotMsg struct {
	generation uint64
	snapshot   Snapshot
	err        error
}
type probeTimeoutMsg uint64
type pollMsg uint64
type activateMsg int

type Model struct {
	ctx                                             context.Context
	load                                            Loader
	width, height                                   int
	section, focus                                  int
	settings                                        bool
	appearance                                      Appearance
	noColor, terminalDark, backgroundKnown, focused bool
	probing                                         bool
	probeID, pollID, loadID                         uint64
	loading                                         bool
	snapshot                                        Snapshot
	loadError                                       string
}

func New(ctx context.Context, load Loader, appearance Appearance, noColor bool) Model {
	return Model{ctx: ctx, load: load, appearance: appearance, noColor: noColor, focused: true}
}
func (m Model) Init() tea.Cmd { return func() tea.Msg { return activateMsg(-1) } }

func (m Model) loadCommand() tea.Cmd {
	id, load, ctx := m.loadID, m.load, m.ctx
	return func() tea.Msg { s, err := load(ctx); return snapshotMsg{id, s, err} }
}
func (m *Model) reload() tea.Cmd {
	if m.loading || m.load == nil {
		return nil
	}
	m.loadID++
	m.loading = true
	m.loadError = ""
	return m.loadCommand()
}
func (m *Model) probe() tea.Cmd {
	if m.noColor || m.appearance != FollowTerminal || m.probing || !m.focused {
		return nil
	}
	m.probing = true
	m.probeID++
	id := m.probeID
	return tea.Batch(tea.RequestBackgroundColor, tea.Tick(750*time.Millisecond, func(time.Time) tea.Msg { return probeTimeoutMsg(id) }))
}
func (m *Model) poll() tea.Cmd {
	m.pollID++
	if m.noColor || m.appearance != FollowTerminal || !m.backgroundKnown || !m.focused {
		return nil
	}
	id := m.pollID
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg { return pollMsg(id) })
}
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case activateMsg:
		if msg == -1 {
			cmd := tea.Batch(m.reload(), m.probe())
			return m, cmd
		}
		cmd := m.activate(int(msg))
		return m, cmd
	case snapshotMsg:
		if msg.generation != m.loadID {
			return m, nil
		}
		m.loading = false
		if msg.err != nil {
			m.loadError = clean(msg.err.Error())
			m.snapshot = Snapshot{}
		} else {
			m.snapshot = msg.snapshot
		}
	case tea.BackgroundColorMsg:
		if msg.Color == nil {
			return m, nil
		}
		m.terminalDark = msg.IsDark()
		m.backgroundKnown = true
		m.probing = false
		cmd := m.poll()
		return m, cmd
	case uv.DarkColorSchemeEvent:
		m.terminalDark = true
		m.backgroundKnown = true
	case uv.LightColorSchemeEvent:
		m.terminalDark = false
		m.backgroundKnown = true
	case probeTimeoutMsg:
		if uint64(msg) != m.probeID || !m.probing {
			return m, nil
		}
		m.probing = false
		m.backgroundKnown = false
		m.pollID++
	case pollMsg:
		if uint64(msg) != m.pollID {
			return m, nil
		}
		cmd := m.probe()
		return m, cmd
	case tea.BlurMsg:
		m.focused = false
		m.pollID++
	case tea.FocusMsg:
		m.focused = true
		cmd := m.probe()
		return m, cmd
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if !m.settings || m.width < 40 || m.height < 12 {
				return m, tea.Quit
			}
		case "esc":
			if m.settings {
				m.settings = false
				m.focus = 4
			} else {
				return m, tea.Quit
			}
		case "a":
			if !m.settings {
				m.openSettings()
			}
		case "tab", "down", "right", "j":
			m.moveFocus(1)
		case "shift+tab", "up", "left", "k":
			m.moveFocus(-1)
		case "enter", "space":
			cmd := m.activate(m.focus)
			return m, cmd
		case "r":
			if !m.settings {
				cmd := m.reload()
				return m, cmd
			}
		}
	case tea.MouseWheelMsg:
		if msg.Button == tea.MouseWheelDown || msg.Button == tea.MouseWheelRight {
			m.moveFocus(1)
		}
		if msg.Button == tea.MouseWheelUp || msg.Button == tea.MouseWheelLeft {
			m.moveFocus(-1)
		}
	}
	return m, nil
}
func (m *Model) moveFocus(delta int) {
	count := 7
	if m.settings {
		count = 6
	}
	m.focus = (m.focus + delta + count) % count
}
func (m *Model) openSettings() {
	m.settings = true
	m.focus = 0
	for i, a := range appearances {
		if a == m.appearance {
			m.focus = i
		}
	}
}
func (m *Model) activate(id int) tea.Cmd {
	if m.settings {
		switch {
		case id >= 0 && id < 4:
			m.focus = id
			m.appearance = appearances[id]
			m.pollID++
			// Invalidate a pending timeout; switching away never resets terminal defaults.
			m.probeID++
			m.probing = false
			if m.appearance == FollowTerminal {
				return m.probe()
			}
		case id == 4:
			m.settings = false
			m.focus = 4
		case id == 5:
			return tea.Quit
		}
		return nil
	}
	if id < 0 || id > 6 {
		return nil
	}
	m.focus = id
	switch id {
	case 0, 1, 2, 3:
		m.section = id
	case 4:
		m.openSettings()
	case 5:
		return m.reload()
	case 6:
		return tea.Quit
	}
	return nil
}
