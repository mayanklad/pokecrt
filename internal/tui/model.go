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
type nameCursorMsg int

type Model struct {
	encounterPending bool
	encounterID      uint64
	encounterNotice  string
	activity         activityState
	activityLoader   activityLoader
	encounterAction  encounterAction
	dex              dexState
	dexLoader        dexLoader

	configWriteStarted                              bool
	screen                                          setupScreen
	actions                                         *profileActions
	selected                                        int
	name                                            []rune
	cursor, keyboardPage                            int
	busy                                            bool
	opID                                            uint64
	opCancel                                        context.CancelFunc
	formError, notice                               string
	routed                                          bool
	configLoad                                      func(context.Context) (Appearance, error)
	configSave                                      func(context.Context, Appearance) error
	configLoaded, appearanceOverride, savingConfig  bool
	appearanceRevision                              uint64
	savedAppearance                                 Appearance
	configError                                     string
	ctx                                             context.Context
	load                                            Loader
	width, height                                   int
	section, focus                                  int
	settingsReturnFocus                             int
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
	if m.loading || m.busy || m.load == nil {
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
	// Below the supported size, only the displayed Quit action is available.
	if m.width > 0 && m.height > 0 && (m.width < 40 || m.height < 12) {
		switch k := msg.(type) {
		case tea.KeyPressMsg:
			switch k.String() {
			case "ctrl+c", "q", "esc", "enter", "space":
				return m, tea.Quit
			}
			return m, nil
		case tea.PasteMsg, tea.MouseWheelMsg:
			return m, nil
		}
	}

	if cmd, handled := m.handleActivity(msg); handled {
		return m, cmd
	}
	if cmd, handled := m.handleDex(msg); handled {
		return m, cmd
	}
	if cmd, handled := m.handleSetup(msg); handled {
		return m, cmd
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		if m.screen == dexScreen && !m.dexWide() && (m.focus == 10 || m.focus == 15 || m.focus >= 2000 || m.settings && (m.settingsReturnFocus == 10 || m.settingsReturnFocus == 15 || m.settingsReturnFocus >= 2000)) {
			m.dex.detail = true
		}
	case activateMsg:
		if msg == -1 {
			var cfg tea.Cmd
			if m.configLoad != nil {
				load, ctx := m.configLoad, m.ctx
				cfg = func() tea.Msg { a, err := load(ctx); return configResult{a, err} }
			}
			cmd := tea.Batch(m.reload(), m.probe(), cfg)
			return m, cmd
		}
		cmd := m.activate(int(msg))
		return m, cmd
	case nameCursorMsg:
		if (m.screen == createScreen || m.screen == dexSearchScreen) && !m.busy {
			m.focus = 0
			m.cursor = max(0, min(len(m.name), int(msg)))
		}
	case profileResult:
		m.profileDone(msg)
	case configResult:
		cmd := m.applyConfig(msg)
		return m, cmd
	case savedConfigMsg:
		m.savingConfig = false
		if msg.err != nil {
			m.configError = clean(msg.err.Error())
		} else {
			m.savedAppearance = msg.appearance
			m.configError = ""
		}
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
			if m.screen == profilesScreen {
				m.selected = min(m.selected, max(0, len(m.snapshot.Entries)-1))
			}
			if !m.routed && m.actions != nil {
				m.routed = true
				if !m.settings && m.screen == mainScreen && !m.snapshot.Active {
					if m.snapshot.Profiles == 0 {
						m.openCreate()
					} else {
						m.openProfiles()
					}
				}
			}
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
				m.closeSettings()
				if m.screen == activityScreen {
					return m, tea.ClearScreen
				}
			} else {
				return m, tea.Quit
			}
		case "a":
			if !m.settings {
				m.openSettings()
			}
		case "tab":
			m.moveFocus(1)
		case "shift+tab":
			m.moveFocus(-1)
		case "down", "up", "left", "right", "j", "k":
			direction := msg.String()
			if direction == "j" {
				direction = "down"
			}
			if direction == "k" {
				direction = "up"
			}
			if m.settings {
				m.navigateSettings(direction)
			} else {
				if m.height >= 28 && direction == "right" && m.focus == 5 {
					m.focus = 6
				}
				if m.height >= 28 && direction == "left" && m.focus == 6 {
					m.focus = 5
				}
				if direction == "down" {
					if m.height >= 28 {
						if m.focus < 5 {
							m.focus++
						}
					} else {
						m.focus = min(6, m.focus+1)
					}
				}
				if direction == "up" {
					if m.height >= 28 && m.focus >= 5 {
						m.focus = 4
					} else {
						m.focus = max(0, m.focus-1)
					}
				}
			}
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
			if m.settings {
				m.navigateSettings("down")
			} else {
				m.focus = min(6, m.focus+1)
			}
		}
		if msg.Button == tea.MouseWheelUp || msg.Button == tea.MouseWheelLeft {
			if m.settings {
				m.navigateSettings("up")
			} else {
				m.focus = max(0, m.focus-1)
			}
		}
	}
	return m, nil
}
func (m *Model) moveFocus(delta int) {
	order := []int{0, 1, 2, 3, 4, 5, 6}
	if m.settings {
		order = []int{6, 0, 1, 2, 3, 4, 5}
	} else if m.screen == profilesScreen {
		order = []int{0, 1, 2, 3, 4, 6, 5}
	} else if m.screen == createScreen || m.screen == dexSearchScreen {
		order = []int{7, 0}
		for i := range m.keyboardLetters() {
			order = append(order, 100+i)
		}
		order = append(order, 3, 4, 5, 1, 2, 6)
		if m.busy {
			order = []int{0, 2, 6, 7}
		}
	}
	for i, id := range order {
		if id == m.focus {
			m.focus = order[(i+delta+len(order))%len(order)]
			return
		}
	}
	m.focus = 0
}
func (m *Model) openSettings() {
	m.settingsReturnFocus = m.focus
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
			m.appearanceRevision++
			m.pollID++
			// Invalidate a pending timeout; switching away never resets terminal defaults.
			m.probeID++
			m.probing = false
			if m.appearance == FollowTerminal {
				return m.probe()
			}
		case id == 4:
			return m.saveSettings()
		case id == 5:
			m.closeSettings()
			if m.screen == activityScreen {
				return tea.ClearScreen
			}
		case id == 6:
			return tea.Quit
		}
		return nil
	}
	if m.screen == activityScreen {
		return m.activateActivity(id)
	}
	if m.screen == dexScreen || m.screen == dexSearchScreen {
		return m.activateDex(id)
	}
	if m.screen != mainScreen {
		return m.activateSetup(id)
	}
	if id < 0 || id > 6 {
		return nil
	}
	m.focus = id
	switch id {
	case 0, 1, 2, 3:
		m.section = id
		if id == 0 && m.dexLoader != nil {
			return m.openDex()
		}
		if id != 0 && m.activityLoader != nil {
			return m.openActivity(id)
		}
		if id == 2 && m.actions != nil {
			m.openProfiles()
			return m.reload()
		}
	case 4:
		m.openSettings()
	case 5:
		return m.reload()
	case 6:
		return tea.Quit
	}
	return nil
}

func (m *Model) closeSettings() {
	m.settings = false
	m.focus = 4
	if m.screen != mainScreen {
		m.focus = m.settingsReturnFocus
	}
}

func (m *Model) navigateSettings(direction string) {
	// Appearance choices are a vertical group; Save and Back share the next row.
	if m.focus < 4 {
		if direction == "down" {
			if m.focus < 3 {
				m.focus++
			} else {
				m.focus = 4
			}
		}
		if direction == "up" {
			if m.focus > 0 {
				m.focus--
			} else {
				m.focus = 6
			}
		}

		return
	}
	switch direction {
	case "up":
		if m.focus == 4 || m.focus == 5 {
			m.focus = 3
		}
	case "down":
		if m.focus == 6 {
			m.focus = 0
		}
	case "left":
		if m.focus == 5 {
			m.focus = 4
		} else if m.focus == 6 {
			m.focus = 0
		}
	case "right":
		if m.focus == 4 {
			m.focus = 5
		}
	}
}
