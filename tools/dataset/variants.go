package main

import (
	"fmt"
	"slices"
	"sort"
	"strings"
)

func slugValid(value string) bool {
	if value == "" || value[0] == '-' || value[len(value)-1] == '-' {
		return false
	}
	for _, r := range value {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-') {
			return false
		}
	}
	return true
}

func assetIdentity(a assetMapping) string {
	return fmt.Sprintf("%d/%s/%s/%s", a.SpeciesID, a.FormID, a.Gender, a.Palette)
}

func validateMappings(m *mappingConfig) error {
	if m.RulesVersion != "d02b-1" && m.RulesVersion != "d06a-1" && m.RulesVersion != "d06b-gender-1" && m.RulesVersion != "d06b-gen1-1" && (m.RulesVersion != "d06-auto-1" && m.RulesVersion != "d06-auto-2" && m.RulesVersion != "d06-auto-3" && (m.RulesVersion != "d06-auto-4" && (m.RulesVersion != "d06-auto-5" && m.RulesVersion != "d06-auto-6"))) {
		return fmt.Errorf("unsupported mapping rules %q", m.RulesVersion)
	}
	if m.SourceStandardForm != "base" || strings.TrimSpace(m.StandardFormReason) == "" || len(m.CatalogSpecies) == 0 {
		return fmt.Errorf("missing catalog or standard-form mapping")
	}
	sort.Ints(m.CatalogSpecies)
	for i, id := range m.CatalogSpecies {
		if id <= 0 || (i > 0 && id == m.CatalogSpecies[i-1]) {
			return fmt.Errorf("invalid or duplicate species %d", id)
		}
	}

	if err := validateSourceNames(*m); err != nil {
		return err
	}
	seen := map[string]bool{}
	for i := range m.Assets {
		a := &m.Assets[i]
		if m.RulesVersion == "d02b-1" {
			if a.FormID == "" {
				a.FormID = "standard"
			}
			if a.SourceFormID == "" {
				a.SourceFormID = "base"
			}
			if a.Gender == "" {
				a.Gender = "default"
			}
			if a.Palette == "" {
				a.Palette = "regular"
			}
		}
		identity := assetIdentity(*a)
		expectedPath, layoutErr := assetSourcePath(*a)
		if !slices.Contains(m.CatalogSpecies, a.SpeciesID) || a.SourceID != "pokesprite-v2" || !slugValid(a.FormID) || !slugValid(a.SourceFormID) || !slugValid(a.SourceSlug) || strings.TrimSpace(a.Reason) == "" || (a.Gender != "default" && a.Gender != "male" && a.Gender != "female") || (a.Palette != "regular" && a.Palette != "shiny") || layoutErr != nil || (a.SourceLayout != "" && m.RulesVersion != "d06b-gender-1" && m.RulesVersion != "d06b-gen1-1" && (m.RulesVersion != "d06-auto-1" && m.RulesVersion != "d06-auto-2" && m.RulesVersion != "d06-auto-3" && (m.RulesVersion != "d06-auto-4" && (m.RulesVersion != "d06-auto-5" && m.RulesVersion != "d06-auto-6")))) || a.Path != expectedPath || seen[identity] {
			return fmt.Errorf("invalid or duplicate asset mapping %s", identity)
		}
		seen[identity] = true
	}
	sort.Slice(m.Assets, func(i, j int) bool {
		a, b := m.Assets[i], m.Assets[j]
		if a.SpeciesID != b.SpeciesID {
			return a.SpeciesID < b.SpeciesID
		}
		return assetIdentity(a) < assetIdentity(b)
	})
	if m.RulesVersion != "d02b-1" && len(m.Forms) == 0 {
		return fmt.Errorf("explicit rules require form mappings")
	}
	seen = map[string]bool{}
	for i := range m.Forms {
		f := &m.Forms[i]
		identity := fmt.Sprintf("%d/%s", f.SpeciesID, f.ID)
		if !slices.Contains(m.CatalogSpecies, f.SpeciesID) || !slugValid(f.ID) || !slugValid(f.SourceFormID) || strings.TrimSpace(f.Name) == "" || strings.TrimSpace(f.Reason) == "" || f.PokemonID <= 0 || seen[identity] || len(f.Genders) == 0 {
			return fmt.Errorf("invalid or duplicate form mapping %s", identity)
		}
		seen[identity] = true
		genders := append([]string{}, f.Genders...)
		sort.Strings(genders)
		if len(slices.Compact(append([]string{}, genders...))) != len(genders) {
			return fmt.Errorf("duplicate visual gender in %s", identity)
		}
		if !slices.Contains(genders, f.DefaultGender) {
			return fmt.Errorf("default gender is not declared in %s", identity)
		}
		if len(genders) == 1 {
			if genders[0] != "default" {
				return fmt.Errorf("single visual slot must use default in %s", identity)
			}
		} else if !slices.Equal(genders, []string{"female", "male"}) {
			return fmt.Errorf("distinct visual slots must be male and female in %s", identity)
		}
		f.Genders = genders
		sort.Strings(f.SourceAliases)
		if len(slices.Compact(append([]string{}, f.SourceAliases...))) != len(f.SourceAliases) {
			return fmt.Errorf("duplicate source-only alias in %s", identity)
		}
	}
	sort.Slice(m.Forms, func(i, j int) bool {
		a, b := m.Forms[i], m.Forms[j]
		if a.SpeciesID != b.SpeciesID {
			return a.SpeciesID < b.SpeciesID
		}
		if a.ID == "standard" {
			return b.ID != "standard"
		}
		if b.ID == "standard" {
			return false
		}
		return a.ID < b.ID
	})
	return nil
}

func sourceFormByID(species sourceSpecies, id string) (sourceForm, error) {
	var found *sourceForm
	for _, form := range species.Forms {
		if form.ID != id {
			continue
		}
		if found != nil {
			return sourceForm{}, fmt.Errorf("duplicate source form #%03d/%s", species.ID, id)
		}
		copy := form
		found = &copy
	}
	if found == nil {
		return sourceForm{}, fmt.Errorf("missing source form #%03d/%s", species.ID, id)
	}
	return *found, nil
}

func normalizeForms(id int, source sourceSpecies, m mappingConfig, defaultID int, owners map[int]int, slots map[int]map[int]string, metadata ...metadataTyping) ([]normalizedForm, error) {
	var mappings []formMapping
	for _, f := range m.Forms {
		if f.SpeciesID == id {
			mappings = append(mappings, f)
		}
	}
	if len(mappings) == 0 && m.RulesVersion == "d02b-1" {
		mappings = []formMapping{{SpeciesID: id, ID: "standard", Name: "Standard", SourceFormID: m.SourceStandardForm, PokemonID: defaultID, DefaultGender: "default", Genders: []string{"default"}}}
	}
	var result []normalizedForm
	claimed := map[string]bool{}
	hasStandard := false
	for _, mapping := range mappings {
		form, err := sourceFormByID(source, mapping.SourceFormID)
		if err != nil {
			return nil, err
		}
		if form.CanonicalForm != nil || claimed[form.ID] {
			return nil, fmt.Errorf("aliased or multiply mapped source form #%03d/%s", id, form.ID)
		}
		claimed[form.ID] = true
		if owners[mapping.PokemonID] != id || slots[mapping.PokemonID][1] == "" {
			return nil, fmt.Errorf("form typing owner mismatch #%03d/%s", id, mapping.ID)
		}
		if mapping.ID == "standard" {
			if hasStandard || mapping.SourceFormID != source.DefaultForm || mapping.PokemonID != defaultID {
				return nil, fmt.Errorf("standard form mismatch #%03d", id)
			}
			hasStandard = true
		}
		selectedSlots := slots[mapping.PokemonID]
		if mapping.MetadataFormID != 0 && len(metadata) > 0 && len(metadata[0].Owners) > 0 {
			if metadata[0].Owners[mapping.MetadataFormID] != mapping.PokemonID {
				return nil, fmt.Errorf("metadata form typing owner mismatch #%03d/%s", id, mapping.ID)
			}
			if explicit := metadata[0].Slots[mapping.MetadataFormID]; len(explicit) > 0 {
				selectedSlots = explicit
			}
		}
		types := []string{selectedSlots[1]}
		for _, typ := range selectedSlots {
			if !supportedPokemonType(typ) {
				return nil, fmt.Errorf("unsupported form type %s for #%03d/%s", typ, id, mapping.ID)
			}
		}
		if second := selectedSlots[2]; second != "" {
			if types[0] == second {
				return nil, fmt.Errorf("duplicate form typing #%03d/%s", id, mapping.ID)
			}
			types = append(types, second)
		}
		for _, aliasID := range mapping.SourceAliases {
			alias, err := sourceFormByID(source, aliasID)
			if err != nil {
				return nil, err
			}
			if alias.CanonicalForm == nil || *alias.CanonicalForm != form.ID || alias.Slug != form.Slug || claimed[aliasID] {
				return nil, fmt.Errorf("invalid source-only form alias #%03d/%s", id, aliasID)
			}
			claimed[aliasID] = true
		}
		// A source form may encode a gender slot instead of a collectible form.
		// That folding is explicit in the asset mappings and requires distinct
		// regular pixels during inventory validation.
		for _, asset := range m.Assets {
			if asset.SpeciesID != id || asset.FormID != mapping.ID || asset.Gender == "default" {
				continue
			}
			if !slices.Contains(mapping.Genders, asset.Gender) {
				return nil, fmt.Errorf("undeclared visual gender #%03d/%s", id, mapping.ID)
			}
			genderForm, err := sourceFormByID(source, asset.SourceFormID)
			if err != nil {
				return nil, err
			}
			if genderForm.CanonicalForm != nil {
				return nil, fmt.Errorf("gender asset uses a source-only alias #%03d/%s", id, genderForm.ID)
			}
			if genderForm.ID != form.ID {
				// Multiple palettes may refer to the same gender identity. A separate
				// normalized form cannot also claim it (checked below).
				claimed[genderForm.ID] = true
			}
		}
		result = append(result, normalizedForm{ID: mapping.ID, Name: mapping.Name, Types: types, DefaultGender: mapping.DefaultGender, Genders: mapping.Genders, SourceFormID: form.ID, SourceAliases: mapping.SourceAliases})
	}
	for _, exclusion := range m.SourceFormExclusions {
		if exclusion.SpeciesID != id {
			continue
		}
		form, err := sourceFormByID(source, exclusion.SourceFormID)
		if err != nil {
			return nil, err
		}
		if form.CanonicalForm != nil || form.ID == source.DefaultForm || claimed[form.ID] {
			return nil, fmt.Errorf("invalid or already claimed source exclusion #%03d/%s", id, form.ID)
		}
		claimed[form.ID] = true
	}
	if !hasStandard {
		return nil, fmt.Errorf("missing standard form mapping #%03d", id)
	}
	// Explicit rules require every source identity to be mapped or folded into a
	// declared source-only alias; missing entries cannot disappear silently.
	if m.RulesVersion != "d02b-1" {
		for _, form := range source.Forms {
			if !claimed[form.ID] {
				return nil, fmt.Errorf("unmapped source form #%03d/%s", id, form.ID)
			}
		}
	}
	return result, nil
}

func findNormalizedForm(species []normalizedSpecies, id int, formID string) (normalizedForm, bool) {
	for _, s := range species {
		if s.ID != id {
			continue
		}
		for _, f := range s.Forms {
			if f.ID == formID {
				return f, true
			}
		}
	}
	return normalizedForm{}, false
}

// Every collectible palette belongs to a real regular visual slot. Shiny-only
// entries and equal regular/shiny pixels are rejected rather than substituted.
func validateVariantInventory(species []normalizedSpecies, assets []normalizedAsset) error {
	hashes := map[string]string{}
	for _, a := range assets {
		key := fmt.Sprintf("%d/%s/%s/%s", a.SpeciesID, a.FormID, a.Gender, a.Palette)
		if hashes[key] != "" {
			return fmt.Errorf("duplicate variant identity %s", key)
		}
		hashes[key] = a.SHA256
	}
	for _, s := range species {
		for _, f := range s.Forms {
			if len(f.Genders) > 1 {
				first := ""
				for _, gender := range f.Genders {
					key := fmt.Sprintf("%d/%s/%s/regular", s.ID, f.ID, gender)
					hash := hashes[key]
					if hash == "" || hash == first {
						return fmt.Errorf("distinct visual genders require different regular assets #%03d/%s", s.ID, f.ID)
					}
					first = hash
				}
			}
		}
	}
	for _, a := range assets {
		if a.Palette != "shiny" {
			continue
		}
		regular := hashes[fmt.Sprintf("%d/%s/%s/regular", a.SpeciesID, a.FormID, a.Gender)]
		if regular == "" || regular == a.SHA256 {
			return fmt.Errorf("shiny-only or substituted shiny asset #%03d/%s/%s", a.SpeciesID, a.FormID, a.Gender)
		}
	}
	return nil
}

// Exclusions require both verified palettes to have exactly equal normalized bytes.
// Distinct shiny artwork can never be removed through this policy.
func excludeDuplicatePalettes(assets []normalizedAsset, exclusions []duplicatePaletteExclusion) ([]normalizedAsset, []string, error) {
	excluded := map[string]bool{}
	reasons := []string{}
	for _, e := range exclusions {
		key := fmt.Sprintf("%d/%s/%s", e.SpeciesID, e.FormID, e.Gender)
		if e.SpeciesID <= 0 || !slugValid(e.FormID) || (e.Gender != "default" && e.Gender != "male" && e.Gender != "female") || strings.TrimSpace(e.Reason) == "" || excluded[key] {
			return nil, nil, fmt.Errorf("invalid duplicate-palette exclusion %s", key)
		}
		var regular, shiny string
		for _, a := range assets {
			if a.SpeciesID == e.SpeciesID && a.FormID == e.FormID && a.Gender == e.Gender {
				if a.Palette == "regular" {
					regular = a.SHA256
				}
				if a.Palette == "shiny" {
					shiny = a.SHA256
				}
			}
		}
		if regular == "" || shiny == "" || regular != shiny {
			return nil, nil, fmt.Errorf("duplicate-palette exclusion lacks equal verified palettes %s", key)
		}
		excluded[key] = true
		reasons = append(reasons, fmt.Sprintf("#%03d/%s/%s/shiny: %s Normalized SHA-256 equals regular: %s.", e.SpeciesID, e.FormID, e.Gender, e.Reason, regular))
	}
	kept := []normalizedAsset{}
	for _, a := range assets {
		if a.Palette == "shiny" && excluded[fmt.Sprintf("%d/%s/%s", a.SpeciesID, a.FormID, a.Gender)] {
			continue
		}
		kept = append(kept, a)
	}
	return kept, reasons, nil
}
