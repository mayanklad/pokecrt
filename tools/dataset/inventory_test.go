package main

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestAutomaticFamilyClosure(t *testing.T) {
	rows := []map[string]string{
		{"id": "1", "generation_id": "1", "evolves_from_species_id": ""},
		{"id": "2", "generation_id": "2", "evolves_from_species_id": "1"},
		{"id": "3", "generation_id": "3", "evolves_from_species_id": "1"},
		{"id": "4", "generation_id": "2", "evolves_from_species_id": ""},
		{"id": "5", "generation_id": "3", "evolves_from_species_id": "4"},
	}
	for _, tc := range []struct {
		policy inventorySelection
		want   []int
	}{
		{inventorySelection{Generations: []int{1}}, []int{1, 2, 3}},
		{inventorySelection{FamilySeeds: []int{2}}, []int{1, 2, 3}},
		{inventorySelection{Generations: []int{1}, FamilySeeds: []int{5}}, []int{1, 2, 3, 4, 5}},
	} {
		ids, e := selectedSpecies(rows, tc.policy)
		if e != nil || !reflect.DeepEqual(ids, tc.want) {
			t.Fatalf("closure=%v, %v", ids, e)
		}
	}
	if _, e := selectedSpecies(rows, inventorySelection{FamilySeeds: []int{99}}); e == nil {
		t.Fatal("accepted unknown seed")
	}
	rows[0]["evolves_from_species_id"] = "2"
	if _, e := selectedSpecies(rows, inventorySelection{Generations: []int{1}}); e == nil {
		t.Fatal("accepted cycle")
	}
}

func automaticFixture(t *testing.T) (sourceLock, string, mappingConfig) {
	lock, cache, _ := fixtureDataset(t)
	fixtureInput(t, &lock, cache, "pokeapi", "data/v2/csv/pokemon.csv", []byte("id,identifier,species_id,is_default\n1,bulbasaur,1,1\n2,ivysaur,2,1\n3,venusaur,3,1\n100,bulbasaur-mega,1,0\n"))
	fixtureInput(t, &lock, cache, "pokesprite-v2", "sources/upstreams/msikma-pokemon.json", []byte(`{"001":{"idx":"001","slug":{"eng":"bulbasaur"},"gen-8":{"forms":{"$":{},"mega":{}}}},"002":{"idx":"002","slug":{"eng":"ivysaur"},"gen-8":{"forms":{"$":{"is_unofficial_icon":true}}}}}`))
	editManifest(t, &lock, cache, func(m *sourceManifest) {
		yes, no := true, false
		alias := "base"
		m.Pokemon[0].Forms = append(m.Pokemon[0].Forms, sourceForm{ID: "mega", Label: "Mega", Slug: "bulbasaur-mega", HasRegular: &yes, HasShiny: &yes, IsGenerated: &no, Source: "msikma/pokesprite"}, sourceForm{ID: "normal", Slug: "bulbasaur", CanonicalForm: &alias})
	})
	return lock, cache, mappingConfig{RulesVersion: "d06-auto-1", Selection: &inventorySelection{Generations: []int{1}}, SourceStandardForm: "base", StandardFormReason: "Source base is standard"}
}

func TestAutomaticFormsAliasesAndQuality(t *testing.T) {
	lock, cache, m := automaticFixture(t)
	a, e := deriveMappings(lock, cache, m)
	if e != nil {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(a.CatalogSpecies, []int{1, 2, 3}) || len(a.Forms) != 4 || len(a.Assets) != 3 {
		t.Fatalf("unexpected derived inventory: %+v", a)
	}
	if a.Forms[0].ID != "standard" || !reflect.DeepEqual(a.Forms[0].SourceAliases, []string{"normal"}) || a.Forms[1].PokemonID != 100 {
		t.Fatal("lost alias folding or exact variety typing")
	}
	for _, asset := range a.Assets {
		if asset.SpeciesID != 1 {
			t.Fatal("accepted missing or provisional quality flags")
		}
	}
	b, e := deriveMappings(lock, cache, m)
	if e != nil {
		t.Fatal(e)
	}
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	if !bytes.Equal(x, y) {
		t.Fatal("nondeterministic derivation")
	}
}

func TestUnresolvedVarietyRequiresReasonedException(t *testing.T) {
	lock, cache, m := automaticFixture(t)
	editManifest(t, &lock, cache, func(s *sourceManifest) { s.Pokemon[0].Forms[1].ID = "cosmetic" })
	if _, e := deriveMappings(lock, cache, m); e == nil || !strings.Contains(e.Error(), "unresolved") {
		t.Fatalf("silently guessed variety: %v", e)
	}
	m.FormOverrides = []formMapping{{SpeciesID: 1, ID: "cosmetic", Name: "Cosmetic", SourceFormID: "cosmetic", PokemonID: 1, DefaultGender: "default", Genders: []string{"default"}, Reason: "Fixture cosmetic shares standard typing"}}
	if _, e := deriveMappings(lock, cache, m); e != nil {
		t.Fatal(e)
	}
	m.FormOverrides[0].PokemonID = 2
	if _, e := deriveMappings(lock, cache, m); e == nil {
		t.Fatal("accepted wrong-owner exception")
	}
}

func TestAutomaticPolicyRejectsMaintainedInventoryAndStaleExceptions(t *testing.T) {
	lock, cache, m := automaticFixture(t)
	m.CatalogSpecies = []int{1}
	if _, e := deriveMappings(lock, cache, m); e == nil {
		t.Fatal("accepted maintained species alongside selection")
	}
	m.CatalogSpecies = nil
	m.FormOverrides = []formMapping{{SpeciesID: 1, SourceFormID: "base", Reason: "unnecessary"}}
	if _, e := deriveMappings(lock, cache, m); e == nil {
		t.Fatal("accepted redundant exception")
	}
	m.FormOverrides = []formMapping{{SpeciesID: 999, SourceFormID: "absent", Reason: "stale"}}
	if _, e := deriveMappings(lock, cache, m); e == nil {
		t.Fatal("accepted unused exception")
	}
}
