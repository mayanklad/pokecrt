package trainer

import (
	"encoding/json"
	"fmt"
	"github.com/mayanklad/pokecrt/internal/catalog"
	"github.com/mayanklad/pokecrt/internal/sprite"
	"math"
	"os"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func TestXPAwardsAndLevels(t *testing.T) {
	for _, c := range []struct {
		s, v, sh bool
		want     int64
	}{{true, true, false, 70}, {false, true, false, 30}, {false, false, false, 10}, {true, true, true, 170}, {false, true, true, 130}, {false, false, true, 110}} {
		if got := AwardXP(c.s, c.v, c.sh); got != c.want {
			t.Fatalf("award %d want %d", got, c.want)
		}
	}
	for _, total := range []int64{0, 999, 1000, 1999, 100000, math.MaxInt64} {
		p, err := XPProgress(total)
		if err != nil || p.Level != 1+total/1000 || p.InLevel != total%1000 || p.ToNext != 1000-total%1000 {
			t.Fatalf("progress %+v %v", p, err)
		}
	}
	for _, c := range [][2]int64{{math.MaxInt64, 1}, {-1, 10}, {0, -1}} {
		if _, err := AdvanceXP(c[0], c[1]); err == nil {
			t.Fatal("invalid XP accepted")
		}
	}
}

func bundledTargets(t *testing.T) *AchievementTargets {
	t.Helper()
	targets, err := NewAchievementTargets(catalog.All(), func(k catalog.VariantKey) bool { _, ok := sprite.Lookup(k); return ok })
	if err != nil {
		t.Fatal(err)
	}
	return targets
}

func TestAllNumericAchievementBoundaries(t *testing.T) {
	targets := bundledTargets(t)
	for _, goal := range targets.Goals(AchievementState{}) {
		if !strings.HasPrefix(goal.ID, "encounters.") && !strings.HasPrefix(goal.ID, "species.") && !strings.HasPrefix(goal.ID, "variants.") {
			continue
		}
		for _, delta := range []int64{-1, 0, 1} {
			n := goal.Target + delta
			state := AchievementState{Encounters: n, Species: n, Variants: n}
			found := false
			for _, g := range targets.Goals(state) {
				if g.ID == goal.ID {
					found = true
					if g.Ready() != (delta >= 0) {
						t.Fatalf("%s boundary %d", g.ID, delta)
					}
				}
			}
			if !found {
				t.Fatal("goal disappeared")
			}
		}
	}
}

func TestCompletionUsesEligibleIntersectionAndEncounteredTypes(t *testing.T) {
	targets := bundledTargets(t)
	if len(targets.eligible) != 1017 || targets.variants != 2669 || len(targets.generations) != 9 || len(targets.types) != 18 || len(targets.families) == 0 {
		t.Fatalf("unexpected derived inventory %+v", targets)
	}
	state := AchievementState{SeenSpecies: map[int]bool{99999: true}, EncounteredTypes: map[string]bool{}, Species: 1025, Variants: 3000, Shiny: true, Regional: true, Transformation: true}
	for _, g := range targets.Goals(state) {
		if (g.ID == "national.complete" || g.ID == "types.complete" || g.ID == "evolution.branching") && g.Ready() {
			t.Fatalf("unsupported history completed %s", g.ID)
		}
	}
	for _, id := range targets.eligible {
		state.SeenSpecies[id] = true
	}
	for _, typ := range targets.types {
		state.EncounteredTypes[typ] = true
	}
	state.Forms = fullInventoryState(targets).Forms
	state.SeenVariants = fullInventoryState(targets).SeenVariants
	for _, g := range targets.Goals(state) {
		if g.ID != "encounters.1" && !strings.HasPrefix(g.ID, "encounters.") && !g.Ready() {
			t.Fatalf("complete inventory did not satisfy %s", g.ID)
		}
	}
	// Missing any member suppresses the entire branching family.
	family := targets.families[0]
	species := catalog.All()
	missing := family[len(family)-1]
	reduced, err := NewAchievementTargets(species, func(k catalog.VariantKey) bool {
		if k.SpeciesID == missing {
			return false
		}
		_, ok := sprite.Lookup(k)
		return ok
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range reduced.families {
		for _, id := range f {
			if id == family[0] {
				t.Fatal("unsupported branch was ignored")
			}
		}
	}
}

func TestTargetsMatchGeneratedCoverage(t *testing.T) {
	targets := bundledTargets(t)
	data, err := os.ReadFile("../../tools/dataset/coverage.json")
	if err != nil {
		t.Fatal(err)
	}
	var report struct {
		Inventory struct {
			Types       []string       `json:"types"`
			Families    [][]int        `json:"branching_families"`
			Generations map[string]int `json:"generation_species"`
			National    int            `json:"national_species"`
		} `json:"achievement_inventory"`
	}
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	sort.Slice(report.Inventory.Families, func(i, j int) bool { return report.Inventory.Families[i][0] < report.Inventory.Families[j][0] })
	if !reflect.DeepEqual(targets.types, report.Inventory.Types) || !reflect.DeepEqual(targets.families, report.Inventory.Families) || len(targets.eligible) != report.Inventory.National {
		t.Fatalf("runtime targets differ: types %v/%v families %v/%v national %d/%d", targets.types, report.Inventory.Types, targets.families, report.Inventory.Families, len(targets.eligible), report.Inventory.National)
	}
	for _, gen := range targets.generations {
		if len(gen.species) != report.Inventory.Generations[strconv.Itoa(gen.generation)] {
			t.Fatalf("generation %d target differs", gen.generation)
		}
	}
}

func TestCompletionExcludesRemovedIdentitiesAndCountsExactPalettes(t *testing.T) {
	targets := bundledTargets(t)
	regular := catalog.VariantKey{SpeciesID: 25, FormID: "standard", Gender: "default", Palette: "regular"}
	shiny := regular
	shiny.Palette = "shiny"
	removed := catalog.VariantKey{SpeciesID: 99999, FormID: "standard", Gender: "default", Palette: "regular"}
	c := targets.Completion(map[int]bool{25: true, 99999: true}, []catalog.VariantKey{regular, regular, shiny, removed})
	if c.Species != 1 || c.Variants != 2 || c.SpeciesTotal != 1017 || c.VariantsTotal != 2669 {
		t.Fatalf("eligible completion %+v", c)
	}
}

func fullInventoryState(targets *AchievementTargets) AchievementState {
	state := AchievementState{Encounters: 1000, Species: int64(len(targets.eligible)), Variants: targets.variants, SeenSpecies: map[int]bool{}, EncounteredTypes: map[string]bool{}, Shiny: true, Regional: true, Transformation: true}
	for _, id := range targets.eligible {
		state.SeenSpecies[id] = true
	}
	for _, typ := range targets.types {
		state.EncounteredTypes[typ] = true
	}
	for _, species := range catalog.All() {
		for _, form := range species.Forms {
			if !targets.forms[species.ID][form.ID] {
				continue
			}
			evidence := FormEvidence{SpeciesID: species.ID, FormID: form.ID, Type1: form.Types[0], Regional: slices.Contains(form.Tags, "regional"), Transformation: slices.Contains(form.Tags, "mega") || slices.Contains(form.Tags, "gigantamax")}
			if len(form.Types) == 2 {
				evidence.Type2 = form.Types[1]
			}
			state.Forms = append(state.Forms, evidence)
		}
	}
	for key := range targets.variantKeys {
		state.SeenVariants = append(state.SeenVariants, key)
	}
	return state
}

func TestFiftyAchievementsAreUniqueFeasibleAndKeepOriginalDefinitions(t *testing.T) {
	targets := bundledTargets(t)
	goals := targets.Goals(AchievementState{})
	if len(goals) != 50 || len(targets.Definitions()) != 50 {
		t.Fatalf("achievement count %d/%d", len(goals), len(targets.Definitions()))
	}
	ids := map[string]bool{}
	names := map[string]bool{}
	for _, goal := range goals {
		if ids[goal.ID] || names[goal.Name] || goal.Ready() || goal.Target <= 0 || goal.Name == "" || goal.Description == "" {
			t.Fatalf("invalid empty goal %+v", goal)
		}
		ids[goal.ID] = true
		names[goal.Name] = true
	}
	for _, goal := range targets.Goals(fullInventoryState(targets)) {
		if !goal.Ready() {
			t.Fatalf("unattainable bundled goal %+v", goal)
		}
	}
	expectedNew := map[string]string{"generations.5": "Across Generations", "generations.complete": "World Traveler", "types.8": "Type Sampler", "types.specialist.10": "Type Specialist", "types.dual.10": "Dual-Type Collector", "colors.6": "Rainbow Collection", "stages.complete": "Growing Collection", "regional.species.3": "Regional Explorer", "transformation.species.3": "Transformation Explorer", "forms.nonstandard.5": "Changing Faces", "forms.species.3": "Form Collector", "evolution.family": "Family Reunion", "legendary.first": "Legendary Encounter", "mythical.first": "Mythical Encounter", "baby.first": "Small Beginnings", "shiny.variants.5": "Shiny Collection"}
	for _, goal := range goals {
		if name, ok := expectedNew[goal.ID]; ok {
			if goal.Name != name {
				t.Fatalf("new name %s/%s", goal.Name, name)
			}
			delete(expectedNew, goal.ID)
		}
	}
	if len(expectedNew) != 0 {
		t.Fatalf("missing approved definitions %v", expectedNew)
	}
	originals := []string{}
	for _, n := range []int{1, 10, 25, 50, 100, 250, 500, 1000} {
		originals = append(originals, fmt.Sprintf("encounters.%d", n))
	}
	for _, n := range []int{5, 10, 25, 50, 100, 250, 500} {
		originals = append(originals, fmt.Sprintf("species.%d", n))
	}
	for _, n := range []int{10, 25, 50, 100} {
		originals = append(originals, fmt.Sprintf("variants.%d", n))
	}
	originals = append(originals, "shiny.first", "types.complete", "regional.first", "transformation.first", "evolution.branching")
	for n := 1; n <= 9; n++ {
		originals = append(originals, fmt.Sprintf("generation.%d.complete", n))
	}
	originals = append(originals, "national.complete")
	got := []string{}
	for _, g := range goals {
		if slices.Contains(originals, g.ID) {
			got = append(got, g.ID)
		}
	}
	if !reflect.DeepEqual(got, originals) {
		t.Fatalf("original order changed %v", got)
	}
	defs := targets.Definitions()
	defs[0].Name = "mutated"
	defs[0].Target = 999
	if targets.Definitions()[0].Name != "First Contact" || targets.Definitions()[0].Target != 1 {
		t.Fatal("registry exposes mutable metadata")
	}
	if !reflect.DeepEqual(targets.Goals(fullInventoryState(targets)), targets.Goals(fullInventoryState(targets))) {
		t.Fatal("registry order/progress is unstable")
	}
}

func diverseAchievementFixture(t *testing.T) (*AchievementTargets, []catalog.Species) {
	t.Helper()
	types := []string{"fire", "water", "grass", "electric", "ice", "fighting", "poison", "ground"}
	colors := []string{"red", "blue", "green", "yellow", "brown", "purple", "gray", "white", "black", "pink"}
	var species []catalog.Species
	for id := 1; id <= 11; id++ {
		forms := []catalog.Form{}
		for _, f := range []struct {
			id   string
			tags []string
		}{{"standard", nil}, {"alola", []string{"regional"}}, {"mega", []string{"mega"}}, {"other", nil}} {
			forms = append(forms, catalog.Form{ID: f.id, Name: f.id, Types: []string{"normal", types[(id-1)%len(types)]}, Tags: f.tags, Genders: []string{"male", "female"}, DefaultGender: "male"})
		}
		species = append(species, catalog.Species{ID: id, Name: fmt.Sprintf("Species %d", id), Generation: 1 + (id-1)%9, Color: colors[(id-1)%len(colors)], Stage: 1 + (id-1)%3, Legendary: id == 1, Mythical: id == 2, Baby: id == 3, Forms: forms})
	}
	species[0].EvolvesTo = []int{2}
	species[1].EvolvesFrom = 1
	species[1].EvolvesTo = []int{3}
	species[2].EvolvesFrom = 2
	species[3].EvolvesTo = []int{5, 6}
	species[4].EvolvesFrom = 4
	species[5].EvolvesFrom = 4
	targets, err := NewAchievementTargets(species, func(catalog.VariantKey) bool { return true })
	if err != nil {
		t.Fatal(err)
	}
	return targets, species
}

func goalByID(t *testing.T, targets *AchievementTargets, state AchievementState, id string) Goal {
	t.Helper()
	for _, goal := range targets.Goals(state) {
		if goal.ID == id {
			return goal
		}
	}
	t.Fatalf("missing goal %s", id)
	return Goal{}
}
func fixtureSeen(n int) map[int]bool {
	ids := map[int]bool{}
	for id := 1; id <= n; id++ {
		ids[id] = true
	}
	return ids
}
func fixtureForms(species []catalog.Species, n int, formID string) []FormEvidence {
	forms := []FormEvidence{}
	for _, s := range species[:n] {
		for _, f := range s.Forms {
			if f.ID != formID {
				continue
			}
			forms = append(forms, FormEvidence{SpeciesID: s.ID, FormID: f.ID, Type1: f.Types[0], Type2: f.Types[1], Regional: slices.Contains(f.Tags, "regional"), Transformation: slices.Contains(f.Tags, "mega")})
		}
	}
	return forms
}

func TestApprovedAchievementThresholdsBelowAtAbove(t *testing.T) {
	targets, species := diverseAchievementFixture(t)
	cases := []struct {
		id        string
		threshold int
		state     func(int) AchievementState
	}{
		{"generations.5", 5, func(n int) AchievementState { return AchievementState{SeenSpecies: fixtureSeen(n)} }},
		{"generations.complete", 9, func(n int) AchievementState { return AchievementState{SeenSpecies: fixtureSeen(n)} }},
		{"types.8", 8, func(n int) AchievementState {
			seen := map[string]bool{}
			for _, typ := range targets.types[:n] {
				seen[typ] = true
			}
			return AchievementState{EncounteredTypes: seen}
		}},
		{"types.specialist.10", 10, func(n int) AchievementState { return AchievementState{Forms: fixtureForms(species, n, "standard")} }},
		{"types.dual.10", 10, func(n int) AchievementState { return AchievementState{Forms: fixtureForms(species, n, "standard")} }},
		{"colors.6", 6, func(n int) AchievementState { return AchievementState{SeenSpecies: fixtureSeen(n)} }},
		{"stages.complete", 3, func(n int) AchievementState { return AchievementState{SeenSpecies: fixtureSeen(n)} }},
		{"regional.species.3", 3, func(n int) AchievementState { return AchievementState{Forms: fixtureForms(species, n, "alola")} }},
		{"transformation.species.3", 3, func(n int) AchievementState { return AchievementState{Forms: fixtureForms(species, n, "mega")} }},
		{"forms.nonstandard.5", 5, func(n int) AchievementState { return AchievementState{Forms: fixtureForms(species, n, "alola")} }},
		{"forms.species.3", 3, func(n int) AchievementState {
			state := AchievementState{}
			for _, f := range species[0].Forms[:n] {
				state.Forms = append(state.Forms, FormEvidence{SpeciesID: 1, FormID: f.ID, Type1: f.Types[0], Type2: f.Types[1]})
			}
			return state
		}},
		{"evolution.family", 3, func(n int) AchievementState { return AchievementState{SeenSpecies: fixtureSeen(n)} }},
		{"shiny.variants.5", 5, func(n int) AchievementState {
			state := AchievementState{}
			for id := 1; id <= n; id++ {
				state.SeenVariants = append(state.SeenVariants, catalog.VariantKey{SpeciesID: id, FormID: "standard", Gender: "male", Palette: "shiny"})
			}
			return state
		}},
	}
	for _, c := range cases {
		t.Run(c.id, func(t *testing.T) {
			for _, delta := range []int{-1, 0, 1} {
				g := goalByID(t, targets, c.state(c.threshold+delta), c.id)
				if g.Ready() != (delta >= 0) {
					t.Fatalf("boundary %d: %+v", delta, g)
				}
			}
		})
	}
	for _, c := range []struct {
		id      string
		species int
	}{{"legendary.first", 1}, {"mythical.first", 2}, {"baby.first", 3}} {
		for _, seen := range []map[int]bool{nil, {10: true}, {c.species: true}, {c.species: true, 10: true}} {
			g := goalByID(t, targets, AchievementState{SeenSpecies: seen}, c.id)
			if g.Ready() != seen[c.species] {
				t.Fatalf("source status %s %+v", c.id, g)
			}
		}
	}
}

func TestRepeatedFormsPalettesGendersAndUnknownHistoryDoNotInflateGoals(t *testing.T) {
	targets, species := diverseAchievementFixture(t)
	state := AchievementState{SeenSpecies: map[int]bool{99999: true}, Forms: fixtureForms(species, 1, "alola")}
	state.Forms = append(state.Forms, state.Forms[0], state.Forms[0], FormEvidence{SpeciesID: 99999, FormID: "alola", Type1: "normal", Type2: "fire", Regional: true, Transformation: true}, FormEvidence{SpeciesID: 1, FormID: "unsupported", Type1: "normal", Type2: "fire", Regional: true, Transformation: true})
	for _, id := range []string{"forms.nonstandard.5", "regional.species.3", "forms.species.3", "types.specialist.10", "types.dual.10"} {
		g := goalByID(t, targets, state, id)
		if g.Current != 1 || g.Ready() {
			t.Fatalf("duplicate/unsupported evidence inflated %s %+v", id, g)
		}
	}
	for _, id := range []string{"generations.5", "colors.6", "stages.complete", "legendary.first", "mythical.first", "baby.first"} {
		if g := goalByID(t, targets, state, id); g.Current != 0 {
			t.Fatalf("unknown history inflated %s", id)
		}
	}
	// Type union alone must not invent ten species with a shared or dual type.
	state = AchievementState{SeenSpecies: fixtureSeen(11), EncounteredTypes: map[string]bool{}}
	for _, typ := range targets.types {
		state.EncounteredTypes[typ] = true
	}
	for _, id := range []string{"types.specialist.10", "types.dual.10", "forms.nonstandard.5", "regional.species.3"} {
		if goalByID(t, targets, state, id).Current != 0 {
			t.Fatalf("unencountered forms inferred for %s", id)
		}
	}
	// A dual-type predicate examines one actual form, never a union of two
	// single-type appearances. Both remain actual evidence for Type Specialist.
	state.Forms = []FormEvidence{{SpeciesID: 1, FormID: "standard", Type1: "normal"}, {SpeciesID: 1, FormID: "alola", Type1: "fire"}}
	if goalByID(t, targets, state, "types.dual.10").Current != 0 {
		t.Fatal("two single-type forms became a dual-type encounter")
	}
	key := catalog.VariantKey{SpeciesID: 1, FormID: "alola", Gender: "male", Palette: "shiny"}
	state.SeenVariants = []catalog.VariantKey{key, key, key}
	key.Palette = "regular"
	state.SeenVariants = append(state.SeenVariants, key)
	if goalByID(t, targets, state, "shiny.variants.5").Current != 1 {
		t.Fatal("shiny repeats/regular palette inflated collection")
	}
}

func TestImpossibleGoalsSuppressedAndDefinitionsRetained(t *testing.T) {
	_, species := diverseAchievementFixture(t)
	// Leave only regular, standard artwork of two ordinary species in one
	// generation/color/stage; catalog evolution families still have missing members.
	for i := range species {
		species[i].Legendary = false
		species[i].Mythical = false
		species[i].Baby = false
		species[i].Generation = 1
		species[i].Color = "red"
		species[i].Stage = 1
		for j := range species[i].Forms {
			species[i].Forms[j].Types = []string{"normal"}
		}
	}
	targets, err := NewAchievementTargets(species, func(k catalog.VariantKey) bool {
		return k.SpeciesID <= 2 && k.FormID == "standard" && k.Palette == "regular"
	})
	if err != nil {
		t.Fatal(err)
	}
	absent := []string{"generations.5", "types.8", "types.specialist.10", "types.dual.10", "colors.6", "regional.species.3", "transformation.species.3", "forms.nonstandard.5", "forms.species.3", "evolution.family", "legendary.first", "mythical.first", "baby.first", "shiny.variants.5"}
	for _, goal := range targets.Goals(AchievementState{}) {
		if slices.Contains(absent, goal.ID) {
			t.Fatalf("impossible goal %+v", goal)
		}
	}
	defs := targets.Definitions()
	for _, id := range absent {
		found := false
		for _, d := range defs {
			if d.ID == id {
				found = true
			}
		}
		if !found {
			t.Fatalf("retained unlock cannot be named: %s", id)
		}
	}
	empty, err := NewAchievementTargets(species, func(catalog.VariantKey) bool { return false })
	if err != nil {
		t.Fatal(err)
	}
	if len(empty.Goals(AchievementState{})) != 0 {
		t.Fatal("empty inventory exposes unattainable goals")
	}
}
