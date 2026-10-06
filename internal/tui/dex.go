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
	family  map[int]string
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
	overviewFacts                      bool
	tab                                int
	filters                            bool
	entry                              trainer.DexEntry
	options                            []trainer.DexAppearance
	optionIndex                        int
	factsScroll                        int
	selection                          catalog.Selection
	art                                string
	artRows                            []string
	artWidth                           int
	family                             map[int]string
	familyOrigin, familyNode           int
	cardNode                           int
	familySelection                    catalog.Selection
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
	m.dex.familyOrigin = 0
	m.dex.snapshot = dexSnapshot{}
	m.dex.rows = nil
	m.dex.entry = trainer.DexEntry{}
	m.dex.options = nil
	m.dex.family = nil
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
	if len(m.dex.rows) == 0 {
		m.focus = 1
	}
	if len(m.dex.rows) > 0 && m.dex.rows[m.dex.selected].Number == number {
		return m.loadDexEntry(m.dex.selection)
	}
	return m.loadDexEntry(catalog.Selection{})
}
func (m *Model) loadDexEntry(q catalog.Selection) tea.Cmd {
	m.dex.entryGeneration++
	id := m.dex.entryGeneration
	m.dex.entry = trainer.DexEntry{}
	m.dex.family = nil
	m.dex.art = ""
	m.dex.artRows = nil
	m.dex.artWidth = 0
	m.dex.options = nil
	m.dex.entryError = ""
	m.dex.scroll = 0
	m.dex.factsScroll = 0
	m.dex.overviewFacts = false
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
		family := map[int]string{}
		if err == nil {
			for _, node := range e.Evolution {
				for _, option := range d.Appearances(node.Number) {
					if !option.Collected {
						continue
					}
					revealed, entryErr := d.Entry(node.Number, option.Selection)
					if entryErr != nil || revealed.ArtworkKey == nil {
						continue
					}
					key := *revealed.ArtworkKey
					pixels, decodeErr := sprite.Decode(key)
					if decodeErr != nil {
						continue
					}
					rendered, renderErr := render.Render(pixels, colored)
					if renderErr == nil {
						family[node.Number] = string(rendered)
						break
					}
				}
			}
		}
		return dexEntryMsg{id: id, entry: e, options: opts, art: string(art), family: family, err: err}
	}
}
func (m *Model) selectDex(delta int) tea.Cmd {
	m.dex.familyOrigin = 0
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
	if id == 90 && !m.dexWide() {
		return m.openCompactMessage()
	}
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
		if m.dex.tab == 2 {
			m.dex.familyOrigin = m.dex.entry.Number
			m.dex.familyNode = id
			m.dex.familySelection = m.dex.selection
		}
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
		m.dex.familyOrigin = 0
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
		if m.dex.familyOrigin > 0 {
			return m.returnToFamily()
		}
		if !m.dexWide() && m.dex.filters {
			m.dex.filters = false
			m.focus = 25
			return nil
		}
		if !m.dexWide() && m.dex.detail {
			m.dex.detail = false
			m.focus = 0
			return nil
		}
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
	case 26:
		m.dex.overviewFacts = !m.dex.overviewFacts
		m.dex.scroll, m.dex.horizontal = 0, 0
		m.focus = 26
	case 27:
		return m.openFamilyArtwork()
	case 25:
		m.dex.filters = !m.dex.filters
		m.focus = 25
	case 16:
		return m.selectDex(-max(1, m.dexIndexHeight()))
	case 17:
		return m.selectDex(max(1, m.dexIndexHeight()))
	case 40:
		m.focus = 40
	case 41:
		m.dex.factsScroll = max(0, m.dex.factsScroll-3)
	case 42:
		m.dex.factsScroll = min(m.dexFactsMaxScroll(), m.dex.factsScroll+3)
	case 18:
		if m.dex.tab == 1 && len(m.dex.options) > 0 {
			return m.activateDex(3000 + m.dex.optionIndex)
		}
		if m.dex.tab == 2 && m.dex.cardNode >= 2000 {
			return m.activateDex(m.dex.cardNode)
		}
		m.focus = 18
	case 19:
		m.focus = 19
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
	m.ensureDexFocus()
	m.revealDexFocus()
	return nil
}
func (m Model) dexContentFocus() int {
	if m.dexGeometry().bodyH < 3 {
		return m.dexTabFocus()
	}
	if m.dex.tab == 1 && len(m.dex.options) > 0 && m.dex.entry.Seen {
		return 15
	}
	if m.dex.tab == 2 && m.dex.entry.Seen && len(m.dex.entry.Evolution) > 0 {
		return 2000 + orderedEvolution(m.dex.entry.Evolution)[0].Number
	}
	if m.dexMaxScroll() > 0 || m.dexMaxHorizontal() > 0 {
		return 18
	}
	return m.dexTabFocus()
}
func (m *Model) setDexTab(tab int) {
	m.screen = dexScreen
	m.dex.tab = tab
	m.dex.detail = true
	m.dex.scroll = 0
	m.dex.horizontal = 0
	m.dex.optionIndex = 0
	if tab == 1 {
		m.revealDexOption()
	}
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
			m.focus = 5
			m.ensureDexFocus()
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
		m.dex.family = msg.family
		if m.dex.tab == 2 {
			m.revealDexFocus()
		}
		if msg.art != "" {
			m.dex.artRows = strings.Split(strings.TrimSuffix(msg.art, "\n"), "\n")
			if msg.entry.ArtworkKey != nil {
				asset, _ := sprite.Lookup(*msg.entry.ArtworkKey)
				m.dex.artWidth = asset.Width
			}
		}
		if msg.err != nil {
			m.dex.entryError = clean(msg.err.Error())
			m.dex.family = nil
			m.dex.art = ""
			m.dex.artRows = nil
			m.dex.artWidth = 0
		}
		if m.screen == dexScreen && !m.settings {
			m.ensureDexFocus()
			m.revealDexFocus()
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
		if m.dex.tab == 2 && (k == "pgup" || k == "pgdown") && m.focus >= 2000 && m.focus < 3000 {
			delta := m.dexBodyHeight()
			if k == "pgup" {
				delta = -delta
			}
			m.dex.scroll = max(0, min(m.dexMaxScroll(), m.dex.scroll+delta))
			return nil, true
		}
		if m.dex.tab == 2 && (k == "up" || k == "down" || k == "left" || k == "right" || k == "j" || k == "k") && (m.focus == 10 || m.focus >= 2000 && m.focus < 3000) {
			m.navigateEvolution(k)
			return nil, true
		}
		switch k {
		case "ctrl+c":
			return tea.Quit, true
		case "esc":
			if m.dex.familyOrigin > 0 && m.screen == dexScreen {
				return m.returnToFamily(), true
			}
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
				delta *= m.pokedexPageSize(m.focus)
			}
			if m.focus == 40 {
				next := m.dex.factsScroll + delta
				if next < 0 {
					m.focus = 1
				} else if next > m.dexFactsMaxScroll() {
					m.focus = m.dexTabFocus()
				} else {
					m.dex.factsScroll = next
				}
				return nil, true
			}
			if m.focus == 0 || m.focus == 19 {
				if len(m.dex.rows) == 0 || abs(delta) == 1 && (m.dex.selected+delta < 0 || m.dex.selected+delta >= len(m.dex.rows)) {
					if delta < 0 {
						m.focus = 31
						if m.width < 64 {
							m.focus = 2
						}
					} else {
						wasScroll := m.focus == 19
						m.focus = 6
						for _, control := range m.dexControls() {
							if control.id == 19 {
								m.focus = 19
								break
							}
						}
						if wasScroll {
							m.pokedexTabsOrFooter(m.pokedexAnchorX(19))
						}
					}
					m.ensureDexFocus()
					return nil, true
				}
				return m.selectDex(delta), true
			}
			if m.focus == 15 || m.focus == 18 && m.dex.tab == 1 && m.dex.entry.Seen {
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
			if m.focus == 18 && m.dex.tab == 2 && m.dex.entry.Seen && len(m.dex.entry.Evolution) > 0 {
				nodes := orderedEvolution(m.dex.entry.Evolution)
				current := 0
				for i, n := range nodes {
					if 2000+n.Number == m.dex.cardNode {
						current = i
					}
				}
				next := current + delta
				if next < 0 {
					m.focus = 1
				} else if next >= len(nodes) {
					m.pokedexTabsOrFooter(m.pokedexAnchorX(18))
				} else {
					m.focus = 2000 + nodes[next].Number
					m.revealDexFocus()
					m.focus = 18
				}
				return nil, true
			}
			if m.focus == 10 || m.focus == 18 {
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
			if (m.focus == 10 || m.focus == 18) && m.dex.tab == 0 && (m.dexWide() || !m.dex.overviewFacts) && m.dexMaxHorizontal() > 0 {
				delta := 5
				if k == "left" {
					delta = -5
				}
				if delta < 0 && m.dex.horizontal == 0 || delta > 0 && m.dex.horizontal >= m.dexMaxHorizontal() {
					if m.focus == 10 {
						m.focus = 8
						if delta < 0 {
							m.focus = 6
							if m.dexWide() {
								m.focus = 0
							}
						}
					} else {
						m.navigateDexControl(k)
					}
				} else {
					m.dex.horizontal = max(0, min(m.dex.horizontal+delta, m.dexMaxHorizontal()))
				}
			} else if m.focus == 0 && k == "right" {
				m.dex.detail = true
				m.focus = m.dexContentFocus()
				m.enterEvolutionFocus()
			} else {
				m.navigateDexControl(k)
			}
		}
		m.ensureDexFocus()
		if k == "[" || k == "]" {
			m.revealDexFocus()
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
		if m.dex.filters && !m.dexWide() && m.height < 16 {
			return nil, true
		}
		g := m.dexGeometry()
		if msg.Y <= g.bodyY || msg.Y >= g.bodyY+max(g.bodyH, g.listH)-1 {
			return nil, true
		}
		inDetail := (m.dex.detail || m.dexWide()) && msg.X > g.entryX && msg.X < g.entryX+g.entryW-1 && msg.Y < g.bodyY+g.bodyH-1
		inIndex := (m.dexWide() || !m.dex.detail) && msg.X > g.listX && msg.X < g.listX+g.listW-1
		if m.dexWide() && m.dex.tab == 0 {
			x, y, w, h := m.dexFactsRect()
			if msg.X > x && msg.X < x+w-1 && msg.Y > y && msg.Y < y+h-1 {
				if msg.Button == tea.MouseWheelUp || msg.Button == tea.MouseWheelDown {
					m.dex.factsScroll = max(0, min(m.dexFactsMaxScroll(), m.dex.factsScroll+delta))
				}
				return nil, true
			}
		}
		if m.dexWide() && m.dex.tab == 0 && inDetail {
			x, y, w, h := m.dexArtRect()
			if msg.X <= x || msg.X >= x+w-1 || msg.Y <= y || msg.Y >= y+h-1 {
				return nil, true
			}
		}
		if !inDetail && !inIndex {
			return nil, true
		}
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
	if m.focus == 18 && m.dex.tab == 1 {
		m.revealDexOption()
		return
	}
	if m.focus == 18 && m.dex.tab == 2 && m.dex.cardNode >= 2000 {
		m.focus = m.dex.cardNode
		m.revealDexFocus()
		m.focus = 18
		return
	}
	if m.focus < 2000 || m.focus >= 3000 {
		return
	}
	m.dex.cardNode = m.focus
	for i, line := range m.dexLines() {
		matches := line.target == m.focus
		for _, node := range line.nodes {
			matches = matches || node.id == m.focus
		}
		if matches {
			offset := i + 1
			m.dex.scroll = max(0, min(offset, m.dexMaxScroll()))
			break
		}
	}
}
func (m *Model) revealDexOption() {
	for row, line := range m.dexLines() {
		for _, node := range line.nodes {
			if node.id == 3000+m.dex.optionIndex {
				offset := row + 1
				m.dex.scroll = min(max(0, offset), m.dexMaxScroll())
				return
			}
		}
	}
}
func (m Model) dexTabFocus() int {
	if !m.dexWide() && m.height < 20 {
		return 24
	}
	return 20 + m.dex.tab
}
func (m *Model) ensureDexFocus() {
	for _, id := range m.dexFocusOrder() {
		if id == m.focus {
			if m.focus == 18 && m.dex.tab == 2 {
				valid := false
				for _, n := range m.dex.entry.Evolution {
					valid = valid || m.dex.cardNode == 2000+n.Number
				}
				if !valid && len(m.dex.entry.Evolution) > 0 {
					m.dex.cardNode = 2000 + orderedEvolution(m.dex.entry.Evolution)[0].Number
				}
			}
			return
		}
	}
	if pokedexEntryFocus(m.focus) && (m.dexWide() || m.dex.detail) {
		m.focus = m.dexContentFocus()
		return
	}
	m.focus = 1
}
func (m *Model) navigateDexControl(direction string) {
	m.navigatePokedex(direction)
	m.ensureDexFocus()
	m.revealDexFocus()
}

func absFloat(n float64) float64 {
	if n < 0 {
		return -n
	}
	return n
}

func (m *Model) enterEvolutionFocus() {
	if m.dex.tab == 2 && m.dex.entry.Seen && len(m.dex.entry.Evolution) > 0 {
		m.focus = 2000 + orderedEvolution(m.dex.entry.Evolution)[0].Number
		m.revealDexFocus()
	}
}
func (m *Model) navigateEvolution(key string) {
	if key == "j" {
		key = "down"
	}
	if key == "k" {
		key = "up"
	}
	if m.focus == 10 {
		m.enterEvolutionFocus()
		return
	}
	targets := []dexControl{}
	seen := map[int]bool{}
	for y, line := range m.evolutionLines() {
		for _, node := range line.nodes {
			if !seen[node.id] {
				node.y = y
				targets = append(targets, node)
				seen[node.id] = true
			}
		}
	}
	next := directionalTarget(targets, m.focus, key)
	if next == m.focus {
		m.pokedexContentExit(key)
	} else {
		m.focus = next
		m.revealDexFocus()
	}

}
func (m *Model) returnToFamily() tea.Cmd {
	origin, node, q := m.dex.familyOrigin, m.dex.familyNode, m.dex.familySelection
	m.dex.familyOrigin = 0
	m.activateDex(2000 + origin)
	m.dex.tab = 2
	m.focus = node
	return m.loadDexEntry(q)
}
