package trainer

import (
	"encoding/json"
	"github.com/mayanklad/pokecrt/internal/catalog"
	"strings"
	"testing"
)

func TestTrainerStatisticsRetainHistoricalCountsAndIntersectInventory(t *testing.T) {
	records := TrainerRecords{Dex: dexFixture("mega-x", "shiny"), XP: 1110, ShinyEncounters: 3}
	first, last := int64(0), int64(30)
	records.FirstEncounterMS = &first
	records.LastEncounterMS = &last
	records.Dex.Encounters = 4
	records.Dex.Species[99999] = Discovery{Count: 3}
	records.Dex.Variants = append(records.Dex.Variants, VariantDiscovery{Key: catalog.VariantKey{SpeciesID: 99999, FormID: "removed", Gender: "default", Palette: "shiny"}, Discovery: Discovery{Count: 2}})
	stats, err := TrainerStatistics(records, catalog.All(), dexAvailable)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Progress.Level != 2 || stats.Progress.InLevel != 110 || stats.Progress.ToNext != 890 || stats.Encounters != 4 || stats.Species != 2 || stats.UnclassifiedSpecies != 1 || stats.Variants != 2 || stats.ShinyEncounters != 3 || stats.ShinyCollections != 2 || stats.Completion.Species != 1 || stats.Completion.Variants != 1 || *stats.FirstEncounterMS != 0 {
		t.Fatal(stats)
	}
	first = 99
	if *stats.FirstEncounterMS != 0 {
		t.Fatal("aliased clock pointer")
	}
	removed, err := TrainerStatistics(records, catalog.All(), func(catalog.VariantKey) bool { return false })
	if err != nil || removed.Species != 2 || removed.Variants != 2 || removed.Completion.SpeciesTotal != 0 || removed.Completion.VariantsTotal != 0 || removed.Completion.Species != 0 {
		t.Fatal(removed, err)
	}
	empty, err := TrainerStatistics(TrainerRecords{}, catalog.All(), dexAvailable)
	if err != nil || empty.Progress.Level != 1 || empty.FirstEncounterMS != nil || empty.LastEncounterMS != nil {
		t.Fatal(empty, err)
	}
	if _, err = TrainerStatistics(TrainerRecords{XP: -1}, catalog.All(), dexAvailable); err == nil {
		t.Fatal("invalid persisted XP accepted")
	}
}

func TestAchievementViewsRegistryOrderPendingAndRetainedUnlocks(t *testing.T) {
	targets := bundledTargets(t)
	state := AchievementState{Encounters: 10, Species: 5, SeenSpecies: map[int]bool{6: true}, EncounteredTypes: map[string]bool{"fire": true, "dragon": true}}
	input := []Unlock{{ID: "encounters.1", UnlockedAtMS: 10, Target: 1, HasTarget: true}, {ID: "shiny.first", UnlockedAtMS: 20}, {ID: "retired.fixture", UnlockedAtMS: 30, DatasetID: "old"}}
	views := targets.Views(state, input)
	if len(views.Unlocked) != 3 || views.Unlocked[0].Name != "First Contact" || views.Unlocked[1].Name != "Shiny Discovery" || views.Unlocked[2].Name != "Retained achievement" || views.Unlocked[0].TargetAtUnlock != 1 || views.Unlocked[0].EarnedAtMS != 10 {
		t.Fatal(views)
	}
	if views.Locked[0].ID != "encounters.10" || views.Locked[0].Current != 10 || views.Locked[0].Target != 10 {
		t.Fatal("ready goal was not retained locked", views.Locked[0])
	}
	b, _ := json.Marshal(views)
	for _, hidden := range []string{"Charizard", "Mega X", "Charmander", "retired.fixture"} {
		if hidden == "retired.fixture" {
			continue
		}
		if strings.Contains(string(b), hidden) {
			t.Fatal("identity leaked", string(b))
		}
	}
	if len(input) != 3 || input[0].UnlockedAtMS != 10 {
		t.Fatal("mutated unlocks")
	}
	unavailable, err := NewAchievementTargets(catalog.All(), func(catalog.VariantKey) bool { return false })
	if err != nil {
		t.Fatal(err)
	}
	retained := unavailable.Views(state, []Unlock{{ID: "national.complete", UnlockedAtMS: 42, Target: 1017, HasTarget: true}})
	if len(retained.Locked) != 0 || len(retained.Unlocked) != 1 || retained.Unlocked[0].Name != "National Researcher" || retained.Unlocked[0].TargetAtUnlock != 1017 {
		t.Fatal(retained)
	}
	empty := targets.Views(AchievementState{}, nil)
	if len(empty.Locked) != 50 || len(empty.Unlocked) != 0 {
		t.Fatal("50 initial goals", empty)
	}
}
