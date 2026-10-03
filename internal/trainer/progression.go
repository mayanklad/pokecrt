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

// FormEvidence contains only facts actually recorded by an encounter. Names and
// undiscovered catalog forms are deliberately absent from achievement progress.
type FormEvidence struct {
	SpeciesID                int
	FormID, Type1, Type2     string
	Regional, Transformation bool
}

type AchievementState struct {
	Forms                           []FormEvidence
	SeenVariants                    []catalog.VariantKey
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

// Completion separates current eligible coverage from retained historical counts.
type Completion struct{ Species, SpeciesTotal, Variants, VariantsTotal int64 }

type achievementRule struct {
	definition Goal
	metric     string
	available  bool
}
type speciesFacts struct {
	stage                     int
	color                     string
	legendary, mythical, baby bool
}

type AchievementTargets struct {
	rules                           []achievementRule
	facts                           map[int]speciesFacts
	forms                           map[int]map[string]bool
	colors                          map[string]bool
	stages                          map[int]bool
	reunionFamilies                 [][]int
	typeCapacity                    map[string]map[int]bool
	dualCapacity                    map[int]bool
	alternateCapacity               map[int]bool
	regionalCapacity                map[int]bool
	transformationCapacity          map[int]bool
	shinyCapacity                   int64
	legendary, mythical, baby       bool
	variantKeys                     map[catalog.VariantKey]bool
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
	t := &AchievementTargets{
		variants: int64(pool.VariantCount()), variantKeys: make(map[catalog.VariantKey]bool, pool.VariantCount()),
		facts: map[int]speciesFacts{}, forms: map[int]map[string]bool{}, colors: map[string]bool{}, stages: map[int]bool{},
		typeCapacity: map[string]map[int]bool{}, dualCapacity: map[int]bool{}, alternateCapacity: map[int]bool{},
		regionalCapacity: map[int]bool{}, transformationCapacity: map[int]bool{},
	}
	eligible := map[int]bool{}
	types := map[string]bool{}
	for _, s := range pool.species {
		eligible[s.id] = true
		t.eligible = append(t.eligible, s.id)
		t.forms[s.id] = map[string]bool{}
		for _, f := range s.forms {
			t.forms[s.id][f.id] = true
			if f.id != "standard" {
				t.alternateCapacity[s.id] = true
			}
			for _, g := range f.genders {
				key := g.regular.Key()
				t.variantKeys[key] = true
				if g.shiny {
					key.Palette = "shiny"
					t.variantKeys[key] = true
					t.shinyCapacity++
				}
				snapshot := g.regular.Snapshot()
				types[snapshot.Type1] = true
				for _, typ := range []string{snapshot.Type1, snapshot.Type2} {
					if typ == "" {
						continue
					}
					if t.typeCapacity[typ] == nil {
						t.typeCapacity[typ] = map[int]bool{}
					}
					t.typeCapacity[typ][s.id] = true
				}
				if snapshot.Type2 != "" {
					t.dualCapacity[s.id] = true
				}
				if snapshot.Regional {
					t.regionalCapacity[s.id] = true
				}
				if snapshot.Transformation {
					t.transformationCapacity[s.id] = true
				}
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
		if _, ok := byGen[s.Generation]; !ok {
			byGen[s.Generation] = nil
		}
		if eligible[s.ID] {
			if s.Generation <= 0 {
				return nil, fmt.Errorf("invalid introduction generation for species %d", s.ID)
			}
			byGen[s.Generation] = append(byGen[s.Generation], s.ID)
			t.facts[s.ID] = speciesFacts{stage: s.Stage, color: s.Color, legendary: s.Legendary, mythical: s.Mythical, baby: s.Baby}
			if s.Color != "" {
				t.colors[s.Color] = true
			}
			if s.Stage > 0 {
				t.stages[s.Stage] = true
			}
			t.legendary = t.legendary || s.Legendary
			t.mythical = t.mythical || s.Mythical
			t.baby = t.baby || s.Baby
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
		if complete {
			sort.Ints(family)
			if branching {
				t.families = append(t.families, family)
			}
			if len(family) >= 3 {
				t.reunionFamilies = append(t.reunionFamilies, family)
			}
		}
	}
	t.buildRegistry()
	return t, nil
}

// buildRegistry is the single ordered source for stable IDs, names, descriptions
// and policy targets. Unsupported rules remain definable for retained unlocks,
// but Goals never exposes an impossible current goal.
func (t *AchievementTargets) buildRegistry() {
	add := func(id, name, description, metric string, target int64, hasTarget, available bool) {
		t.rules = append(t.rules, achievementRule{Goal{ID: id, Name: name, Description: description, Target: target, HasTarget: hasTarget}, metric, available})
	}
	for _, n := range []int64{1, 10, 25, 50, 100, 250, 500, 1000} {
		name := fmt.Sprintf("%d Encounters", n)
		if n == 1 {
			name = "First Contact"
		}
		add(fmt.Sprintf("encounters.%d", n), name, fmt.Sprintf("Recorded %d encounters.", n), "encounters", n, true, len(t.eligible) > 0)
	}
	for _, n := range []int64{5, 10, 25, 50, 100, 250, 500} {
		add(fmt.Sprintf("species.%d", n), fmt.Sprintf("%d Species Discovered", n), fmt.Sprintf("Discovered %d distinct species.", n), "species", n, true, n <= int64(len(t.eligible)))
	}
	for _, n := range []int64{10, 25, 50, 100} {
		add(fmt.Sprintf("variants.%d", n), fmt.Sprintf("%d Variants Collected", n), fmt.Sprintf("Collected %d distinct visual variants.", n), "variants", n, true, n <= t.variants)
	}
	// Original themed definitions retain their IDs and relative display order.
	add("shiny.first", "Shiny Discovery", "Record a shiny encounter.", "shiny.first", 1, false, t.shiny)
	add("types.complete", "Type Explorer", "Encounter forms covering all supported types.", "types", int64(len(t.types)), true, len(t.types) > 0)
	add("regional.first", "Regional Discovery", "Encounter a regional form.", "regional.first", 1, false, t.regional)
	add("transformation.first", "Transformation Discovery", "Encounter a supported Mega or Gigantamax form.", "transformation.first", 1, false, t.transformation)
	add("evolution.branching", "Branching Out", "Discover every eligible species in a fully supported branching evolution family.", "evolution.branching", 0, true, len(t.families) > 0)
	generations := int64(0)
	for _, gen := range t.generations {
		if len(gen.species) > 0 {
			generations++
		}
	}
	specialistCapacity := 0
	for _, ids := range t.typeCapacity {
		specialistCapacity = max(specialistCapacity, len(ids))
	}
	formCapacity := 0
	for _, forms := range t.forms {
		formCapacity = max(formCapacity, len(forms))
	}
	add("generations.5", "Across Generations", "Discover species from five different generations.", "generations", 5, true, generations >= 5)
	add("generations.complete", "World Traveler", "Discover a species from every supported generation.", "generations", generations, true, generations > 0)
	add("types.8", "Type Sampler", "Encounter forms covering eight distinct types.", "types", 8, true, len(t.types) >= 8)
	add("types.specialist.10", "Type Specialist", "Discover ten species through encounters sharing one type.", "types.specialist.10", 10, true, specialistCapacity >= 10)
	add("types.dual.10", "Dual-Type Collector", "Discover ten species through dual-type form encounters.", "types.dual.10", 10, true, len(t.dualCapacity) >= 10)
	add("colors.6", "Rainbow Collection", "Discover species covering six distinct Pokédex colors.", "colors.6", 6, true, len(t.colors) >= 6)
	add("stages.complete", "Growing Collection", "Discover a species at every supported evolution stage.", "stages.complete", int64(len(t.stages)), true, len(t.stages) > 0)
	add("regional.species.3", "Regional Explorer", "Encounter regional forms of three distinct species.", "regional.species.3", 3, true, len(t.regionalCapacity) >= 3)
	add("transformation.species.3", "Transformation Explorer", "Encounter Mega or Gigantamax forms of three distinct species.", "transformation.species.3", 3, true, len(t.transformationCapacity) >= 3)
	add("forms.nonstandard.5", "Changing Faces", "Encounter nonstandard forms of five distinct species.", "forms.nonstandard.5", 5, true, len(t.alternateCapacity) >= 5)
	add("forms.species.3", "Form Collector", "Collect three distinct forms of one species.", "forms.species.3", 3, true, formCapacity >= 3)
	add("evolution.family", "Family Reunion", "Discover every species in a fully supported evolution family of at least three species.", "evolution.family", 0, true, len(t.reunionFamilies) > 0)
	add("legendary.first", "Legendary Encounter", "Encounter a species marked legendary by the source metadata.", "legendary.first", 1, false, t.legendary)
	add("mythical.first", "Mythical Encounter", "Encounter a species marked mythical by the source metadata.", "mythical.first", 1, false, t.mythical)
	add("baby.first", "Small Beginnings", "Encounter a species marked baby by the source metadata.", "baby.first", 1, false, t.baby)
	add("shiny.variants.5", "Shiny Collection", "Collect five distinct shiny variants.", "shiny.variants.5", 5, true, t.shinyCapacity >= 5)
	for _, gen := range t.generations {
		add(fmt.Sprintf("generation.%d.complete", gen.generation), fmt.Sprintf("Generation %d Researcher", gen.generation), fmt.Sprintf("Discover every eligible species introduced in generation %d.", gen.generation), fmt.Sprintf("generation.%d.complete", gen.generation), int64(len(gen.species)), true, len(gen.species) > 0)
	}
	add("national.complete", "National Researcher", "Discover every species in the current eligible inventory.", "national.complete", int64(len(t.eligible)), true, len(t.eligible) > 0)
}

// Definitions returns caller-owned metadata, including unavailable policy rules
// so previously earned IDs can still be named after inventory changes.
func (t *AchievementTargets) Definitions() []Goal {
	if t == nil {
		return nil
	}
	defs := make([]Goal, len(t.rules))
	for i, rule := range t.rules {
		defs[i] = rule.definition
	}
	return defs
}

type achievementProgress struct{ current, target int64 }

func (t *AchievementTargets) Goals(state AchievementState) []Goal {
	if t == nil {
		return nil
	}
	progress := t.achievementProgress(state)
	goals := make([]Goal, 0, len(t.rules))
	for _, rule := range t.rules {
		if !rule.available {
			continue
		}
		goal := rule.definition
		value := progress[rule.metric]
		goal.Current = value.current
		if value.target > 0 {
			goal.Target = value.target
		}
		goals = append(goals, goal)
	}
	return goals
}

func (t *AchievementTargets) achievementProgress(state AchievementState) map[string]achievementProgress {
	result := map[string]achievementProgress{"encounters": {current: state.Encounters}, "species": {current: state.Species}, "variants": {current: state.Variants}}
	flag := func(id string, ready bool) {
		if ready {
			result[id] = achievementProgress{current: 1}
		}
	}
	flag("shiny.first", state.Shiny)
	flag("regional.first", state.Regional)
	flag("transformation.first", state.Transformation)
	types := int64(0)
	for _, typ := range t.types {
		if state.EncounteredTypes[typ] {
			types++
		}
	}
	result["types"] = achievementProgress{current: types}
	generations := int64(0)
	for _, gen := range t.generations {
		n := countSeen(gen.species, state.SeenSpecies)
		if n > 0 {
			generations++
		}
		result[fmt.Sprintf("generation.%d.complete", gen.generation)] = achievementProgress{current: n}
	}
	result["generations"] = achievementProgress{current: generations}
	result["national.complete"] = achievementProgress{current: countSeen(t.eligible, state.SeenSpecies)}
	colors := map[string]bool{}
	stages := map[int]bool{}
	for id, facts := range t.facts {
		if !state.SeenSpecies[id] {
			continue
		}
		if facts.color != "" {
			colors[facts.color] = true
		}
		if facts.stage > 0 {
			stages[facts.stage] = true
		}
		flag("legendary.first", facts.legendary)
		flag("mythical.first", facts.mythical)
		flag("baby.first", facts.baby)
	}
	result["colors.6"] = achievementProgress{current: int64(len(colors))}
	result["stages.complete"] = achievementProgress{current: int64(len(stages))}
	typeSpecies := map[string]map[int]bool{}
	dual, regional, transformation, alternate := map[int]bool{}, map[int]bool{}, map[int]bool{}, map[int]bool{}
	seenForms := map[int]map[string]bool{}
	for _, form := range state.Forms {
		if !t.forms[form.SpeciesID][form.FormID] {
			continue
		}
		if seenForms[form.SpeciesID] == nil {
			seenForms[form.SpeciesID] = map[string]bool{}
		}
		seenForms[form.SpeciesID][form.FormID] = true
		if form.FormID != "standard" {
			alternate[form.SpeciesID] = true
		}
		if form.Regional {
			regional[form.SpeciesID] = true
		}
		if form.Transformation {
			transformation[form.SpeciesID] = true
		}
		if form.Type2 != "" && form.Type2 != form.Type1 {
			dual[form.SpeciesID] = true
		}
		for _, typ := range []string{form.Type1, form.Type2} {
			if _, supported := t.typeCapacity[typ]; !supported {
				continue
			}
			if typeSpecies[typ] == nil {
				typeSpecies[typ] = map[int]bool{}
			}
			typeSpecies[typ][form.SpeciesID] = true
		}
	}
	specialist := 0
	for _, ids := range typeSpecies {
		specialist = max(specialist, len(ids))
	}
	formCount := 0
	for _, forms := range seenForms {
		formCount = max(formCount, len(forms))
	}
	result["types.specialist.10"] = achievementProgress{current: int64(specialist)}
	result["types.dual.10"] = achievementProgress{current: int64(len(dual))}
	result["regional.species.3"] = achievementProgress{current: int64(len(regional))}
	result["transformation.species.3"] = achievementProgress{current: int64(len(transformation))}
	result["forms.nonstandard.5"] = achievementProgress{current: int64(len(alternate))}
	result["forms.species.3"] = achievementProgress{current: int64(formCount)}
	result["evolution.branching"] = bestFamilyProgress(t.families, state.SeenSpecies)
	result["evolution.family"] = bestFamilyProgress(t.reunionFamilies, state.SeenSpecies)
	shiny := map[catalog.VariantKey]bool{}
	for _, key := range state.SeenVariants {
		if key.Palette == "shiny" {
			shiny[key] = true
		}
	}
	result["shiny.variants.5"] = achievementProgress{current: int64(len(shiny))}
	return result
}

func bestFamilyProgress(families [][]int, seen map[int]bool) achievementProgress {
	best := achievementProgress{}
	for _, family := range families {
		n, target := countSeen(family, seen), int64(len(family))
		if best.target == 0 || n*best.target > best.current*target || (n*best.target == best.current*target && n > best.current) {
			best = achievementProgress{n, target}
		}
	}
	return best
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

// Completion counts only identities still eligible in this inventory. Duplicate
// input variants do not inflate coverage; neither input collection is retained.
func (t *AchievementTargets) Completion(seenSpecies map[int]bool, seenVariants []catalog.VariantKey) Completion {
	c := Completion{Species: countSeen(t.eligible, seenSpecies), SpeciesTotal: int64(len(t.eligible)), VariantsTotal: t.variants}
	counted := make(map[catalog.VariantKey]bool, len(seenVariants))
	for _, key := range seenVariants {
		if t.variantKeys[key] && !counted[key] {
			c.Variants++
			counted[key] = true
		}
	}
	return c
}
