package trainer

import (
	"encoding/json"
	"github.com/mayanklad/pokecrt/internal/catalog"
	"github.com/mayanklad/pokecrt/internal/sprite"
	"math"
	"os"
	"reflect"
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
