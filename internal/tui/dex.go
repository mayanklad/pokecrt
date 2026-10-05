package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/mayanklad/pokecrt/internal/catalog"
	"github.com/mayanklad/pokecrt/internal/render"
	"github.com/mayanklad/pokecrt/internal/sprite"
	"github.com/mayanklad/pokecrt/internal/storage"
	"github.com/mayanklad/pokecrt/internal/trainer"
)

type dexSnapshot struct {
	trainerID int64
	name      string
	data      *trainer.Dex
	summary   trainer.DexSummary
}
type dexLoader func(context.Context) (dexSnapshot, error)
type dexLoadedMsg struct {
	id       uint64
	snapshot dexSnapshot
	err      error
}
type dexEntryMsg struct {
	id      uint64
	entry   trainer.DexEntry
	options []trainer.DexAppearance
	art     string
	err     error
}
type dexState struct {
	pendingKey                         *catalog.VariantKey
	generation, entryGeneration        uint64
	cancel                             context.CancelFunc
	snapshot                           dexSnapshot
	loading, entryLoading              bool
	error, entryError                  string
	rows                               []trainer.DexRow
	selected, status, generationFilter int
	query                              string
	detail                             bool
	tab                                int
	filters                            bool
	entry                              trainer.DexEntry
	options                            []trainer.DexAppearance
	optionIndex                        int
	selection                          catalog.Selection
	art                                string
	artRows                            []string
	artWidth                           int
	scroll, horizontal                 int
	searchBefore                       []rune
	searchCursor                       int
}

func loadDex(ctx context.Context) (dexSnapshot, error) {
	path, err := storage.ResolvePath()
	if err != nil {
		return dexSnapshot{}, err
	}
	repo, err := storage.ReadOnly(ctx, path)
	if err != nil {
		return dexSnapshot{}, err
	}
	defer repo.Close()
	p, err := repo.ActiveProfile(ctx)
	if err != nil {
		return dexSnapshot{}, err
	}
	records, err := repo.DexRecords(ctx, p.ID)
	if err != nil {
		return dexSnapshot{}, err
	}
	if err = ctx.Err(); err != nil {
		return dexSnapshot{}, err
	}
	d := trainer.NewDex(catalog.All(), records, func(k catalog.VariantKey) bool { _, ok := sprite.Lookup(k); return ok })
	return dexSnapshot{p.ID, p.Name, d, d.Summary()}, ctx.Err()
}
func (m *Model) openDex() tea.Cmd {
	if m.dex.cancel != nil {
		m.dex.cancel()
	}
	next := m.dex.generation + 1
	entry := m.dex.entryGeneration + 1
	m.dex = dexState{generation: next, entryGeneration: entry}
	m.screen = dexScreen
	m.focus = 0
	return m.reloadDex()
}
func (m *Model) reloadDex() tea.Cmd {
	if m.dexLoader == nil || m.dex.loading {
		return nil
	}
	if m.dex.cancel != nil {
		m.dex.cancel()
	}
	m.dex.generation++
	m.dex.entryGeneration++
	m.dex.entryLoading = false
	m.dex.loading = true
	m.dex.error = ""
	m.dex.snapshot = dexSnapshot{}
	m.dex.rows = nil
	m.dex.entry = trainer.DexEntry{}
	m.dex.options = nil
	m.dex.art = ""
	m.dex.artRows = nil
	m.dex.artWidth = 0
	ctx, cancel := context.WithCancel(m.ctx)
	m.dex.cancel = cancel
	id, load := m.dex.generation, m.dexLoader
	return func() tea.Msg { s, err := load(ctx); return dexLoadedMsg{id, s, err} }
}
func (m *Model) leaveDex() {
	if m.dex.cancel != nil {
		m.dex.cancel()
	}
	m.dex.generation++
	m.dex.entryGeneration++
	m.screen = mainScreen
	m.focus = 0
}
func (m *Model) filterDex() tea.Cmd {
	if m.dex.snapshot.data == nil {
		return nil
	}
	number := 0
	if len(m.dex.rows) > 0 {
		number = m.dex.rows[min(m.dex.selected, len(m.dex.rows)-1)].Number
	}
	f := trainer.DexFilter{Seen: m.dex.status == 1, Unseen: m.dex.status == 2}
	if m.dex.generationFilter > 0 {
		f.Selection.Generations = []int{m.dex.generationFilter}
	}
	rows := m.dex.snapshot.data.List(f)
	query := strings.ToLower(strings.TrimSpace(m.dex.query))
	query = strings.TrimPrefix(query, "#")
	m.dex.rows = nil
	for _, r := range rows {
		if query == "" || strings.Contains(strconv.Itoa(r.Number), query) || strings.Contains(fmt.Sprintf("%03d", r.Number), query) || r.Seen && strings.Contains(strings.ToLower(r.Name), query) {
			m.dex.rows = append(m.dex.rows, r)
		}
	}
	m.dex.selected = 0
	for i, r := range m.dex.rows {
		if r.Number == number {
			m.dex.selected = i
			break
		}
	}
	m.dex.detail = false
	m.dex.filters = false
	if len(m.dex.rows) > 0 && m.dex.rows[m.dex.selected].Number == number {
		return m.loadDexEntry(m.dex.selection)
	}
	return m.loadDexEntry(catalog.Selection{})
}
func (m *Model) loadDexEntry(q catalog.Selection) tea.Cmd {
	m.dex.entryGeneration++
	id := m.dex.entryGeneration
	m.dex.entry = trainer.DexEntry{}
	m.dex.art = ""
	m.dex.artRows = nil
	m.dex.artWidth = 0
	m.dex.options = nil
	m.dex.entryError = ""
	m.dex.scroll = 0
	m.dex.horizontal = 0
	m.dex.selection = q
	if len(m.dex.rows) == 0 || m.dex.snapshot.data == nil {
		m.dex.entryLoading = false
		return nil
	}
	m.dex.entryLoading = true
	number := m.dex.rows[m.dex.selected].Number
	d := m.dex.snapshot.data
	colored := !m.noColor
	return func() tea.Msg {
		e, err := d.Entry(number, q)
		opts := d.Appearances(number)
		var art []byte
		if err == nil && e.ArtworkKey != nil {
			pixels, decodeErr := sprite.Decode(*e.ArtworkKey)
			if decodeErr != nil {
				err = decodeErr
			} else {
				art, err = render.Render(pixels, colored)
			}
		}
		return dexEntryMsg{id, e, opts, string(art), err}
	}
}
func (m *Model) selectDex(delta int) tea.Cmd {
	if len(m.dex.rows) == 0 {
		return nil
	}
	next := max(0, min(m.dex.selected+delta, len(m.dex.rows)-1))
	if next == m.dex.selected {
		return nil
	}
	m.dex.selected = next
	return m.loadDexEntry(catalog.Selection{})
}
func (m *Model) openDexSearch() {
	m.dex.searchBefore = append([]rune(nil), m.name...)
	m.dex.searchCursor = m.cursor
	m.name = []rune(m.dex.query)
	m.cursor = len(m.name)
	m.formError = ""
	m.screen = dexSearchScreen
	m.focus = 0
}
func (m *Model) closeDexSearch(apply bool) tea.Cmd {
	if apply {
		m.dex.query = string(m.name)
	}
	m.name = m.dex.searchBefore
	m.cursor = m.dex.searchCursor
	m.formError = ""
	m.screen = dexScreen
	m.focus = 1
	if apply {
		m.focus = 0
		return m.filterDex()
	}
	return nil
}
func (m *Model) moveDexFocus(delta int) {
	order := m.dexFocusOrder()
	for i, id := range order {
		if id == m.focus {
			m.focus = order[(i+delta+len(order))%len(order)]
			m.revealDexFocus()
			return
		}
	}
	m.focus = order[0]
}
func (m Model) dexWide() bool { return m.width >= 100 && m.height >= 24 }
func (m *Model) activateDex(id int) tea.Cmd {
	if m.screen == dexSearchScreen {
		if id >= 100 && id < 100+len(m.keyboardLetters()) {
			m.focus = id
			m.insertName(string(m.keyboardLetters()[id-100]))
			return nil
		}
		if id < 0 || id > 7 {
			return nil
		}
		m.focus = id
		switch id {
		case 0:
			return nil
		case 1:
			return m.closeDexSearch(true)
		case 2:
			return m.closeDexSearch(false)
		case 3:
			m.keyboardPage = (m.keyboardPage + 1) % 3
		case 4:
			return m.editName(tea.KeyPressMsg{Code: tea.KeyBackspace})
		case 5:
			m.insertName(" ")
		case 6:
			m.openSettings()
		case 7:
			return tea.Quit
		}
		return nil
	}

	if id >= 3000 {
		index := id - 3000
		if index >= len(m.dex.options) {
			return nil
		}
		m.dex.optionIndex = index
		q := m.dex.options[index].Selection
		m.dex.detail = true
		m.dex.tab = 0
		m.focus = 10
		return m.loadDexEntry(q)
	}
	if id >= 2000 {
		number := id - 2000
		m.dex.status = 0
		m.dex.generationFilter = 0
		m.dex.query = ""
		m.filterDex()
		for i, r := range m.dex.rows {
			if r.Number == number {
				m.dex.selected = i
				break
			}
		}
		m.dex.detail = true
		m.dex.tab = 0
		m.focus = 10
		return m.loadDexEntry(catalog.Selection{})
	}
	if id >= 1000 {
		index := id - 1000
		if index >= len(m.dex.rows) {
			return nil
		}
		m.dex.selected = index
		m.focus = 0
		return m.loadDexEntry(catalog.Selection{})
	}
	m.focus = id
	switch id {
	case 0:
		m.dex.detail = true
		m.focus = m.dexContentFocus()
	case 4:
		if m.dex.detail && !m.dexWide() {
			m.dex.detail = false
			m.focus = 0
		} else {
			m.dex.detail = true
			m.focus = m.dexContentFocus()
		}
	case 1:
		m.openDexSearch()
	case 2:
		m.dex.status = (m.dex.status + 1) % 3
		return m.filterDex()
	case 31, 32, 33:
		m.dex.status = id - 31
		return m.filterDex()
	case 3:
		m.dex.generationFilter = (m.dex.generationFilter + 1) % 10
		return m.filterDex()
	case 5:
		return m.reloadDex()
	case 6:
		m.leaveDex()
	case 7:
		m.openSettings()
	case 8:
		return tea.Quit
	case 9:
		m.setDexTab(1)
	case 20, 21, 22, 23:
		m.setDexTab(id - 20)
	case 24:
		m.setDexTab((m.dex.tab + 1) % 4)
		m.focus = m.dexContentFocus()
		m.focus = 24
	case 25:
		m.dex.filters = !m.dex.filters
		m.focus = 1
	case 16:
		return m.selectDex(-max(1, m.dexBodyHeight()))
	case 17:
		return m.selectDex(max(1, m.dexBodyHeight()))
	case 15:
		if len(m.dex.options) > 0 {
			return m.activateDex(3000 + m.dex.optionIndex)
		}
	case 11:
		m.dex.detail = true
		m.dex.scroll = max(0, m.dex.scroll-3)
	case 12:
		m.dex.detail = true
		m.dex.scroll = min(m.dex.scroll+3, m.dexMaxScroll())
	case 13:
		m.dex.detail = true
		m.dex.horizontal = max(0, m.dex.horizontal-5)
	case 14:
		m.dex.detail = true
		m.dex.horizontal = min(m.dex.horizontal+5, m.dexMaxHorizontal())
	}
	return nil
}
func (m Model) dexContentFocus() int {
	if m.dex.tab == 1 {
		return 15
	}
	return 10
}
func (m *Model) setDexTab(tab int) {
	m.screen = dexScreen
	m.dex.tab = tab
	m.dex.detail = true
	m.dex.scroll = 0
	m.dex.horizontal = 0
	m.dex.optionIndex = 0
}
func (m *Model) handleDex(msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case dexLoadedMsg:
		if msg.id != m.dex.generation || !m.dex.loading {
			return nil, true
		}
		m.dex.loading = false
		m.dex.error = ""
		if msg.err != nil {
			m.dex.error = clean(msg.err.Error())
			return nil, true
		}
		m.dex.snapshot = msg.snapshot
		if key := m.dex.pendingKey; key != nil {
			m.dex.rows = msg.snapshot.data.List(trainer.DexFilter{})
			for i, row := range m.dex.rows {
				if row.Number == key.SpeciesID {
					m.dex.selected = i
					break
				}
			}
			m.dex.detail = true
			m.focus = 10
			m.dex.pendingKey = nil
			return m.loadDexEntry(catalog.Selection{Form: key.FormID, Gender: key.Gender, Shiny: key.Palette == "shiny"}), true
		}
		return m.filterDex(), true
	case dexEntryMsg:
		if msg.id != m.dex.entryGeneration || !m.dex.entryLoading {
			return nil, true
		}
		m.dex.entryLoading = false
		m.dex.entry = msg.entry
		m.dex.options = msg.options
		m.dex.art = msg.art
		if msg.art != "" {
			m.dex.artRows = strings.Split(strings.TrimSuffix(msg.art, "\n"), "\n")
			if msg.entry.ArtworkKey != nil {
				asset, _ := sprite.Lookup(*msg.entry.ArtworkKey)
				m.dex.artWidth = asset.Width
			}
		}
		if msg.err != nil {
			m.dex.entryError = clean(msg.err.Error())
			m.dex.art = ""
			m.dex.artRows = nil
			m.dex.artWidth = 0
		}
		if m.screen == dexScreen && !m.settings {
			return tea.ClearScreen, true
		}
		return nil, true
	}

	if m.settings || (m.screen != dexScreen && m.screen != dexSearchScreen) {
		return nil, false
	}

	switch msg := msg.(type) {
	case tea.PasteMsg:
		if m.screen == dexSearchScreen && m.focus == 0 {
			m.insertName(msg.Content)
		}
		return nil, true
	case tea.KeyPressMsg:
		k := msg.String()
		switch k {
		case "ctrl+c":
			return tea.Quit, true
		case "esc":
			if m.screen == dexSearchScreen {
				return m.closeDexSearch(false), true
			}
			if m.dex.filters {
				m.dex.filters = false
				m.focus = 25
			} else if m.dex.detail {
				m.dex.detail = false
				m.focus = 0
			} else {
				m.leaveDex()
			}
			return nil, true
		}
		if m.screen == dexSearchScreen {
			switch k {
			case "tab":
				m.moveFocus(1)
			case "shift+tab":
				m.moveFocus(-1)
			default:
				if m.focus == 0 {
					if k == "enter" {
						return m.closeDexSearch(true), true
					}
					return m.editName(msg), true
				}
				switch k {
				case "up", "down", "left", "right":
					m.navigateSetup(k)
				case "enter", "space":
					return m.activateDex(m.focus), true
				case "q":
					return tea.Quit, true
				case "a":
					m.openSettings()
				}
			}
			return nil, true
		}
		switch k {
		case "/":
			m.openDexSearch()
		case "q":
			return tea.Quit, true
		case "a":
			m.openSettings()
		case "r":
			return m.reloadDex(), true
		case "tab":
			m.moveDexFocus(1)
		case "shift+tab":
			m.moveDexFocus(-1)
		case "enter", "space":
			return m.activateDex(m.focus), true
		case "[":
			m.setDexTab((m.dex.tab + 3) % 4)
			m.focus = m.dexContentFocus()
		case "]":
			m.setDexTab((m.dex.tab + 1) % 4)
			m.focus = m.dexContentFocus()
		case "up", "down", "j", "k", "pgup", "pgdown":
			delta := 1
			if k == "up" || k == "k" || k == "pgup" {
				delta = -1
			}
			if k == "pgup" || k == "pgdown" {
				delta *= 10
			}
			if m.focus == 0 {
				if len(m.dex.rows) == 0 || abs(delta) == 1 && (m.dex.selected+delta < 0 || m.dex.selected+delta >= len(m.dex.rows)) {
					if delta < 0 {
						m.focus = 31
					} else {
						m.focus = 4
						if m.dexWide() {
							m.focus = 20
						}
					}
					m.ensureDexFocus()
					return nil, true
				}
				return m.selectDex(delta), true
			}
			if m.focus == 15 {
				next := m.dex.optionIndex + delta
				if next < 0 {
					m.focus = 1
				} else if next >= len(m.dex.options) {
					m.focus = m.dexTabFocus()
				} else {
					m.dex.optionIndex = next
					m.revealDexOption()
				}
				return nil, true
			}
			if m.focus >= 2000 && m.focus < 3000 {
				for i, n := range m.dex.entry.Evolution {
					if 2000+n.Number == m.focus {
						next := i + delta
						if next < 0 {
							m.focus = m.dexTabFocus()
						} else if next >= len(m.dex.entry.Evolution) {
							m.focus = 6
						} else {
							m.focus = 2000 + m.dex.entry.Evolution[next].Number
							m.revealDexFocus()
						}
						break
					}
				}
				return nil, true
			}
			if m.focus == 10 {
				next := m.dex.scroll + delta
				if next < 0 {
					m.focus = 1
				} else if next > m.dexMaxScroll() {
					m.focus = m.dexTabFocus()
				} else {
					m.dex.scroll = next
				}
				return nil, true
			}
			direction := "down"
			if delta < 0 {
				direction = "up"
			}
			m.navigateDexControl(direction)
		case "left", "right":
			if m.focus == 10 && m.dex.tab == 0 {
				delta := 5
				if k == "left" {
					delta = -5
				}
				if delta < 0 && m.dex.horizontal == 0 {
					m.focus = 4
					if m.dexWide() {
						m.focus = 0
					}
				} else {
					m.dex.horizontal = max(0, min(m.dex.horizontal+delta, m.dexMaxHorizontal()))
				}
			} else if m.focus == 0 && k == "right" {
				m.dex.detail = true
				m.focus = m.dexContentFocus()
			} else {
				m.navigateDexControl(k)
			}
		}
		return nil, true
	case tea.MouseWheelMsg:
		delta := 1
		if msg.Button == tea.MouseWheelUp || msg.Button == tea.MouseWheelLeft {
			delta = -1
		}
		if m.screen == dexSearchScreen {
			return nil, true
		}
		g := m.dexGeometry()
		inDetail := m.dex.detail || m.dexWide() && msg.X >= g.entryX
		if !inDetail {
			return m.selectDex(delta), true
		}
		if msg.Button == tea.MouseWheelLeft || msg.Button == tea.MouseWheelRight {
			m.dex.horizontal = max(0, min(m.dex.horizontal+5*delta, m.dexMaxHorizontal()))
		} else if m.dex.tab == 1 {
			m.dex.optionIndex = max(0, min(m.dex.optionIndex+delta, len(m.dex.options)-1))
			m.revealDexOption()
		} else {
			m.dex.scroll = max(0, min(m.dex.scroll+delta, m.dexMaxScroll()))
		}
		return nil, true
	}
	return nil, false
}
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
func (m *Model) revealDexFocus() {
	if m.focus < 2000 || m.focus >= 3000 {
		return
	}
	for i, line := range m.dexLines() {
		if line.target == m.focus {
			m.dex.scroll = max(0, min(i, m.dexMaxScroll()))
			break
		}
	}
}
func (m *Model) revealDexOption() {
	m.dex.scroll = max(0, min(m.dex.optionIndex-m.dexBodyHeight()/2, m.dexMaxScroll()))
}
func (m Model) dexTabFocus() int {
	if min(m.height, 40) < 20 {
		return 24
	}
	return 20 + m.dex.tab
}
func (m *Model) ensureDexFocus() {
	for _, id := range m.dexFocusOrder() {
		if id == m.focus {
			return
		}
	}
	m.focus = 1
}
func (m *Model) navigateDexControl(direction string) {
	if m.focus >= 20 && m.focus <= 23 && direction == "up" {
		if !m.dexWide() && m.focus >= 22 {
			m.focus -= 2
		} else {
			m.focus = m.dexContentFocus()
		}
		return
	}
	m.focus = directionalTarget(m.dexTargets(), m.focus, direction)
	m.revealDexFocus()
}

func absFloat(n float64) float64 {
	if n < 0 {
		return -n
	}
	return n
}
