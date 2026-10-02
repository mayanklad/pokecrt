package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"
)

// Selection describes development scope, not a manually enumerated catalog.
type inventorySelection struct {
	Generations             []int `json:"generations"`
	FamilySeeds             []int `json:"family_seeds"`
	VisualGenderSpecies     []int `json:"visual_gender_species,omitempty"`
	AuditedInheritedGenders bool  `json:"audited_inherited_genders,omitempty"`
}

func validateSelection(m mappingConfig) error {
	if m.Selection != nil && m.Selection.AuditedInheritedGenders && (automaticRuleLevel(m.RulesVersion) < 8 || len(m.Selection.VisualGenderSpecies) != 0) {
		return fmt.Errorf("automatic inherited genders require rules v8 and no maintained species list")
	}

	if !automaticRules(m.RulesVersion) || m.Selection == nil || m.SourceStandardForm != "base" || m.StandardFormReason == "" || len(m.Selection.Generations)+len(m.Selection.FamilySeeds) == 0 {
		return fmt.Errorf("automatic inventory requires a supported policy and selection")
	}
	if len(m.SourceFormExclusions) > 0 && automaticRuleLevel(m.RulesVersion) < 3 {
		return fmt.Errorf("source-only exclusions require automatic rules v3")
	}
	for _, e := range m.SourceFormExclusions {
		if e.UnsupportedType != "" {
			return fmt.Errorf("unsupported-type exclusions must be derived from pinned metadata")
		}
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
	if err := validateReviewedProviders(m); err != nil {
		return m, err
	}
	if len(m.FormSuffixRules)+len(m.ReviewedDefaultAliases) > 0 && automaticRuleLevel(m.RulesVersion) < 6 {
		return m, fmt.Errorf("form suffix and default alias corrections require automatic rules v6")
	}
	if len(m.DuplicatePaletteExclusions) > 0 && automaticRuleLevel(m.RulesVersion) < 5 {
		return m, fmt.Errorf("duplicate palette exclusions require automatic rules v5")
	}
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
	if err := applyReviewedDefaultAliases(&manifest, m); err != nil {
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
	metadataIDs := map[string]int{}
	metadataMega := map[string]bool{}
	defaultMetadataIDs := map[int]int{}
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
			required := []string{"identifier", "pokemon_id"}
			if automaticRuleLevel(m.RulesVersion) >= 9 {
				required = append(required, "id", "is_mega")
			}
			forms, e := readTable(lock, cache, "pokemon_forms.csv", required...)
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
				if automaticRuleLevel(m.RulesVersion) >= 9 {
					mega, e := sourceBool(row, "is_mega")
					if e != nil {
						return m, e
					}
					metadataMega[key] = mega
				}
				if row["id"] != "" {
					formID, e := integer(row, "id", false)
					if e != nil {
						return m, e
					}
					metadataIDs[key] = formID
					if row["is_default"] == "1" {
						if defaultMetadataIDs[id] != 0 {
							return m, fmt.Errorf("duplicate metadata default form for variety %d", id)
						}
						defaultMetadataIDs[id] = formID
					}
				}
				if existing := names[key]; existing != 0 && existing != id {
					return m, fmt.Errorf("conflicting variety/form identity %s", key)
				}
				names[key] = id
			}
		}
	}
	if automaticRuleLevel(m.RulesVersion) >= 2 && !foundFormTable {
		return m, fmt.Errorf("automatic rules v2 require pinned pokemon_forms.csv")
	}
	formOwners, formTypes, err := metadataFormTyping(lock, cache, m.RulesVersion)
	if err != nil {
		return m, err
	}

	if err := applyFormSuffixRules(m, ids, byID, names, owners, metadataIDs); err != nil {
		return m, err
	}
	for _, a := range m.ReviewedDefaultAliases {
		sp := byID[a.SpeciesID]
		key := sp.Slug + "-" + a.SourceFormID
		if names[key] != defaults[sp.ID] || metadataIDs[key] == 0 || metadataIDs[key] != defaultMetadataIDs[defaults[sp.ID]] {
			return m, fmt.Errorf("reviewed alias does not match exact default metadata %d/%s", a.SpeciesID, a.SourceFormID)
		}
	}
	inherited, err := loadInheritedInventory(lock, cache)
	if err != nil {
		return m, err
	}
	excluded := map[string]sourceFormExclusion{}
	usedExclusions := map[string]bool{}
	for _, e := range m.SourceFormExclusions {
		key := fmt.Sprintf("%d/%s", e.SpeciesID, e.SourceFormID)
		if _, ok := excluded[key]; ok || strings.TrimSpace(e.Reason) == "" || !slugValid(e.SourceFormID) || !slices.Contains(ids, e.SpeciesID) {
			return m, fmt.Errorf("invalid or duplicate source exclusion %s", key)
		}
		excluded[key] = e
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
	m.Exclusions = []string{"Scope is selected generations and complete connected evolution families; wider coverage remains pending.", "Visual genders require reviewed source declarations, exact identity, nonprovisional provenance and distinct pixels; additional providers/layouts remain pending audit."}
	for _, a := range m.ReviewedDefaultAliases {
		m.Exclusions = append(m.Exclusions, fmt.Sprintf("#%03d/%s: %s Regular source/default SHA-256: %s/%s; shiny: %s/%s.", a.SpeciesID, a.SourceFormID, a.Reason, a.SourceHashes["regular"], a.DefaultHashes["regular"], a.SourceHashes["shiny"], a.DefaultHashes["shiny"]))
	}
	for _, id := range ids {
		sp, ok := byID[id]
		if !ok {
			return m, fmt.Errorf("missing source species %d", id)
		}
		if sp.DefaultForm == "" || defaults[id] == 0 {
			return m, fmt.Errorf("unresolved standard identity %d", id)
		}
		gendered := slices.Contains(m.Selection.VisualGenderSpecies, id)
		if m.Selection.AuditedInheritedGenders {
			entry := inherited[fmt.Sprintf("%03d", id)]
			flags := entry.Gen8.Forms["$"]
			if flags.HasFemale != nil && *flags.HasFemale {
				if entry.ID != fmt.Sprintf("%03d", id) || entry.Slug.English != sp.Slug {
					return m, fmt.Errorf("inherited gender owner mismatch #%03d", id)
				}
				base, err := sourceFormByID(sp, "base")
				if err != nil {
					return m, err
				}
				if sp.DefaultForm != "base" || base.Source != "msikma/pokesprite" || base.IsGenerated == nil || *base.IsGenerated || flags.Unofficial || flags.UnofficialFemale {
					m.Exclusions = append(m.Exclusions, fmt.Sprintf("#%03d/standard/female: declared inherited female candidate excluded by provider/layout/provisional policy.", id))
				} else {
					gendered = true
				}
			}
		}

		metadataOnlyAliases := map[string]bool{}
		if automaticRuleLevel(m.RulesVersion) >= 9 {
			for _, alias := range sp.Forms {
				key := sp.Slug + "-" + alias.ID
				if alias.CanonicalForm == nil || !metadataMega[key] {
					continue
				}
				target, err := sourceFormByID(sp, *alias.CanonicalForm)
				if err != nil {
					return m, err
				}
				targetKey := sp.Slug + "-" + target.ID
				targetID := names[targetKey]
				if target.ID == sp.DefaultForm {
					targetID = defaults[id]
				}
				if metadataIDs[key] == 0 || owners[names[key]] != id || owners[targetID] != id || alias.Slug != target.Slug || target.CanonicalForm != nil {
					return m, fmt.Errorf("invalid transformation alias evidence %d/%s", id, alias.ID)
				}
				if names[key] != targetID && !metadataMega[targetKey] {
					metadataOnlyAliases[alias.ID] = true
				}
			}
		}
		for _, sf := range sp.Forms {
			metadataOnly := metadataOnlyAliases[sf.ID]
			if sf.CanonicalForm != nil && !metadataOnly {
				continue
			}
			exclusionKey := fmt.Sprintf("%d/%s", id, sf.ID)
			if exclusion, ok := excluded[exclusionKey]; ok {
				if sf.ID == sp.DefaultForm {
					return m, fmt.Errorf("cannot exclude a standard metadata identity %s", exclusionKey)
				}
				sourceID := fmt.Sprintf("%03d", id)
				entry, hasSpecies := inherited[sourceID]
				flags, hasForm := entry.Gen8.Forms[sf.ID]
				if names[sp.Slug+"-"+sf.ID] != 0 || !hasSpecies || !hasForm || entry.ID != sourceID || entry.Slug.English != sp.Slug || !flags.Unofficial {
					return m, fmt.Errorf("source exclusion lacks verified unofficial template evidence %s", exclusionKey)
				}
				usedExclusions[exclusionKey] = true
				m.Exclusions = append(m.Exclusions, fmt.Sprintf("#%03d/%s: %s", id, sf.ID, exclusion.Reason))
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
			if sf.ID == sp.DefaultForm {
				f.MetadataFormID = defaultMetadataIDs[f.PokemonID]
			} else {
				f.MetadataFormID = metadataIDs[sp.Slug+"-"+sf.ID]
			}
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
			unsupported := ""
			if f.MetadataFormID != 0 && len(formOwners) > 0 {
				if formOwners[f.MetadataFormID] != f.PokemonID {
					return m, fmt.Errorf("metadata form owner mismatch %s", key)
				}
				for _, slot := range []int{1, 2} {
					if typ := formTypes[f.MetadataFormID][slot]; typ != "" && !supportedPokemonType(typ) {
						unsupported = typ
						break
					}
				}
			}
			if unsupported != "" {
				if sf.ID == sp.DefaultForm {
					return m, fmt.Errorf("unsupported standard typing %s", key)
				}
				reason := fmt.Sprintf("Pinned metadata form #%d has unsupported type %s; excluded without substitution.", f.MetadataFormID, unsupported)
				m.SourceFormExclusions = append(m.SourceFormExclusions, sourceFormExclusion{SpeciesID: id, SourceFormID: sf.ID, Reason: reason, UnsupportedType: unsupported})
				m.Exclusions = append(m.Exclusions, fmt.Sprintf("#%03d/%s: %s", id, sf.ID, reason))
				continue
			}
			for _, alias := range sp.Forms {
				if alias.CanonicalForm != nil && *alias.CanonicalForm == sf.ID && !metadataOnlyAliases[alias.ID] {
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
			f.MetadataOnlyAlias = metadataOnly
			m.Forms = append(m.Forms, f)
			if metadataOnly {
				m.Exclusions = append(m.Exclusions, fmt.Sprintf("#%03d/%s: exact metadata transformation has a different owning variety; source aliases ordinary artwork. Retained as metadata-only with no substituted asset.", id, f.ID))
				continue
			}
			a := assetMapping{SpeciesID: id, SourceID: "pokesprite-v2", Provider: sf.Source, SourceSlug: sf.Slug, FormID: f.ID, SourceFormID: sf.ID, Gender: f.DefaultGender, Palette: "regular", Reason: "Derived from pinned manifest; reviewed provider and exact provenance required."}
			if sf.IsGenerated == nil || *sf.IsGenerated || !providerReviewed(m, sf.Source) {
				m.Exclusions = append(m.Exclusions, fmt.Sprintf("#%03d/%s: provider or generated status is outside the reviewed artwork policy.", id, f.ID))
				continue
			}
			var providerErr error
			if a.Provider == "msikma/pokesprite" {
				_, providerErr = inherited.verify(a, sp.Slug)
			} else {
				_, providerErr = verifyAssetProvider(lock, cache, m, a, sp.Slug, false)
			}
			if providerErr != nil {
				if a.Provider == "bamq/pokemon-sprites" {
					return m, fmt.Errorf("provider evidence for #%03d/%s: %w", id, f.ID, providerErr)
				}
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
	for key := range excluded {
		if !usedExclusions[key] {
			return m, fmt.Errorf("unused source exclusion %s", key)
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

// Default aliases are explicit corrections, not filename similarity guesses.
func applyReviewedDefaultAliases(manifest *sourceManifest, m mappingConfig) error {
	seen := map[string]bool{}
	for _, a := range m.ReviewedDefaultAliases {
		key := fmt.Sprintf("%d/%s", a.SpeciesID, a.SourceFormID)
		if seen[key] || strings.TrimSpace(a.Reason) == "" || !slugValid(a.SourceFormID) {
			return fmt.Errorf("invalid default alias %s", key)
		}

		for _, hashes := range []map[string]string{a.SourceHashes, a.DefaultHashes} {
			if len(hashes) != 2 {
				return fmt.Errorf("default alias requires both palette hashes %s", key)
			}
			for _, palette := range []string{"regular", "shiny"} {
				decoded, err := hex.DecodeString(hashes[palette])
				if err != nil || len(decoded) != 32 {
					return fmt.Errorf("invalid default alias hash %s/%s", key, palette)
				}
			}
		}
		seen[key] = true
		found := false
		for i := range manifest.Pokemon {
			sp := &manifest.Pokemon[i]
			if sp.ID != a.SpeciesID {
				continue
			}
			target, err := sourceFormByID(*sp, sp.DefaultForm)
			if err != nil {
				return err
			}
			for j := range sp.Forms {
				sf := &sp.Forms[j]
				if sf.ID != a.SourceFormID {
					continue
				}
				if sf.ID == sp.DefaultForm || sf.CanonicalForm != nil {
					return fmt.Errorf("redundant default alias %s", key)
				}
				canonical := sp.DefaultForm
				sf.CanonicalForm = &canonical
				sf.Slug = target.Slug
				found = true
			}
		}
		if !found {
			return fmt.Errorf("unused default alias %s", key)
		}
	}
	return nil
}

func reviewedAliasAssets(lock sourceLock, cache string, m mappingConfig) ([]assetMapping, error) {
	data, _, _, err := readInput(lock, cache, "pokesprite-v2", "data/pokemon.json")
	if err != nil {
		return nil, err
	}
	var manifest sourceManifest
	if err = json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	result := []assetMapping{}
	for _, a := range m.ReviewedDefaultAliases {
		found := false
		for _, sp := range manifest.Pokemon {
			if sp.ID != a.SpeciesID {
				continue
			}
			sf, err := sourceFormByID(sp, a.SourceFormID)
			if err != nil {
				return nil, err
			}
			if sf.IsGenerated == nil || *sf.IsGenerated || sf.Source != "msikma/pokesprite" || sf.HasRegular == nil || !*sf.HasRegular || sf.HasShiny == nil || !*sf.HasShiny {
				return nil, fmt.Errorf("alias lacks audited palettes %d/%s", a.SpeciesID, a.SourceFormID)
			}
			for _, palette := range []string{"regular", "shiny"} {
				asset := assetMapping{SpeciesID: sp.ID, FormID: "standard", SourceFormID: sf.ID, SourceID: "pokesprite-v2", Provider: sf.Source, SourceSlug: sf.Slug, Gender: "default", Palette: palette}
				asset.Path, _ = assetSourcePath(asset)
				result = append(result, asset)
			}
			found = true
		}
		if !found {
			return nil, fmt.Errorf("missing alias species %d", a.SpeciesID)
		}
	}
	return result, nil
}

func applyFormSuffixRules(m mappingConfig, ids []int, byID map[int]sourceSpecies, names map[string]int, owners map[int]int, metadataIDs map[string]int) error {
	usedSuffix := map[int]bool{}
	seenSuffix := map[int]bool{}
	for _, rule := range m.FormSuffixRules {
		if rule.SpeciesID <= 0 || seenSuffix[rule.SpeciesID] || rule.Suffix == "" || !slugValid("form"+rule.Suffix) || strings.TrimSpace(rule.Reason) == "" || !slices.Contains(ids, rule.SpeciesID) {
			return fmt.Errorf("invalid form suffix rule %d", rule.SpeciesID)
		}
		seenSuffix[rule.SpeciesID] = true
		sp := byID[rule.SpeciesID]
		for _, sf := range sp.Forms {
			key := sp.Slug + "-" + sf.ID
			corrected := key + rule.Suffix
			if sf.ID == sp.DefaultForm || (sf.CanonicalForm != nil && !slices.ContainsFunc(m.ReviewedDefaultAliases, func(a reviewedDefaultAlias) bool { return a.SpeciesID == sp.ID && a.SourceFormID == sf.ID })) || names[key] != 0 {
				continue
			}
			if names[corrected] == 0 {
				continue
			}
			if owners[names[corrected]] != sp.ID || metadataIDs[corrected] == 0 {
				return fmt.Errorf("suffix rule lacks same-species metadata form %s", corrected)
			}
			names[key] = names[corrected]
			metadataIDs[key] = metadataIDs[corrected]
			usedSuffix[rule.SpeciesID] = true
		}
		if !usedSuffix[rule.SpeciesID] {
			return fmt.Errorf("unused form suffix rule %d", rule.SpeciesID)
		}
	}
	return nil
}

func automaticRules(version string) bool { return automaticRuleLevel(version) > 0 }
func automaticRuleLevel(version string) int {
	for level := 1; level <= 9; level++ {
		if version == fmt.Sprintf("d06-auto-%d", level) {
			return level
		}
	}
	return 0
}
