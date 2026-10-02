package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

type inheritedInventory map[string]struct {
	ID   string `json:"idx"`
	Slug struct {
		English string `json:"eng"`
	} `json:"slug"`
	Gen8 struct {
		Forms map[string]struct {
			HasFemale          *bool `json:"has_female"`
			UnofficialFemale   bool  `json:"has_unofficial_female_icon"`
			Unofficial         bool  `json:"is_unofficial_icon"`
			PreviousGeneration bool  `json:"is_prev_gen_icon"`
		} `json:"forms"`
	} `json:"gen-8"`
}

func loadInheritedInventory(lock sourceLock, cache string) (inheritedInventory, error) {
	data, _, _, err := readInput(lock, cache, "pokesprite-v2", "sources/upstreams/msikma-pokemon.json")
	if err != nil {
		return nil, err
	}
	var result inheritedInventory
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("decode inherited inventory: %w", err)
	}
	return result, nil
}

func (inventory inheritedInventory) verify(a assetMapping, speciesSlug string) (bool, error) {
	id := fmt.Sprintf("%03d", a.SpeciesID)
	species, ok := inventory[id]
	formID := a.SourceFormID
	if formID == "base" {
		formID = "$"
	}
	form, exists := species.Gen8.Forms[formID]
	if !ok || species.ID != id || species.Slug.English != speciesSlug || !exists || form.Unofficial {
		return false, fmt.Errorf("unverified or provisional inherited appearance %s", assetIdentity(a))
	}
	return form.PreviousGeneration, nil
}

func assetSourcePath(a assetMapping) (string, error) {
	switch a.SourceLayout {
	case "":
		return "pokemon/" + a.Palette + "/" + a.SourceSlug + ".png", nil
	case "gen8-female":
		if a.Gender != "female" || a.FormID != "standard" || a.SourceFormID != "base" {
			return "", fmt.Errorf("gen8-female requires the standard female slot")
		}
		return "pokemon-gen8/" + a.Palette + "/female/" + a.SourceSlug + ".png", nil
	default:
		return "", fmt.Errorf("unsupported source layout %q", a.SourceLayout)
	}
}

// The consolidated manifest omits these female paths. Require the separately
// pinned inherited inventory, rather than inferring gender from biological data.
func verifyFemaleProvenance(lock sourceLock, cache string, a assetMapping) (bool, error) {
	data, _, _, err := readInput(lock, cache, a.SourceID, "sources/upstreams/msikma-pokemon.json")
	if err != nil {
		return false, err
	}
	var inventory map[string]struct {
		ID   string `json:"idx"`
		Slug struct {
			English string `json:"eng"`
		} `json:"slug"`
		Gen8 struct {
			Forms map[string]struct {
				HasFemale          *bool `json:"has_female"`
				UnofficialFemale   bool  `json:"has_unofficial_female_icon"`
				Unofficial         bool  `json:"is_unofficial_icon"`
				PreviousGeneration bool  `json:"is_prev_gen_icon"`
			} `json:"forms"`
		} `json:"gen-8"`
	}
	if err := json.Unmarshal(data, &inventory); err != nil {
		return false, fmt.Errorf("decode inherited gender inventory: %w", err)
	}
	id := fmt.Sprintf("%03d", a.SpeciesID)
	species, ok := inventory[id]
	form, exists := species.Gen8.Forms["$"]
	if !ok || species.ID != id || species.Slug.English != a.SourceSlug || !exists || form.HasFemale == nil || !*form.HasFemale || form.UnofficialFemale || form.Unofficial {
		return false, fmt.Errorf("unverified or unofficial female source appearance %s", assetIdentity(a))
	}
	return form.PreviousGeneration, nil
}

func validateSourceNames(m mappingConfig) error {
	seen := map[int]bool{}
	for _, name := range m.SourceNameOverrides {
		if m.RulesVersion != "d06b-gen1-1" && !automaticRules(m.RulesVersion) || !slices.Contains(m.CatalogSpecies, name.SpeciesID) || seen[name.SpeciesID] || strings.TrimSpace(name.SourceName) == "" || strings.TrimSpace(name.CanonicalName) == "" || name.SourceName == name.CanonicalName || strings.TrimSpace(name.Reason) == "" {
			return fmt.Errorf("invalid or duplicate source-name correction for #%03d", name.SpeciesID)
		}
		seen[name.SpeciesID] = true
	}
	return nil
}

func sourceNameMatches(m mappingConfig, id int, source, canonical string) bool {
	for _, name := range m.SourceNameOverrides {
		if name.SpeciesID == id {
			return name.SourceName == source && name.CanonicalName == canonical
		}
	}
	return source == canonical
}

func providerReviewed(m mappingConfig, provider string) bool {
	return provider == "msikma/pokesprite" || (provider == "bamq/pokemon-sprites" && slices.Contains(m.ReviewedProviders, provider))
}
func validateReviewedProviders(m mappingConfig) error {
	if len(m.ReviewedProviders) == 0 {
		return nil
	}
	if (m.RulesVersion != "d06-auto-7" && m.RulesVersion != "d06-auto-8") || len(m.ReviewedProviders) != 1 || m.ReviewedProviders[0] != "bamq/pokemon-sprites" {
		return fmt.Errorf("unsupported or duplicate reviewed provider policy")
	}
	return nil
}
func verifyAssetProvider(lock sourceLock, cache string, m mappingConfig, a assetMapping, slug string, verifyPixels bool) (bool, error) {
	provider := a.Provider
	if provider == "" {
		provider = "msikma/pokesprite"
	}
	if !providerReviewed(m, provider) {
		return false, fmt.Errorf("unreviewed provider %s", provider)
	}
	if provider == "msikma/pokesprite" {
		inventory, err := loadInheritedInventory(lock, cache)
		if err != nil {
			return false, err
		}
		return inventory.verify(a, slug)
	}
	if a.SourceLayout != "" || a.Gender != "default" {
		return false, fmt.Errorf("unsupported provider gender/layout %s", assetIdentity(a))
	}
	data, _, _, err := readInput(lock, cache, "pokesprite-v2", "sources/upstreams/upstream-lock.json")
	if err != nil {
		return false, err
	}
	var evidence struct {
		Sources struct {
			Bamq struct {
				URL  string `json:"url"`
				Head string `json:"head"`
			} `json:"bamq_repo"`
		} `json:"sources"`
	}
	if err = json.Unmarshal(data, &evidence); err != nil {
		return false, err
	}
	found := false
	for _, src := range lock.Sources {
		if src.ID == "bamq" {
			found = src.Repository == "bamq/pokemon-sprites" && evidence.Sources.Bamq.URL == "https://github.com/"+src.Repository && evidence.Sources.Bamq.Head == src.Revision
		}
	}
	if !found {
		return false, fmt.Errorf("provider revision does not match pinned import evidence")
	}
	for _, name := range []string{"README.md", "contributors.md"} {
		if _, _, _, err = readInput(lock, cache, "bamq", name); err != nil {
			return false, err
		}
	}
	if verifyPixels {
		canonical, _, _, err := readInput(lock, cache, "pokesprite-v2", a.Path)
		if err != nil {
			return false, err
		}
		original, _, _, err := readInput(lock, cache, "bamq", a.Path)
		if err != nil {
			return false, err
		}
		if !bytes.Equal(canonical, original) {
			return false, fmt.Errorf("provider import bytes differ %s", assetIdentity(a))
		}
	}
	return false, nil
}
