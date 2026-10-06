package tui

import (
	tea "charm.land/bubbletea/v2"
	"context"
	"github.com/charmbracelet/x/ansi"
	"github.com/mayanklad/pokecrt/internal/catalog"
	"github.com/mayanklad/pokecrt/internal/render"
	"github.com/mayanklad/pokecrt/internal/sprite"
	"github.com/mayanklad/pokecrt/internal/storage"
	"github.com/mayanklad/pokecrt/internal/trainer"
	"strings"
)

type activitySnapshot struct {
	profile      trainer.Profile
	stats        trainer.Statistics
	achievements trainer.AchievementViews
	history      []storage.HistoryEntry
}
type activityLoaded struct {
	id   uint64
	data activitySnapshot
	err  error
}
type encounterFinished struct {
	trainerID int64
	id        uint64
	result    trainer.EncounterResult
	err       error
}
type activityState struct {
	generation                          uint64
	loading, recording                  bool
	data                                activitySnapshot
	error                               string
	selected, scroll, artScroll, artPan int
	refreshAfter                        bool
	panelScroll                         [2]int
	informationPanel                    int
	historyMode, resultDetails          bool
	result                              *trainer.Record
	art                                 []string
}
type activityLoader func(context.Context) (activitySnapshot, error)
type encounterAction func(context.Context, bool, int64) (trainer.EncounterResult, error)

func loadActivity(ctx context.Context) (activitySnapshot, error) {
	var out activitySnapshot
	path, err := storage.ResolvePath()
	if err != nil {
		return out, err
	}
	r, err := storage.ReadOnly(ctx, path)
	if err != nil {
		return out, err
	}
	defer r.Close()
	out.profile, err = r.ActiveProfile(ctx)
	if err != nil {
		return out, err
	}
	records, err := r.TrainerRecords(ctx, out.profile.ID)
	if err != nil {
		return out, err
	}
	available := func(k catalog.VariantKey) bool { _, ok := sprite.Lookup(k); return ok }
	out.stats, err = trainer.TrainerStatistics(records, catalog.All(), available)
	if err != nil {
		return out, err
	}
	targets, err := trainer.NewAchievementTargets(catalog.All(), available)
	if err != nil {
		return out, err
	}
	out.achievements = targets.Views(records.AchievementState, records.Unlocks)
	out.history, err = r.RecentEncounters(ctx, out.profile.ID)
	return out, err
}

// The captured trainer prevents a concurrent profile switch from redirecting
// the explicit action. Preparation happens before the atomic domain commit.
type capturedEncounterRepo struct {
	*storage.Repository
	profile trainer.Profile
}

func (r capturedEncounterRepo) ActiveProfile(context.Context) (trainer.Profile, error) {
	return r.profile, nil
}
func recordActivity(ctx context.Context, noColor bool, trainerID int64) (trainer.EncounterResult, error) {
	path, err := storage.ResolvePath()
	if err != nil {
		return trainer.EncounterResult{}, err
	}
	// Check existing current-schema storage before opening for writing.
	check, err := storage.ReadOnly(ctx, path)
	if err != nil {
		return trainer.EncounterResult{}, err
	}
	profiles, err := check.ListProfiles(ctx)
	check.Close()
	if err != nil {
		return trainer.EncounterResult{}, err
	}
	var p trainer.Profile
	for _, candidate := range profiles {
		if candidate.ID == trainerID {
			p = candidate
		}
	}
	if p.ID == 0 {
		return trainer.EncounterResult{}, trainer.ErrNotFound
	}
	r, err := storage.Open(ctx, path)
	if err != nil {
		return trainer.EncounterResult{}, err
	}
	defer r.Close()
	service, err := trainer.NewBundledEncounterService(capturedEncounterRepo{r, p}, func(c trainer.Choice) ([]byte, error) {
		pixels, err := sprite.Decode(c.Key())
		if err != nil {
			return nil, err
		}
		return render.Render(pixels, !noColor)
	})
	if err != nil {
		return trainer.EncounterResult{}, err
	}
	return service.Encounter(ctx)
}
func (m *Model) openActivity(section int) tea.Cmd {
	id := m.activity.generation + 1
	m.activity = activityState{generation: id, recording: m.encounterPending}
	m.section = section
	m.screen = activityScreen
	m.focus = 10
	if section == 2 {
		m.focus = 15
	}
	if section == 3 {
		m.focus = 12
		if m.dexGeometry().wide {
			m.focus = 40
		}
	}
	return tea.Batch(tea.ClearScreen, m.reloadActivity())
}
func (m *Model) reloadActivity() tea.Cmd {
	if m.activity.loading || m.activityLoader == nil {
		return nil
	}
	m.activity.generation++
	m.activity.loading = true
	m.activity.error = ""
	id, load, ctx := m.activity.generation, m.activityLoader, m.ctx
	return func() tea.Msg { data, err := load(ctx); return activityLoaded{id, data, err} }
}
func (m *Model) activateActivity(id int) tea.Cmd {
	m.focus = id
	switch {
	case id == 10:
		if m.section != 1 || m.encounterPending || m.activity.loading || m.activity.data.profile.ID == 0 || m.encounterAction == nil {
			return nil
		}
		m.activity.recording = true
		m.encounterNotice = ""
		m.encounterPending = true
		m.encounterID++
		m.activity.error = ""
		m.activity.result = nil
		m.activity.art = nil
		generation, action, ctx, noColor, trainerID := m.encounterID, m.encounterAction, m.ctx, m.noColor, m.activity.data.profile.ID
		return func() tea.Msg {
			result, err := action(ctx, noColor, trainerID)
			return encounterFinished{trainerID: trainerID, id: generation, result: result, err: err}
		}
	case id == 11:
		return m.reloadActivity()
	case id == 12:
		m.activity.generation++
		m.screen = mainScreen
		m.focus = m.section
		return tea.Batch(tea.ClearScreen, m.reload())
	case id == 13:
		m.openSettings()
		return tea.ClearScreen
	case id == 14:
		return tea.Quit
	case id == 15:
		m.openProfiles()
		return tea.Batch(tea.ClearScreen, m.reload())
	case id == 30:
		m.activity.historyMode = !m.activity.historyMode
		m.activity.scroll = 0
		return tea.ClearScreen
	case id == 31:
		m.activity.selected = max(0, m.activity.selected-1)
		m.revealHistorySelection()
	case id == 32:
		m.activity.selected = min(max(0, len(m.activity.data.history)-1), m.activity.selected+1)
		m.revealHistorySelection()
	case id == 33:
		m.activity.artScroll = max(0, m.activity.artScroll-1)
	case id == 34:
		m.activity.artScroll = min(max(0, len(m.activity.art)-m.activityBodyHeight()+2), m.activity.artScroll+1)
	case id == 35:
		m.activity.resultDetails = !m.activity.resultDetails
		return tea.ClearScreen
	case id == 40 || id == 41:
		m.activity.informationPanel = id - 40
	case id >= 42 && id <= 45:
		panel := (id - 42) / 2
		delta := -1
		if id%2 == 1 {
			delta = 1
		}
		m.activity.informationPanel = panel
		m.activity.panelScroll[panel] = max(0, min(m.activityPanelMaxScroll(panel), m.activity.panelScroll[panel]+delta))
	case id == 16:
		m.activity.scroll = max(0, m.activity.scroll-1)
	case id == 17:
		m.activity.scroll = min(m.activityMaxScroll(), m.activity.scroll+1)
	case id == 18:
		m.activity.artPan = max(0, m.activity.artPan-4)
	case id == 19:
		m.activity.artPan = min(m.activityMaxPan(), m.activity.artPan+4)
	case id >= 1000 && id < 1050:
		m.activity.selected = id - 1000
		m.focus = 20
	case id == 20:
		if m.section == 1 && len(m.activity.data.history) > 0 {
			h := m.activity.data.history[min(m.activity.selected, len(m.activity.data.history)-1)]
			cmd := m.openDex()
			m.dex.pendingKey = &h.Key
			m.dex.selection = catalog.Selection{Form: h.Key.FormID, Gender: h.Key.Gender, Shiny: h.Key.Palette == "shiny"}
			m.dex.detail = true
			return tea.Batch(tea.ClearScreen, cmd)
		}
	}
	return nil
}
func (m *Model) handleActivity(msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case activityLoaded:
		if msg.id != m.activity.generation {
			return nil, true
		}
		m.activity.loading = false
		if msg.err != nil {
			m.activity.error = clean(msg.err.Error())
		} else {
			if m.activity.result != nil && m.activity.result.TrainerID != msg.data.profile.ID {
				m.activity.result = nil
				m.activity.art = nil
			}
			if m.activity.data.profile.ID != msg.data.profile.ID {
				m.activity.selected = 0
				m.activity.scroll = 0
				m.activity.panelScroll = [2]int{}
			}
			m.activity.data = msg.data
			m.activity.selected = min(m.activity.selected, max(0, len(msg.data.history)-1))
			m.activity.scroll = min(m.activity.scroll, m.activityMaxScroll())
			for panel := range m.activity.panelScroll {
				m.activity.panelScroll[panel] = min(m.activity.panelScroll[panel], m.activityPanelMaxScroll(panel))
			}
		}
		if m.activity.refreshAfter {
			m.activity.refreshAfter = false
			return m.reloadActivity(), true
		}
		return nil, true
	case encounterFinished:
		if msg.id != m.encounterID {
			return nil, true
		}
		m.encounterPending = false
		m.activity.recording = false
		if msg.err != nil {
			m.encounterNotice = "Encounter failed. Open Encounters to retry."
			if m.activity.data.profile.ID == msg.trainerID {
				m.activity.error = clean(msg.err.Error())
			}
			return tea.ClearScreen, true
		}
		m.encounterNotice = "Encounter saved."
		record := msg.result.Record
		if m.screen == activityScreen && m.section == 1 && m.activity.data.profile.ID == record.TrainerID {
			m.activity.result = &record
			m.activity.art = strings.Split(strings.TrimRight(string(msg.result.Artwork()), "\n"), "\n")
			m.activity.artScroll = 0
			m.activity.artPan = 0
			m.activity.selected = 0
			m.activity.scroll = 0
			m.activity.historyMode = false
		}
		var read tea.Cmd
		if m.screen == activityScreen {
			if m.activity.loading {
				m.activity.refreshAfter = true
			} else {
				read = m.reloadActivity()
			}
		}
		return tea.Batch(tea.ClearScreen, read, m.reload()), true
	}
	if m.settings || m.screen != activityScreen {
		return nil, false
	}
	controlsNow := m.activityControls()
	found := false
	for _, c := range controlsNow {
		if c.id == m.focus {
			found = true
		}
	}
	if !found && len(controlsNow) > 0 {
		m.focus = controlsNow[0].id
	}
	if m.section != 1 && m.dexGeometry().wide {
		if m.focus == 40 || m.focus == 41 {
			m.activity.informationPanel = m.focus - 40
		}
		if m.focus >= 42 && m.focus <= 45 {
			m.activity.informationPanel = (m.focus - 42) / 2
		}
	}
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return tea.Quit, true
		case "esc":
			return m.activateActivity(12), true
		case "a":
			m.openSettings()
			return tea.ClearScreen, true
		case "r":
			return m.reloadActivity(), true
		case "enter", "space":
			if msg.IsRepeat {
				return nil, true
			}
			return m.activateActivity(m.focus), true
		case "tab", "shift+tab":
			controls := m.activityControls()
			index := 0
			for i, c := range controls {
				if c.id == m.focus {
					index = i
				}
			}
			delta := 1
			if msg.String() == "shift+tab" {
				delta = -1
			}
			m.focus = controls[(index+delta+len(controls))%len(controls)].id
		case "up", "down", "j", "k", "left", "right":
			delta := 1
			if msg.String() == "up" || msg.String() == "k" || msg.String() == "left" {
				delta = -1
			}
			if m.section == 3 && (msg.String() == "left" || msg.String() == "right") {
				m.moveAchievementControl(delta)
			} else if m.section != 1 && m.dexGeometry().wide && (m.focus == 40 || m.focus == 41) {
				panel := m.focus - 40
				m.activity.informationPanel = panel
				if msg.String() == "left" || msg.String() == "right" {
					if panel == 0 && msg.String() == "left" {
						m.focus = 12
					} else if panel == 1 && msg.String() == "right" {
						m.focus = 14
					} else {
						m.focus = 40 + 1 - panel
						m.activity.informationPanel = 1 - panel
					}
				} else {
					next := max(0, min(m.activityPanelMaxScroll(panel), m.activity.panelScroll[panel]+delta))
					if next == m.activity.panelScroll[panel] {
						m.activityMove(msg.String())
					} else {
						m.activity.panelScroll[panel] = next
					}
				}
			} else if m.section == 1 && m.activity.historyMode && m.focus == 20 && (msg.String() == "up" || msg.String() == "down" || msg.String() == "j" || msg.String() == "k") {
				next := max(0, min(max(0, len(m.activity.data.history)-1), m.activity.selected+delta))
				if next == m.activity.selected {
					m.activityMove(msg.String())
				} else {
					m.activity.selected = next
					m.revealHistorySelection()
				}
			} else if m.focus == 21 {
				if msg.String() == "left" || msg.String() == "right" {
					next := max(0, min(m.activityMaxPan(), m.activity.artPan+delta*4))
					if next == m.activity.artPan {
						m.activityMove(msg.String())
					} else {
						m.activity.artPan = next
					}
				} else {
					next := max(0, min(max(0, len(m.activity.art)-m.activityBodyHeight()+2), m.activity.artScroll+delta))
					if next == m.activity.artScroll {
						m.activityMove(msg.String())
					} else {
						m.activity.artScroll = next
					}
				}
			} else if m.focus == 22 && (msg.String() == "left" || msg.String() == "right") {
				m.activityMove(msg.String())
			} else if m.focus == 22 {
				next := max(0, min(m.activityMaxScroll(), m.activity.scroll+delta))
				if next == m.activity.scroll {
					m.activityMove(msg.String())
				} else {
					m.activity.scroll = next
				}
			} else {
				m.activityMove(msg.String())
			}
		case "pgdown":
			if m.section != 1 && m.dexGeometry().wide {
				m.scrollActivityPanel(m.activity.informationPanel, m.activityBodyHeight()-2)
			} else {
				m.activity.scroll = min(m.activityMaxScroll(), m.activity.scroll+m.activityBodyHeight()-2)
			}
		case "pgup":
			if m.section != 1 && m.dexGeometry().wide {
				m.scrollActivityPanel(m.activity.informationPanel, -m.activityBodyHeight()+2)
			} else {
				m.activity.scroll = max(0, m.activity.scroll-m.activityBodyHeight()+2)
			}
		}
		return nil, true
	case tea.MouseWheelMsg:
		if !m.dexWide() {
			delta := 1
			if msg.Button == tea.MouseWheelUp || msg.Button == tea.MouseWheelLeft {
				delta = -1
			}
			artView := m.section == 1 && !m.activity.historyMode && len(m.activity.art) > 0 && !m.activity.resultDetails && m.activity.error == ""
			if artView {
				if msg.Button == tea.MouseWheelLeft || msg.Button == tea.MouseWheelRight {
					m.activity.artPan = max(0, min(m.activityMaxPan(), m.activity.artPan+4*delta))
				} else {
					m.activity.artScroll = max(0, min(max(0, len(m.activity.art)-m.activityBodyHeight()+2), m.activity.artScroll+delta))
				}
			} else if m.section == 1 && m.activity.historyMode && len(m.activity.data.history) > 0 {
				m.activity.selected = max(0, min(len(m.activity.data.history)-1, m.activity.selected+delta))
				m.revealHistorySelection()
			} else {
				m.activity.scroll = max(0, min(m.activityMaxScroll(), m.activity.scroll+delta))
			}
			return nil, true
		}
		delta := 1
		if msg.Button == tea.MouseWheelUp || msg.Button == tea.MouseWheelLeft {
			delta = -1
		}
		if m.section != 1 && m.dexGeometry().wide {
			panel := 0
			if msg.X >= m.dexGeometry().w/2 {
				panel = 1
			}
			m.focus = 40 + panel
			m.activity.informationPanel = panel
			m.scrollActivityPanel(panel, delta)
		} else {
			m.activity.scroll = max(0, min(m.activityMaxScroll(), m.activity.scroll+delta))
		}
		return nil, true
	}
	return nil, false
}
func (m *Model) activityMove(key string) {
	if key == "k" {
		key = "up"
	}
	if key == "j" {
		key = "down"
	}
	m.focus = directionalTarget(m.activityControls(), m.focus, key)
}

func (m *Model) revealHistorySelection() {
	for row, owner := range m.activityHistoryTargets() {
		if owner == m.activity.selected {
			m.activity.scroll = min(m.activityMaxScroll(), row)
			return
		}
	}
}

func (m Model) activityPanelMaxScroll(panel int) int {
	left, right := m.activityColumns()
	lines := left
	if panel == 1 {
		lines = right
	}
	width := m.dexGeometry().w/2 - 7
	if panel == 1 {
		width++
	}
	return max(0, len(wrapActivityLines(lines, width))-m.activityBodyHeight()+2)
}
func (m *Model) scrollActivityPanel(panel, delta int) {
	m.activity.panelScroll[panel] = max(0, min(m.activityPanelMaxScroll(panel), m.activity.panelScroll[panel]+delta))
}

// Horizontal movement follows the visible panel controls, then footer actions.
func (m *Model) moveAchievementControl(delta int) {
	controls := m.activityControls()
	for i, control := range controls {
		if control.id == m.focus {
			next := (i + delta + len(controls)) % len(controls)
			m.focus = controls[next].id
			return
		}
	}
}

func (m Model) achievementPanelStyle(panel int) string {
	if m.noColor {
		return ""
	}
	active := m.focus == 22 || m.focus == 16 || m.focus == 17
	if m.dexGeometry().wide {
		active = m.focus == 40+panel || m.focus == 42+panel*2 || m.focus == 43+panel*2
	}
	if active {
		return m.palette().muted + "\x1b[1m"
	}
	return m.palette().accent
}

func (m Model) paintAchievementHints(c *canvas, x, y, w int) {
	first := "[↑↓←→] Move  [Tab] Focus"
	if m.focus == 22 || m.focus == 40 || m.focus == 41 {
		first = "[↑↓] Scroll  [←→] Move  [Tab] Focus"
	}
	second := "[Enter] Select  [Esc] Back  [Q] Quit"
	full := first + "  " + second
	y = c.height - 3
	if ansi.StringWidth(full) <= w {
		y = c.height - 2
	}
	if y > 0 {
		c.put(0, y-1, "├"+strings.Repeat("─", max(0, c.width-2))+"┤", m.footerDividerStyle())
	}
	if ansi.StringWidth(full) <= w {
		c.put(x, y, full, m.hintStyle())
		return
	}
	c.put(x, y, first, m.hintStyle())
	c.put(x, y+1, second, m.hintStyle())
}
