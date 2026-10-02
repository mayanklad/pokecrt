package trainer

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"sort"

	"github.com/mayanklad/pokecrt/internal/catalog"
)

type Progress struct{ Total, Level, InLevel, ToNext int64 }

func XPProgress(total int64) (Progress, error) {
	if total < 0 {
		return Progress{}, errors.New("trainer XP cannot be negative")
	}
	return Progress{Total: total, Level: 1 + total/1000, InLevel: total % 1000, ToNext: 1000 - total%1000}, nil
}

func AwardXP(firstSpecies, firstVariant, shiny bool) int64 {
	award := int64(10)
	if firstSpecies {
		award += 40
	}
	if firstVariant {
		award += 20
	}
	if shiny {
		award += 100
	}
	return award
}

func AdvanceXP(total, award int64) (Progress, error) {
	if total < 0 || award < 0 || total > math.MaxInt64-award {
		return Progress{}, errors.New("trainer XP overflow or invalid value")
	}
	return XPProgress(total + award)
}

type AchievementState struct {
	Encounters, Species, Variants   int64
	SeenSpecies                     map[int]bool
	EncounteredTypes                map[string]bool
	Shiny, Regional, Transformation bool
}

type Goal struct {
	ID, Name, Description string
	Current, Target       int64
	HasTarget             bool
}

func (g Goal) Ready() bool { return g.Target > 0 && g.Current >= g.Target }

type Unlock struct {
	ID, Name, Description, DatasetID string
	UnlockedAtMS, Target             int64
	HasTarget                        bool
}

type generationTarget struct {
	generation int
	species    []int
}
type AchievementTargets struct {
	eligible                        []int
	generations                     []generationTarget
	types                           []string
	families                        [][]int
	variants                        int64
	shiny, regional, transformation bool
}

// NewAchievementTargets derives completion requirements from current exact
// regular artwork and catalog relationships. It owns all values it retains.
func NewAchievementTargets(species []catalog.Species, available func(catalog.VariantKey) bool) (*AchievementTargets, error) {
	pool, err := NewPool(species, available)
	if err != nil {
		return nil, err
	}
	t := &AchievementTargets{variants: int64(pool.VariantCount())}
	eligible := map[int]bool{}
	types := map[string]bool{}
	for _, s := range pool.species {
		eligible[s.id] = true
		t.eligible = append(t.eligible, s.id)
		for _, f := range s.forms {
			for _, g := range f.genders {
				snapshot := g.regular.Snapshot()
				types[snapshot.Type1] = true
				if snapshot.Type2 != "" {
					types[snapshot.Type2] = true
				}
				t.shiny = t.shiny || g.shiny
				t.regional = t.regional || snapshot.Regional
				t.transformation = t.transformation || snapshot.Transformation
			}
		}
	}
	for typ := range types {
		t.types = append(t.types, typ)
	}
	sort.Strings(t.types)
	byGen := map[int][]int{}
	byID := map[int]catalog.Species{}
	edges := map[int][]int{}
	for _, s := range species {
		byID[s.ID] = s
		if eligible[s.ID] {
			if s.Generation <= 0 {
				return nil, fmt.Errorf("invalid introduction generation for species %d", s.ID)
			}
			byGen[s.Generation] = append(byGen[s.Generation], s.ID)
		}
	}
	for _, s := range species {
		for _, child := range s.EvolvesTo {
			if _, ok := byID[child]; !ok {
				return nil, fmt.Errorf("missing evolution species %d", child)
			}
			if !slices.Contains(edges[s.ID], child) {
				edges[s.ID] = append(edges[s.ID], child)
			}
			if !slices.Contains(edges[child], s.ID) {
				edges[child] = append(edges[child], s.ID)
			}
		}
		if s.EvolvesFrom != 0 {
			if _, ok := byID[s.EvolvesFrom]; !ok {
				return nil, fmt.Errorf("missing evolution parent %d", s.EvolvesFrom)
			}
			if !slices.Contains(edges[s.ID], s.EvolvesFrom) {
				edges[s.ID] = append(edges[s.ID], s.EvolvesFrom)
			}
			if !slices.Contains(edges[s.EvolvesFrom], s.ID) {
				edges[s.EvolvesFrom] = append(edges[s.EvolvesFrom], s.ID)
			}
		}
	}
	for gen, ids := range byGen {
		sort.Ints(ids)
		t.generations = append(t.generations, generationTarget{gen, ids})
	}
	sort.Slice(t.generations, func(i, j int) bool { return t.generations[i].generation < t.generations[j].generation })
	// Each entire connected catalog family must be eligible. A removed branch
	// cannot turn an incomplete family into a fake completion goal.
	visited := map[int]bool{}
	ids := make([]int, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, root := range ids {
		if visited[root] {
			continue
		}
		queue := []int{root}
		family := []int{}
		complete, branching := true, false
		for len(queue) > 0 {
			id := queue[0]
			queue = queue[1:]
			if visited[id] {
				continue
			}
			visited[id] = true
			family = append(family, id)
			complete = complete && eligible[id]
			children := map[int]bool{}
			for _, child := range byID[id].EvolvesTo {
				children[child] = true
			}
			// Parent relationships also provide child edges in normalized graph data.
			for _, neighbor := range edges[id] {
				if byID[neighbor].EvolvesFrom == id {
					children[neighbor] = true
				}
			}
			branching = branching || len(children) >= 2
			queue = append(queue, edges[id]...)
		}
		if complete && branching {
			sort.Ints(family)
			t.families = append(t.families, family)
		}
	}
	return t, nil
}

func (t *AchievementTargets) Goals(state AchievementState) []Goal {
	if t == nil {
		return nil
	}
	goals := []Goal{}
	for _, n := range []int64{1, 10, 25, 50, 100, 250, 500, 1000} {
		name := fmt.Sprintf("%d Encounters", n)
		if n == 1 {
			name = "First Contact"
		}
		goals = append(goals, Goal{ID: fmt.Sprintf("encounters.%d", n), Name: name, Description: fmt.Sprintf("Recorded %d encounters.", n), Current: state.Encounters, Target: n, HasTarget: true})
	}
	for _, n := range []int64{5, 10, 25, 50, 100, 250, 500} {
		if n > int64(len(t.eligible)) {
			continue
		}
		goals = append(goals, Goal{ID: fmt.Sprintf("species.%d", n), Name: fmt.Sprintf("%d Species Discovered", n), Description: fmt.Sprintf("Discovered %d distinct species.", n), Current: state.Species, Target: n, HasTarget: true})
	}
	for _, n := range []int64{10, 25, 50, 100} {
		if n > t.variants {
			continue
		}
		goals = append(goals, Goal{ID: fmt.Sprintf("variants.%d", n), Name: fmt.Sprintf("%d Variants Collected", n), Description: fmt.Sprintf("Collected %d distinct visual variants.", n), Current: state.Variants, Target: n, HasTarget: true})
	}
	flag := func(id, name, description string, ready bool) Goal {
		n := int64(0)
		if ready {
			n = 1
		}
		return Goal{ID: id, Name: name, Description: description, Current: n, Target: 1}
	}
	if t.shiny {
		goals = append(goals, flag("shiny.first", "Shiny Discovery", "Record a shiny encounter.", state.Shiny))
	}
	if len(t.types) > 0 {
		count := int64(0)
		for _, typ := range t.types {
			if state.EncounteredTypes[typ] {
				count++
			}
		}
		goals = append(goals, Goal{ID: "types.complete", Name: "Type Explorer", Description: "Encounter forms covering all supported types.", Current: count, Target: int64(len(t.types)), HasTarget: true})
	}
	if t.regional {
		goals = append(goals, flag("regional.first", "Regional Discovery", "Encounter a regional form.", state.Regional))
	}
	if t.transformation {
		goals = append(goals, flag("transformation.first", "Transformation Discovery", "Encounter a supported Mega or Gigantamax form.", state.Transformation))
	}
	if len(t.families) > 0 {
		bestCurrent, bestTarget := int64(0), int64(len(t.families[0]))
		for _, family := range t.families {
			count := countSeen(family, state.SeenSpecies)
			target := int64(len(family))
			// Best fraction, ties by discovered count; never return family identities.
			if count*bestTarget > bestCurrent*target || (count*bestTarget == bestCurrent*target && count > bestCurrent) {
				bestCurrent, bestTarget = count, target
			}
		}
		goals = append(goals, Goal{ID: "evolution.branching", Name: "Branching Out", Description: "Discover every eligible species in a fully supported branching evolution family.", Current: bestCurrent, Target: bestTarget, HasTarget: true})
	}
	for _, gen := range t.generations {
		goals = append(goals, Goal{ID: fmt.Sprintf("generation.%d.complete", gen.generation), Name: fmt.Sprintf("Generation %d Researcher", gen.generation), Description: fmt.Sprintf("Discover every eligible species introduced in generation %d.", gen.generation), Current: countSeen(gen.species, state.SeenSpecies), Target: int64(len(gen.species)), HasTarget: true})
	}
	if len(t.eligible) > 0 {
		goals = append(goals, Goal{ID: "national.complete", Name: "National Researcher", Description: "Discover every species in the current eligible inventory.", Current: countSeen(t.eligible, state.SeenSpecies), Target: int64(len(t.eligible)), HasTarget: true})
	}
	return goals
}

func countSeen(ids []int, seen map[int]bool) int64 {
	n := int64(0)
	for _, id := range ids {
		if seen[id] {
			n++
		}
	}
	return n
}
