package tui

import (
	tea "charm.land/bubbletea/v2"
	"context"
	"errors"
	"github.com/mayanklad/pokecrt/internal/storage"
	"github.com/mayanklad/pokecrt/internal/trainer"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func activityModel() Model {
	m := New(context.Background(), nil, Dark, false)
	m.width = 120
	m.height = 40
	m.screen = activityScreen
	m.section = 1
	m.focus = 22
	m.activity.data.profile = trainer.Profile{ID: 1, Name: "Ash"}
	return m
}
func TestEncounterExplicitAndInFlightGuard(t *testing.T) {
	m := activityModel()
	calls := 0
	m.encounterAction = func(context.Context, bool, int64) (trainer.EncounterResult, error) {
		calls++
		return trainer.EncounterResult{}, errors.New("retry")
	}
	for _, id := range []int{22, 16, 17, 31, 32} {
		if cmd := m.activateActivity(id); cmd != nil {
			t.Fatalf("browse %d produced action", id)
		}
	}
	if calls != 0 {
		t.Fatal("implicit encounter")
	}
	cmd := m.activateActivity(10)
	if cmd == nil || !m.activity.recording {
		t.Fatal("explicit action missing")
	}
	if m.activateActivity(10) != nil {
		t.Fatal("duplicate request")
	}
	m.activateActivity(12)
	if m.screen != mainScreen || !m.encounterPending {
		t.Fatal("navigation blocked or in-flight guard lost")
	}
	m.screen = activityScreen
	if m.activateActivity(10) != nil {
		t.Fatal("navigation allowed duplicate write")
	}
	msg := cmd()
	if calls != 1 {
		t.Fatal(calls)
	}
	m.handleActivity(msg)
	if m.activity.recording || m.activity.error != "retry" {
		t.Fatal("failed action is not recoverable")
	}
	if m.activateActivity(10) == nil {
		t.Fatal("retry disabled")
	}
}
func TestActivityStaleReadsAndTabSelection(t *testing.T) {
	m := activityModel()
	m.activity.generation = 4
	m.activity.loading = true
	m.handleActivity(activityLoaded{3, activitySnapshot{profile: trainer.Profile{ID: 2, Name: "Misty"}}, nil})
	if !m.activity.loading || m.activity.data.profile.ID != 1 {
		t.Fatal("stale trainer read applied")
	}
	m.activityLoader = func(context.Context) (activitySnapshot, error) {
		return activitySnapshot{profile: trainer.Profile{ID: 2}}, nil
	}
	m.activity.loading = false
	m.openActivity(3)
	data, _ := m.activityLoader(m.ctx)
	m.handleActivity(activityLoaded{m.activity.generation, data, nil})
	if m.activity.data.profile.ID != 2 || m.activity.result != nil || m.activity.selected != 0 {
		t.Fatal("trainer scoped state retained")
	}
}
func TestActivityLayoutsControlsAndFocus(t *testing.T) {
	for _, size := range [][2]int{{120, 40}, {100, 24}, {80, 24}, {48, 36}, {40, 12}, {32, 8}} {
		for _, section := range []int{1, 2, 3} {
			for _, mode := range appearances {
				m := activityModel()
				m.width = size[0]
				m.height = size[1]
				m.section = section
				m.appearance = mode
				text := m.View().Content
				if text == "" {
					t.Fatal("empty view")
				}
				seen := map[int]bool{}
				for _, c := range m.activityControls() {
					if seen[c.id] {
						t.Fatalf("duplicate control %d", c.id)
					}
					seen[c.id] = true
					if c.x+c.w > m.dexGeometry().x+m.dexGeometry().w-1 {
						t.Fatalf("overflow %#v", c)
					}
				}
				if size[0] >= 40 && size[1] >= 12 {
					m.focus = 12
					model, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
					m = model.(Model)
					if m.focus == 12 {
						t.Fatal("tab stuck")
					}
				}
			}
		}
	}
}
func TestActivityReadsDoNotInitializeOrMutate(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("POKECRT_DATA_DIR", dir)
	if _, err := loadActivity(context.Background()); !errors.Is(err, storage.ErrNoState) {
		t.Fatal(err)
	}
	files, _ := os.ReadDir(dir)
	if len(files) != 0 {
		t.Fatal("read initialized state")
	}
	path := filepath.Join(dir, "trainers.sqlite3")
	r, err := storage.Initialize(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	name, _ := trainer.ParseName("Ash")
	p, err := r.CreateProfile(context.Background(), name, 123)
	if err != nil {
		t.Fatal(err)
	}
	r.Close()
	before, _ := os.ReadFile(path)
	data, err := loadActivity(context.Background())
	if err != nil || data.profile.ID != p.ID {
		t.Fatalf("load: %v", err)
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("reading changed database")
	}
	m := activityModel()
	m.section = 3
	m.activity.data = data
	for _, line := range m.activityLines() {
		if strings.Contains(line, "pikachu") {
			t.Fatal("hidden identity leaked")
		}
	}
}

func TestHistoryOpensExactCollectedAppearance(t *testing.T) {
	m := dexModel(t, collectedDex())
	m.screen = activityScreen
	m.section = 1
	m.activity.data.profile = trainer.Profile{ID: 1}
	key := collectedDex().Variants[0].Key
	m.activity.data.history = []storage.HistoryEntry{{Key: key}}
	m.activateActivity(20)
	snapshot, _ := m.dexLoader(m.ctx)
	cmd, _ := m.handleDex(dexLoadedMsg{m.dex.generation, snapshot, nil})
	m = applyEntry(t, m, cmd)
	if m.dex.entry.ArtworkKey == nil || *m.dex.entry.ArtworkKey != key || !m.dex.detail {
		t.Fatal("history did not open exact collected variant")
	}
}
func TestActivityReloadClearsOtherTrainersResult(t *testing.T) {
	m := activityModel()
	m.activity.result = &trainer.Record{TrainerID: 1}
	m.activity.art = []string{"secret"}
	m.activity.generation = 5
	m.handleActivity(activityLoaded{5, activitySnapshot{profile: trainer.Profile{ID: 2}}, nil})
	if m.activity.result != nil || len(m.activity.art) != 0 {
		t.Fatal("previous trainer result leaked")
	}
}
func BenchmarkActivityView(b *testing.B) {
	for _, section := range []int{1, 2, 3} {
		b.Run(sections[section], func(b *testing.B) {
			m := activityModel()
			m.section = section
			m.activity.historyMode = true
			for i := 0; i < 50; i++ {
				m.activity.data.history = append(m.activity.data.history, storage.HistoryEntry{Snapshot: trainer.Snapshot{SpeciesName: "Charizard", FormName: "Mega X"}})
				m.activity.data.achievements.Locked = append(m.activity.data.achievements.Locked, trainer.AchievementView{Name: "Collection goal", Description: "Collect eligible species.", Current: 1, Target: 50, HasTarget: true})
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = m.View()
			}
		})
	}
}

func TestHistoryActionsHaveBoundedWidths(t *testing.T) {
	m := New(context.Background(), nil, Dark, true)
	m.width, m.height, m.section = 120, 40, 1
	m.activity.historyMode = true
	m.activity.data.history = make([]storage.HistoryEntry, 1)
	for _, control := range m.activityControls() {
		if control.id == 20 || control.id == 31 || control.id == 32 {
			if control.w > 30 {
				t.Fatal("history action stretches across panel", control)
			}
		}
	}
}
