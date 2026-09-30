package main

import (
	"encoding/json"
	"fmt"
)

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
