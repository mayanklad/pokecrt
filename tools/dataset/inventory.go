package main

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"
)

// Selection describes development scope, not a manually enumerated catalog.
type inventorySelection struct {
	Generations         []int `json:"generations"`
	FamilySeeds         []int `json:"family_seeds"`
	VisualGenderSpecies []int `json:"visual_gender_species"`
}

func validateSelection(m mappingConfig) error {
	if (m.RulesVersion != "d06-auto-1" && m.RulesVersion != "d06-auto-2") || m.Selection == nil || m.SourceStandardForm != "base" || m.StandardFormReason == "" || len(m.Selection.Generations)+len(m.Selection.FamilySeeds) == 0 {
		return fmt.Errorf("automatic inventory requires a supported policy and selection")
	}
	if len(m.CatalogSpecies)+len(m.Forms)+len(m.Assets)+len(m.Exclusions) != 0 {
		return fmt.Errorf("automatic selection cannot contain maintained catalog, forms, assets or exclusions")
	}
	for _, values := range [][]int{m.Selection.Generations, m.Selection.FamilySeeds, m.Selection.VisualGenderSpecies} {
		seen := map[int]bool{}
		for _, id := range values {
			if id <= 0 || seen[id] {
				return fmt.Errorf("invalid or duplicate selection ID %d", id)
			}
			seen[id] = true
		}
	}
	return nil
}

// selectedSpecies computes full connected evolution families from pinned edges.
func selectedSpecies(rows []map[string]string, policy inventorySelection) ([]int, error) {
	parents := map[int]int{}
	generations := map[int]int{}
	selected := map[int]bool{}
	for _, r := range rows {
		id, e := integer(r, "id", false)
		if e != nil {
			return nil, e
		}
		parent, e := integer(r, "evolves_from_species_id", true)
		if e != nil {
			return nil, e
		}
		gen, e := integer(r, "generation_id", false)
		if e != nil {
			return nil, e
		}
		if _, ok := parents[id]; ok {
			return nil, fmt.Errorf("duplicate species %d", id)
		}
		parents[id] = parent
		generations[id] = gen
		if slices.Contains(policy.Generations, gen) {
			selected[id] = true
		}
	}
	if _, e := evolutionStages(parents); e != nil {
		return nil, e
	}
	for _, gen := range policy.Generations {
		found := false
		for _, v := range generations {
			if v == gen {
				found = true
			}
		}
		if !found {
			return nil, fmt.Errorf("unknown selected generation %d", gen)
		}
	}
	for _, id := range policy.FamilySeeds {
		if _, ok := parents[id]; !ok {
			return nil, fmt.Errorf("unknown family seed %d", id)
		}
		selected[id] = true
	}
	for changed := true; changed; {
		changed = false
		for id, parent := range parents {
			if parent != 0 && (selected[id] || selected[parent]) {
				if !selected[id] || !selected[parent] {
					changed = true
				}
				selected[id] = true
				selected[parent] = true
			}
		}
	}
	var ids []int
	for id := range selected {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids, nil
}

func deriveMappings(lock sourceLock, cache string, m mappingConfig) (mappingConfig, error) {
	if m.Selection == nil {
		return m, nil
	} // Existing fixture schemas remain supported.
	if err := validateSelection(m); err != nil {
		return m, err
	}
	rows, err := readTable(lock, cache, "pokemon_species.csv", "id", "generation_id", "evolves_from_species_id")
	if err != nil {
		return m, err
	}
	ids, err := selectedSpecies(rows, *m.Selection)
	if err != nil {
		return m, err
	}
	data, _, _, err := readInput(lock, cache, "pokesprite-v2", "data/pokemon.json")
	if err != nil {
		return m, err
	}
	var manifest sourceManifest
	if err = json.Unmarshal(data, &manifest); err != nil {
		return m, err
	}
	byID := map[int]sourceSpecies{}
	for _, sp := range manifest.Pokemon {
		if _, ok := byID[sp.ID]; ok {
			return m, fmt.Errorf("duplicate manifest species %d", sp.ID)
		}
		byID[sp.ID] = sp
	}
	varieties, err := readTable(lock, cache, "pokemon.csv", "id", "identifier", "species_id", "is_default")
	if err != nil {
		return m, err
	}
	defaults := map[int]int{}
	names := map[string]int{}
	owners := map[int]int{}
	for _, r := range varieties {
		id, e := integer(r, "id", false)
		if e != nil {
			return m, e
		}
		owner, e := integer(r, "species_id", false)
		if e != nil {
			return m, e
		}
		if names[r["identifier"]] != 0 || owners[id] != 0 {
			return m, fmt.Errorf("duplicate variety identity")
		}
		names[r["identifier"]] = id
		owners[id] = owner
		yes, e := sourceBool(r, "is_default")
		if e != nil {
			return m, e
		}
		if yes {
			if defaults[owner] != 0 {
				return m, fmt.Errorf("duplicate default variety")
			}
			defaults[owner] = id
		}
	}
	// Exact metadata form identifiers resolve cosmetic states such as Unown.
	// Optional only for legacy source locks and small test fixtures.
	foundFormTable := false
	for _, src := range lock.Sources {
		if src.ID != "pokeapi" {
			continue
		}
		for _, file := range src.Files {
			if file.Path != "data/v2/csv/pokemon_forms.csv" {
				continue
			}
			foundFormTable = true
			forms, e := readTable(lock, cache, "pokemon_forms.csv", "identifier", "pokemon_id")
			if e != nil {
				return m, e
			}
			seen := map[string]bool{}
			for _, row := range forms {
				key := row["identifier"]
				id, e := integer(row, "pokemon_id", false)
				if e != nil || key == "" || seen[key] || owners[id] == 0 {
					return m, fmt.Errorf("invalid or duplicate metadata form %s", key)
				}
				seen[key] = true
				if existing := names[key]; existing != 0 && existing != id {
					return m, fmt.Errorf("conflicting variety/form identity %s", key)
				}
				names[key] = id
			}
		}
	}
	if m.RulesVersion == "d06-auto-2" && !foundFormTable {
		return m, fmt.Errorf("automatic rules v2 require pinned pokemon_forms.csv")
	}
	inherited, err := loadInheritedInventory(lock, cache)
	if err != nil {
		return m, err
	}
	overrides := map[string]formMapping{}
	used := map[string]bool{}
	for _, f := range m.FormOverrides {
		key := fmt.Sprintf("%d/%s", f.SpeciesID, f.SourceFormID)
		if _, ok := overrides[key]; ok || f.Reason == "" {
			return m, fmt.Errorf("invalid duplicate form exception %s", key)
		}
		overrides[key] = f
	}
	m.CatalogSpecies = ids
	m.Exclusions = []string{"Scope is selected generations and complete connected evolution families; wider coverage remains pending.", "Distinct visual genders are limited to the explicitly reviewed policy scope; other candidates remain pending audit."}
	for _, id := range ids {
		sp, ok := byID[id]
		if !ok {
			return m, fmt.Errorf("missing source species %d", id)
		}
		if sp.DefaultForm == "" || defaults[id] == 0 {
			return m, fmt.Errorf("unresolved standard identity %d", id)
		}
		gendered := slices.Contains(m.Selection.VisualGenderSpecies, id)
		for _, sf := range sp.Forms {
			if sf.CanonicalForm != nil {
				continue
			}
			formID, name := sf.ID, sf.Label
			pokemonID := names[sp.Slug+"-"+sf.ID]
			if sf.ID == sp.DefaultForm {
				formID = "standard"
				name = "Standard"
				if sf.ID != "base" && pokemonID != defaults[id] {
					return m, fmt.Errorf("source default does not match metadata default %d/%s", id, sf.ID)
				}
				pokemonID = defaults[id]
			}
			f := formMapping{SpeciesID: id, ID: formID, Name: name, SourceFormID: sf.ID, PokemonID: pokemonID, DefaultGender: "default", Genders: []string{"default"}, Reason: "Derived from pinned source identity and exact same-species PokéAPI variety."}
			key := fmt.Sprintf("%d/%s", id, sf.ID)
			if override, ok := overrides[key]; ok {
				if pokemonID != 0 {
					return m, fmt.Errorf("unnecessary exception %s", key)
				}
				f = override
				used[key] = true
			}
			if f.ID != formID || f.SourceFormID != sf.ID || owners[f.PokemonID] != id {
				return m, fmt.Errorf("unresolved or wrong-owner variety %s; add a reasoned exception", key)
			}
			for _, alias := range sp.Forms {
				if alias.CanonicalForm != nil && *alias.CanonicalForm == sf.ID {
					if alias.Slug != sf.Slug {
						return m, fmt.Errorf("alias artwork mismatch %d/%s", id, alias.ID)
					}
					f.SourceAliases = append(f.SourceAliases, alias.ID)
				}
			}
			if gendered && sf.ID == "base" {
				f.DefaultGender = "male"
				f.Genders = []string{"male", "female"}
			}
			m.Forms = append(m.Forms, f)
			a := assetMapping{SpeciesID: id, SourceID: "pokesprite-v2", SourceSlug: sf.Slug, FormID: f.ID, SourceFormID: sf.ID, Gender: f.DefaultGender, Palette: "regular", Reason: "Derived from pinned manifest; audited inherited provider and quality flags required."}
			if sf.IsGenerated == nil || *sf.IsGenerated || sf.Source != "msikma/pokesprite" {
				m.Exclusions = append(m.Exclusions, fmt.Sprintf("#%03d/%s: provider or generated status is outside the audited inherited policy.", id, f.ID))
				continue
			}
			if _, err := inherited.verify(a, sp.Slug); err != nil {
				m.Exclusions = append(m.Exclusions, fmt.Sprintf("#%03d/%s: inherited appearance flags are missing or provisional; artwork excluded.", id, f.ID))
				continue
			}
			if sf.HasRegular == nil || !*sf.HasRegular {
				m.Exclusions = append(m.Exclusions, fmt.Sprintf("#%03d/%s: regular artwork unavailable; shiny-only slots excluded.", id, f.ID))
				continue
			}
			for _, palette := range []string{"regular", "shiny"} {
				if palette == "shiny" && (sf.HasShiny == nil || !*sf.HasShiny) {
					continue
				}
				a.Palette = palette
				a.Path, _ = assetSourcePath(a)
				m.Assets = append(m.Assets, a)
				if gendered && sf.ID == "base" {
					female := a
					female.Gender = "female"
					female.SourceLayout = "gen8-female"
					female.SourceSlug = sp.Slug
					female.Path, _ = assetSourcePath(female)
					if _, e := verifyFemaleProvenance(lock, cache, female); e != nil {
						return m, e
					}
					m.Assets = append(m.Assets, female)
				}
			}
		}
	}
	for key := range overrides {
		if !used[key] {
			return m, fmt.Errorf("unused form exception %s", key)
		}
	}
	for _, id := range m.Selection.VisualGenderSpecies {
		if !slices.Contains(ids, id) {
			return m, fmt.Errorf("gender policy species outside selection %d", id)
		}
	}
	return m, validateMappings(&m)
}
