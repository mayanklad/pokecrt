package tui

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/mayanklad/pokecrt/internal/catalog"
	"github.com/mayanklad/pokecrt/internal/sprite"
	"github.com/mayanklad/pokecrt/internal/storage"
	"github.com/mayanklad/pokecrt/internal/trainer"
)

func dexModel(t *testing.T, records trainer.DexRecords) Model {
	t.Helper()
	m := New(context.Background(), nil, Dark, false)
	m.width, m.height = 120, 40
	d := trainer.NewDex(catalog.All(), records, func(k catalog.VariantKey) bool { _, ok := sprite.Lookup(k); return ok })
	m.dexLoader = func(context.Context) (dexSnapshot, error) { return dexSnapshot{1, "Ash", d, d.Summary()}, nil }
	cmd := m.openDex()
	m, next := update(m, cmd())
	if next != nil {
		m, _ = update(m, next())
	}
	return m
}
func collectedDex() trainer.DexRecords {
	return trainer.DexRecords{Encounters: 1, Species: map[int]trainer.Discovery{6: {Count: 1, FirstMS: 10, LastMS: 10}}, Variants: []trainer.VariantDiscovery{{Key: catalog.VariantKey{SpeciesID: 6, FormID: "mega-x", Gender: "default", Palette: "shiny"}, Discovery: trainer.Discovery{Count: 1, FirstMS: 10, LastMS: 10}, FormName: "Mega X", Types: []string{"fire", "dragon"}}}}
}
func applyEntry(t *testing.T, m Model, cmd tea.Cmd) Model {
	t.Helper()
	if cmd == nil {
		t.Fatal("missing worker")
	}
	m, _ = update(m, cmd())
	return m
}
func TestDexDisclosureSearchAndExactArtwork(t *testing.T) {
	m := dexModel(t, collectedDex())
	if len(m.dex.rows) != 1025 || strings.Contains(m.View().Content, "Bulbasaur") {
		t.Fatal("anonymous list leaked")
	}
	m.dex.query = "bulbasaur"
	cmd := m.filterDex()
	if len(m.dex.rows) != 0 || cmd != nil {
		t.Fatal("search matched unseen name")
	}
	m.dex.query = "charizard"
	m = applyEntry(t, m, m.filterDex())
	if len(m.dex.rows) != 1 || m.dex.entry.Number != 6 || m.dex.art != "" || !strings.Contains(m.dex.entry.Notice, "LOCKED FORM") {
		t.Fatal("alternate discovery fell back to artwork")
	}
	if len(m.dex.options) != 2 || strings.Contains(m.dex.options[0].Label, "standard") {
		t.Fatal("unobserved form revealed")
	}
	m = applyEntry(t, m, m.activateDex(3001))
	if m.dex.art == "" || m.dex.entry.SelectedName == "" || m.dex.entry.ArtworkKey == nil {
		t.Fatal("collected artwork missing")
	}
	m = applyEntry(t, m, m.activateDex(3000))
	if m.dex.art != "" || !strings.Contains(m.dex.entry.Notice, "LOCKED VARIANT") {
		t.Fatal("uncollected palette preview")
	}
	for _, node := range m.dex.entry.Evolution {
		if node.Number != 6 && node.Name != "?????" {
			t.Fatal("unseen evolution revealed")
		}
	}
	m = applyEntry(t, m, m.activateDex(2004))
	if m.dex.entry.Seen || m.dex.art != "" || strings.Contains(m.View().Content, "Charmander") {
		t.Fatal("evolution link leaked")
	}
}
func TestDexAsyncStaleResultsResizeAndSettings(t *testing.T) {
	m := dexModel(t, collectedDex())
	m.dex.query = "6"
	m = applyEntry(t, m, m.filterDex())
	old := m.loadDexEntry(catalog.Selection{Form: "mega-x", Shiny: true})
	m.dex.selected = 0
	current := m.loadDexEntry(catalog.Selection{})
	m, _ = update(m, old())
	if !m.dex.entryLoading || m.dex.art != "" {
		t.Fatal("stale sprite accepted")
	}
	m = applyEntry(t, m, current)
	cmd := m.reloadDex()
	if cmd == nil || !m.dex.loading {
		t.Fatal("load not deferred")
	}
	m, _ = update(m, tea.WindowSizeMsg{Width: 48, Height: 24})
	m.openSettings()
	focus := m.focus
	m, next := update(m, cmd())
	if !m.settings || m.focus != focus {
		t.Fatal("worker stole settings focus")
	}
	if next != nil {
		m = applyEntry(t, m, next)
	}
	stale := m.reloadDex()
	m.leaveDex()
	m, _ = update(m, stale())
	if m.screen != mainScreen {
		t.Fatal("late load reopened Dex")
	}
}
func TestDexSearchTypingAndMouseKeyboard(t *testing.T) {
	m := dexModel(t, collectedDex())
	m.openDexSearch()
	for _, r := range "qajk" {
		m, _ = update(m, key(r))
	}
	if string(m.name) != "qajk" || m.settings {
		t.Fatal("search shortcuts intercepted typing")
	}
	m, _ = update(m, key(tea.KeyDown))
	if m.focus != 100 {
		t.Fatal("search grid inaccessible")
	}
	m.activateDex(101)
	if string(m.name) != "qajkb" || m.focus != 101 {
		t.Fatal("search key activation")
	}
	m, _ = update(m, key(tea.KeyEscape))
	if m.dex.query != "" || m.screen != dexScreen {
		t.Fatal("cancel applied query")
	}
	m.openDexSearch()
	m.name = []rune("charizard")
	m.cursor = len(m.name)
	m = applyEntry(t, m, m.activateDex(1))
	if len(m.dex.rows) != 1 {
		t.Fatal("search did not apply")
	}
}
func TestDexEveryScreenFitsEveryModeAndControlsHaveMouseTargets(t *testing.T) {
	for _, size := range [][2]int{{190, 50}, {120, 40}, {100, 24}, {80, 24}, {48, 36}, {40, 24}, {40, 12}, {32, 8}} {
		for view := 0; view < 6; view++ {
			for _, a := range appearances {
				for _, nc := range []bool{false, true} {
					m := dexModel(t, collectedDex())
					m.width, m.height = size[0], size[1]
					m.appearance = a
					m.noColor = nc
					if view == 5 {
						m.screen = dexSearchScreen
						m.name = []rune("Query")
						m.cursor = 2
					} else if view > 0 {
						m.dex.detail = true
						m.dex.tab = view - 1
					}
					frame := m.View()
					lines := strings.Split(frame.Content, "\n")
					if len(lines) != size[1] {
						t.Fatal("height")
					}
					for _, line := range lines {
						if ansi.StringWidth(line) != size[0] {
							t.Fatal("width")
						}
					}
					if nc && strings.Contains(frame.Content, "\x1b") {
						t.Fatal("NO_COLOR")
					}
					ids := map[int]bool{}
					for y := 0; y < size[1]; y++ {
						for x := 0; x < size[0]; x++ {
							cmd := frame.OnMouse(tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft})
							if cmd != nil {
								switch k := cmd().(type) {
								case activateMsg:
									ids[int(k)] = true
								case nameCursorMsg:
									ids[0] = true
								}
							}
						}
					}
					expected := []int{1, 5, 6, 7, 8}
					if m.height >= 20 && (m.dexWide() || !m.dex.detail) {
						expected = append(expected, 3, 31, 32, 33)
					}
					if m.height >= 20 && (m.dexWide() || m.dex.detail) {
						expected = append(expected, 20, 21, 22, 23)
					}
					if m.height < 20 && m.dex.detail {
						expected = append(expected, 24)
					}
					if view == 5 {
						expected = []int{0, 1, 2, 3, 4, 5, 6, 7, 100, 125}
					}
					if size[0] < 40 || size[1] < 12 {
						expected = []int{8}
						if view == 5 {
							expected = []int{7}
						}
					}
					for _, id := range expected {
						if !ids[id] {
							t.Fatalf("%v view%d missing target%d", size, view, id)
						}
					}
				}
			}
		}
	}
}
func TestLoadDexMissingAndBrowseStorageUnchanged(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("POKECRT_DATA_DIR", filepath.Join(dir, "state"))
	_, err := loadDex(context.Background())
	if err == nil {
		t.Fatal("missing trainer allowed")
	}
	if _, err = os.Stat(filepath.Join(dir, "state")); !os.IsNotExist(err) {
		t.Fatal("browse initialized state")
	}
	path, err := storage.ResolvePath()
	if err != nil {
		t.Fatal(err)
	}
	repo, err := storage.Initialize(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	name, _ := trainer.ParseName("Ash")
	_, err = repo.CreateProfile(context.Background(), name, 1)
	if err != nil {
		t.Fatal(err)
	}
	repo.Close()
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	snap, err := loadDex(context.Background())
	if err != nil || snap.name != "Ash" {
		t.Fatal(snap, err)
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("browse mutated database")
	}
}

func TestDexEvolutionKeyboardFocusScrollAndLocks(t *testing.T) {
	m := dexModel(t, collectedDex())
	m.dex.query = "charizard"
	m = applyEntry(t, m, m.filterDex())
	m.width, m.height = 40, 12
	m.dex.detail = true
	m.dex.tab = 2
	m.focus = 10
	m.moveDexFocus(1)
	if m.focus != 2004 || m.dex.scroll == 0 || !strings.Contains(m.View().Content, "▶") {
		t.Fatal("evolution unreachable by keyboard")
	}
	cmd := m.activateDex(m.focus)
	m = applyEntry(t, m, cmd)
	if m.dex.entry.Seen || m.dex.art != "" || !strings.Contains(m.View().Content, "LOCKED") {
		t.Fatal("unseen evolution link revealed")
	}
}
func TestDexFiltersAndSpatialFocusDoNotActivateOrTrapLists(t *testing.T) {
	m := dexModel(t, collectedDex())
	m = applyEntry(t, m, m.activateDex(32))
	if m.focus != 32 || len(m.dex.rows) != 1 {
		t.Fatal("Seen filter lost focus")
	}
	m.focus = 31
	m.navigateDexControl("right")
	if m.focus != 32 {
		t.Fatal("filter row skipped Seen")
	}
	m.dex.detail = true
	m.dex.tab = 0
	m.focus = 20
	m.navigateDexControl("right")
	if m.focus != 21 || m.dex.tab != 0 {
		t.Fatal("tab movement activated tab")
	}
	m.activateDex(21)
	if m.dex.tab != 1 {
		t.Fatal("tab activation")
	}
	m.width, m.height = 48, 36
	m.focus = 20
	m.navigateDexControl("down")
	if m.focus != 22 {
		t.Fatal("compact tab crossed columns")
	}
	m.focus = 0
	m, _ = update(m, key(tea.KeyDown))
	if m.focus == 0 {
		t.Fatal("list trap")
	}
	m.dex.tab = 0
	m.focus = 10
	m.dex.scroll = 0
	m, _ = update(m, key(tea.KeyUp))
	if m.focus == 10 {
		t.Fatal("display trap")
	}
}
func BenchmarkDexView(b *testing.B) {
	m := New(context.Background(), nil, Dark, false)
	m.width, m.height = 120, 40
	m.screen = dexScreen
	d := trainer.NewDex(catalog.All(), collectedDex(), func(k catalog.VariantKey) bool { _, ok := sprite.Lookup(k); return ok })
	m.dex.snapshot = dexSnapshot{1, "Ash", d, d.Summary()}
	m.dex.rows = d.List(trainer.DexFilter{})
	m.dex.selected = 5
	cmd := m.loadDexEntry(catalog.Selection{Form: "mega-x", Shiny: true})
	next, _ := m.Update(cmd())
	m = next.(Model)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		m.View()
	}
}

func TestDexTrainerSwitchDiscardsPreviousCollectionAndDoesNotWrite(t *testing.T) {
	ctx := context.Background()
	t.Setenv("POKECRT_DATA_DIR", filepath.Join(t.TempDir(), "state"))
	path, _ := storage.ResolvePath()
	repo, err := storage.Initialize(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	ash, _ := trainer.ParseName("Ash")
	p, err := repo.CreateProfile(ctx, ash, 1)
	if err != nil {
		t.Fatal(err)
	}
	species, _ := catalog.ByNumber(6)
	for _, f := range species.Forms {
		if f.ID == "mega-x" {
			f.Genders = []string{"default"}
			species.Forms = []catalog.Form{f}
			break
		}
	}
	pool, err := trainer.NewPool([]catalog.Species{species}, func(k catalog.VariantKey) bool { _, ok := sprite.Lookup(k); return ok })
	if err != nil {
		t.Fatal(err)
	}
	choice, err := pool.Select(func(int) (int, error) { return 0, nil })
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.RecordEncounter(ctx, p.ID, choice, 10); err != nil {
		t.Fatal(err)
	}
	misty, _ := trainer.ParseName("Misty")
	if _, err = repo.CreateProfile(ctx, misty, 2); err != nil {
		t.Fatal(err)
	}
	repo.Close()
	before, _ := os.ReadFile(path)
	m := New(ctx, nil, Dark, false)
	m.width, m.height = 120, 40
	m.dexLoader = loadDex
	cmd := m.openDex()
	m, next := update(m, cmd())
	m = applyEntry(t, m, next)
	if m.dex.snapshot.summary.Completion.Species != 1 {
		t.Fatal("Ash collection missing")
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("Dex read wrote history or progression")
	}
	m.leaveDex()
	repo, err = storage.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.UseProfile(ctx, misty)
	if err != nil {
		t.Fatal(err)
	}
	repo.Close()
	before, _ = os.ReadFile(path)
	cmd = m.openDex()
	m, next = update(m, cmd())
	m = applyEntry(t, m, next)
	if m.dex.snapshot.name != "Misty" || m.dex.snapshot.summary.Completion.Species != 0 {
		t.Fatal("previous trainer cache retained")
	}
	m.dex.query = "charizard"
	m.filterDex()
	if len(m.dex.rows) != 0 {
		t.Fatal("previous trainer search leaked")
	}
	after, _ = os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("second trainer browse wrote storage")
	}
}

func TestDexResizeKeepsFocusedPaneAndSearchWheelDoesNotBrowse(t *testing.T) {
	m := dexModel(t, collectedDex())
	m.dex.query = "charizard"
	m = applyEntry(t, m, m.filterDex())
	m.focus = 10
	m.dex.detail = false
	m, _ = update(m, tea.WindowSizeMsg{Width: 48, Height: 24})
	if !m.dex.detail || m.focus != 10 || !strings.Contains(m.View().Content, "DEVICE DISPLAY") {
		t.Fatal("resize hid focused detail")
	}
	m.dex.detail = false
	m.activateDex(12)
	if !m.dex.detail {
		t.Fatal("mouse scroll left detail hidden")
	}
	selected := m.dex.selected
	m.openDexSearch()
	m, _ = update(m, tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	if m.dex.selected != selected || m.dex.entryLoading {
		t.Fatal("search wheel browsed underlying list")
	}
}

func TestDeviceTabsSeparateFactsAndPreserveExactVariant(t *testing.T) {
	m := dexModel(t, collectedDex())
	m.dex.query = "charizard"
	m = applyEntry(t, m, m.filterDex())
	m = applyEntry(t, m, m.activateDex(3001))
	keyBefore := *m.dex.entry.ArtworkKey
	overview := m.View().Content
	if !strings.Contains(overview, "DEVICE DISPLAY") || strings.Contains(overview, "First:") || strings.Contains(overview, "OBSERVED FORMS") {
		t.Fatal("overview is a raw record dump")
	}
	for tab := 1; tab < 4; tab++ {
		m.activateDex(20 + tab)
		if m.dex.tab != tab || m.dex.entryLoading || *m.dex.entry.ArtworkKey != keyBefore {
			t.Fatal("tabs reloaded or changed appearance")
		}
	}
	records := m.View().Content
	if !strings.Contains(records, "SPECIES") || !strings.Contains(records, "First:") {
		t.Fatal("records missing")
	}
	m.activateDex(21)
	if !strings.Contains(m.View().Content, "LOCKED") || !strings.Contains(m.View().Content, "COLLECTED") {
		t.Fatal("variant states missing")
	}
	m = applyEntry(t, m, m.activateDex(3000))
	if m.dex.art != "" || m.dex.tab != 0 {
		t.Fatal("locked palette previewed artwork")
	}
}
func TestDeviceFocusOrderNoDuplicatesAndCompactFilterCanCloseWithMouse(t *testing.T) {
	for _, size := range [][2]int{{120, 40}, {48, 36}, {48, 24}, {40, 12}} {
		for tab := 0; tab < 4; tab++ {
			m := dexModel(t, collectedDex())
			m.width, m.height = size[0], size[1]
			m.dex.detail = true
			m.dex.tab = tab
			order := m.dexFocusOrder()
			seen := map[int]bool{}
			for _, id := range order {
				if seen[id] {
					t.Fatalf("duplicate focus %d at%v", id, size)
				}
				seen[id] = true
			}
			m.focus = order[0]
			for i := 0; i < len(order); i++ {
				m.moveDexFocus(1)
			}
			if m.focus != order[0] {
				t.Fatal("Tab traversal traps focus")
			}
		}
	}
	m := dexModel(t, collectedDex())
	m.width, m.height = 48, 24
	m.dex.detail = true
	m.activateDex(25)
	if !m.dex.filters {
		t.Fatal("filters did not open")
	}
	found := false
	for _, c := range m.dexControls() {
		if c.id == 25 {
			found = true
		}
	}
	if !found {
		t.Fatal("mouse cannot close filters")
	}
	m.activateDex(25)
	if m.dex.filters {
		t.Fatal("filters did not close")
	}
}
func TestDeviceNoColorCollectedArtworkAndBorders(t *testing.T) {
	m := dexModel(t, collectedDex())
	m.noColor = true
	m.dex.query = "charizard"
	m = applyEntry(t, m, m.filterDex())
	m = applyEntry(t, m, m.activateDex(3001))
	for _, size := range [][2]int{{120, 40}, {48, 36}, {40, 24}, {40, 12}} {
		m.width, m.height = size[0], size[1]
		m.dex.detail = true
		for tab := 0; tab < 4; tab++ {
			m.dex.tab = tab
			s := m.View().Content
			if strings.Contains(s, "\x1b") {
				t.Fatal("NO_COLOR art styling")
			}
			lines := strings.Split(s, "\n")
			for _, line := range lines {
				r := []rune(line)
				if len(r) == 0 || r[0] != '║' && r[0] != '╔' && r[0] != '╚' {
					t.Fatal("outer border overwritten", line)
				}
			}
		}
	}
}

func TestDeviceFilteringKeepsExplicitAppearanceForSameEntry(t *testing.T) {
	m := dexModel(t, collectedDex())
	m.dex.query = "charizard"
	m = applyEntry(t, m, m.filterDex())
	m = applyEntry(t, m, m.activateDex(3001))
	selected := *m.dex.entry.ArtworkKey
	m = applyEntry(t, m, m.activateDex(32))
	if m.dex.entry.ArtworkKey == nil || *m.dex.entry.ArtworkKey != selected {
		t.Fatal("Seen filter lost explicit appearance")
	}
	m.dex.query = ""
	m = applyEntry(t, m, m.filterDex())
	if m.dex.entry.ArtworkKey == nil || *m.dex.entry.ArtworkKey != selected {
		t.Fatal("search clearing lost explicit appearance")
	}
}
