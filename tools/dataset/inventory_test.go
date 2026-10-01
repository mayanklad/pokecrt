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

func TestMetadataFormResolutionAndDefaultIdentity(t *testing.T) {
	lock, cache, m := automaticFixture(t)
	m.RulesVersion = "d06-auto-2"
	if _, e := deriveMappings(lock, cache, m); e == nil {
		t.Fatal("accepted missing v2 form table")
	}
	fixtureInput(t, &lock, cache, "pokeapi", "data/v2/csv/pokemon_forms.csv", []byte("identifier,pokemon_id\nbulbasaur-cosmetic,1\n"))
	editManifest(t, &lock, cache, func(s *sourceManifest) { s.Pokemon[0].Forms[1].ID = "cosmetic" })
	resolved, e := deriveMappings(lock, cache, m)
	if e != nil {
		t.Fatal(e)
	}
	if resolved.Forms[1].PokemonID != 1 {
		t.Fatal("cosmetic form did not inherit its exact metadata owner typing")
	}
	fixtureInput(t, &lock, cache, "pokeapi", "data/v2/csv/pokemon.csv", []byte("id,identifier,species_id,is_default\n1,bulbasaur-two-segment,1,1\n2,ivysaur,2,1\n3,venusaur,3,1\n100,bulbasaur-mega,1,0\n"))
	editManifest(t, &lock, cache, func(s *sourceManifest) {
		s.Pokemon[0].DefaultForm = "two-segment"
		s.Pokemon[0].Forms[0].ID = "two-segment"
		s.Pokemon[0].Forms = s.Pokemon[0].Forms[:2]
	})
	resolved, e = deriveMappings(lock, cache, m)
	if e != nil {
		t.Fatal(e)
	}
	if resolved.Forms[0].ID != "standard" || resolved.Forms[0].SourceFormID != "two-segment" || resolved.Forms[0].PokemonID != 1 {
		t.Fatal("non-base default not mapped to standard")
	}
	fixtureInput(t, &lock, cache, "pokeapi", "data/v2/csv/pokemon.csv", []byte("id,identifier,species_id,is_default\n1,bulbasaur,1,1\n2,ivysaur,2,1\n3,venusaur,3,1\n100,bulbasaur-two-segment,1,0\n"))
	if _, e := deriveMappings(lock, cache, m); e == nil {
		t.Fatal("accepted source/metadata default mismatch")
	}
}

func TestSourceTemplatesRequireEvidenceAndDoNotBecomeForms(t *testing.T) {
	for _, tc := range []struct {
		name         string
		unofficial   bool
		metadataForm bool
		exclusion    string
		valid        bool
	}{
		{"verified unofficial template", true, false, "blank", true},
		{"official form cannot be hidden", false, false, "blank", false},
		{"metadata form cannot be hidden", true, true, "blank", false},
		{"unused exclusion rejected", true, false, "absent", false},
		{"standard exclusion rejected", true, false, "base", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			lock, cache, m := automaticFixture(t)
			m.RulesVersion = "d06-auto-3"
			table := "identifier,pokemon_id\nbulbasaur,1\n"
			if tc.metadataForm {
				table += "bulbasaur-blank,1\n"
			}
			fixtureInput(t, &lock, cache, "pokeapi", "data/v2/csv/pokemon_forms.csv", []byte(table))
			unofficial := "false"
			if tc.unofficial {
				unofficial = "true"
			}
			fixtureInput(t, &lock, cache, "pokesprite-v2", "sources/upstreams/msikma-pokemon.json", []byte(`{"001":{"idx":"001","slug":{"eng":"bulbasaur"},"gen-8":{"forms":{"$":{},"mega":{},"blank":{"is_unofficial_icon":`+unofficial+`}}}}}`))
			editManifest(t, &lock, cache, func(s *sourceManifest) {
				yes, no := true, false
				s.Pokemon[0].Forms = append(s.Pokemon[0].Forms, sourceForm{ID: "blank", Label: "Blank", Slug: "bulbasaur-blank", HasRegular: &yes, HasShiny: &yes, IsGenerated: &no, Source: "msikma/pokesprite"})
			})
			m.SourceFormExclusions = []sourceFormExclusion{{SpeciesID: 1, SourceFormID: tc.exclusion, Reason: "Fixture source template is unofficial and has no metadata form"}}
			resolved, e := deriveMappings(lock, cache, m)
			if !tc.valid {
				if e == nil {
					t.Fatal("accepted unsupported exclusion")
				}
				return
			}
			if e != nil {
				t.Fatal(e)
			}
			for _, f := range resolved.Forms {
				if f.SourceFormID == "blank" {
					t.Fatal("template became a collectible form")
				}
			}
			if !strings.Contains(strings.Join(resolved.Exclusions, "\n"), "#001/blank") {
				t.Fatal("excluded source record vanished from coverage reasons")
			}
		})
	}
}

func TestUnsupportedMetadataFormIsDerivedAsExclusion(t *testing.T) {
	lock, cache, m := automaticFixture(t)
	m.RulesVersion = "d06-auto-4"
	if _, e := deriveMappings(lock, cache, m); e == nil {
		t.Fatal("accepted missing required v4 metadata tables")
	}
	fixtureInput(t, &lock, cache, "pokeapi", "data/v2/csv/pokemon_forms.csv", []byte("id,identifier,pokemon_id,is_default\n1,bulbasaur,1,1\n2,ivysaur,2,1\n3,venusaur,3,1\n10000,bulbasaur-mega,100,1\n"))
	fixtureInput(t, &lock, cache, "pokeapi", "data/v2/csv/pokemon_form_types.csv", []byte("pokemon_form_id,type_id,slot\n10000,10001,1\n"))
	data, _, _, e := readInput(lock, cache, "pokeapi", "data/v2/csv/types.csv")
	if e != nil {
		t.Fatal(e)
	}
	fixtureInput(t, &lock, cache, "pokeapi", "data/v2/csv/types.csv", append(data, []byte("10001,unknown\n")...))
	resolved, e := deriveMappings(lock, cache, m)
	if e != nil {
		t.Fatal(e)
	}
	if len(resolved.Forms) != 3 || len(resolved.SourceFormExclusions) != 1 || resolved.SourceFormExclusions[0].UnsupportedType != "unknown" {
		t.Fatalf("unsupported appearance not excluded explicitly: %+v", resolved)
	}
	for _, a := range resolved.Assets {
		if a.FormID == "mega" {
			t.Fatal("unsupported typing remained eligible")
		}
	}
	m.SourceFormExclusions = resolved.SourceFormExclusions
	if _, e := deriveMappings(lock, cache, m); e == nil {
		t.Fatal("accepted manually invented unsupported-type evidence")
	}
}
