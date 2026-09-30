package main

import (
	"encoding/json"
	"fmt"
)

type inheritedInventory map[string]struct {
	ID   string `json:"idx"`
	Slug struct {
		English string `json:"eng"`
	} `json:"slug"`
	Gen8 struct {
		Forms map[string]struct {
			Unofficial         bool `json:"is_unofficial_icon"`
			PreviousGeneration bool `json:"is_prev_gen_icon"`
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
