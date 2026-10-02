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

func TestFormSuffixRulesRequireExactOwnedMetadata(t *testing.T) {
	m := mappingConfig{FormSuffixRules: []formSuffixRule{{SpeciesID: 869, Suffix: "-sweet", Reason: "Reviewed source naming discrepancy"}}}
	species := map[int]sourceSpecies{869: {ID: 869, Slug: "alcremie", DefaultForm: "base", Forms: []sourceForm{{ID: "base"}, {ID: "ruby-cream-berry"}, {ID: "ruby-cream-plain"}}}}
	names := map[string]int{"alcremie-ruby-cream-berry-sweet": 869}
	owners := map[int]int{869: 869}
	forms := map[string]int{"alcremie-ruby-cream-berry-sweet": 10450}
	if err := applyFormSuffixRules(m, []int{869}, species, names, owners, forms); err != nil {
		t.Fatal(err)
	}
	if names["alcremie-ruby-cream-berry"] != 869 || forms["alcremie-ruby-cream-berry"] != 10450 || names["alcremie-ruby-cream-plain"] != 0 {
		t.Fatal("rule guessed an unmatched identity")
	}
	for _, tc := range []struct{ owner, form int }{{1, 10450}, {869, 0}} {
		fresh := map[string]int{"alcremie-ruby-cream-berry-sweet": 869}
		if err := applyFormSuffixRules(m, []int{869}, species, fresh, map[int]int{869: tc.owner}, map[string]int{"alcremie-ruby-cream-berry-sweet": tc.form}); err == nil {
			t.Fatal("accepted missing or wrong-owner metadata")
		}
	}
	if err := applyFormSuffixRules(m, []int{869}, species, map[string]int{}, owners, map[string]int{}); err == nil {
		t.Fatal("accepted unused suffix rule")
	}
	m.FormSuffixRules = append(m.FormSuffixRules, m.FormSuffixRules[0])
	if err := applyFormSuffixRules(m, []int{869}, species, map[string]int{}, owners, forms); err == nil {
		t.Fatal("accepted duplicate suffix rule")
	}
}

func TestReviewedDefaultAliasRequiresCompleteHashEvidence(t *testing.T) {
	hash := strings.Repeat("a", 64)
	alias := reviewedDefaultAlias{SpeciesID: 1, SourceFormID: "duplicate", Reason: "Reviewed same metadata identity", SourceHashes: map[string]string{"regular": hash, "shiny": hash}, DefaultHashes: map[string]string{"regular": hash, "shiny": hash}}
	fixture := func() sourceManifest {
		return sourceManifest{Pokemon: []sourceSpecies{{ID: 1, DefaultForm: "base", Forms: []sourceForm{{ID: "base", Slug: "bulbasaur"}, {ID: "duplicate", Slug: "bulbasaur-duplicate"}}}}}
	}
	manifest := fixture()
	if err := applyReviewedDefaultAliases(&manifest, mappingConfig{ReviewedDefaultAliases: []reviewedDefaultAlias{alias}}); err != nil {
		t.Fatal(err)
	}
	folded := manifest.Pokemon[0].Forms[1]
	if folded.CanonicalForm == nil || *folded.CanonicalForm != "base" || folded.Slug != "bulbasaur" {
		t.Fatal("did not fold reviewed identity")
	}
	for _, id := range []string{"base", "absent"} {
		bad := alias
		bad.SourceFormID = id
		manifest = fixture()
		if err := applyReviewedDefaultAliases(&manifest, mappingConfig{ReviewedDefaultAliases: []reviewedDefaultAlias{bad}}); err == nil {
			t.Fatal("accepted invalid alias")
		}
	}
	alias.SourceHashes = map[string]string{"regular": hash}
	manifest = fixture()
	if err := applyReviewedDefaultAliases(&manifest, mappingConfig{ReviewedDefaultAliases: []reviewedDefaultAlias{alias}}); err == nil {
		t.Fatal("accepted missing shiny hash evidence")
	}
}

func TestAutomaticInheritedGenderDeclarations(t *testing.T) {
	for _, name := range []string{"audited", "unofficial female", "unofficial base", "biological only", "wrong owner", "wrong slug"} {
		t.Run(name, func(t *testing.T) {
			lock, cache, m := automaticFixture(t)
			m.RulesVersion = "d06-auto-8"
			m.Selection.AuditedInheritedGenders = true
			fixtureInput(t, &lock, cache, "pokeapi", "data/v2/csv/pokemon_forms.csv", []byte("id,identifier,pokemon_id,is_default\n1,bulbasaur,1,1\n2,ivysaur,2,1\n3,venusaur,3,1\n100,bulbasaur-mega,100,1\n"))
			fixtureInput(t, &lock, cache, "pokeapi", "data/v2/csv/pokemon_form_types.csv", []byte("pokemon_form_id,type_id,slot\n"))
			flags := `"has_female":true`
			idx, slug := "001", "bulbasaur"
			switch name {
			case "unofficial female":
				flags += `,"has_unofficial_female_icon":true`
			case "unofficial base":
				flags += `,"is_unofficial_icon":true`
			case "biological only":
				flags = `"has_female":false`
			case "wrong owner":
				idx = "002"
			case "wrong slug":
				slug = "other"
			}
			fixtureInput(t, &lock, cache, "pokesprite-v2", "sources/upstreams/msikma-pokemon.json", []byte(`{"001":{"idx":"`+idx+`","slug":{"eng":"`+slug+`"},"gen-8":{"forms":{"$":{`+flags+`},"mega":{}}}}}`))
			got, err := deriveMappings(lock, cache, m)
			if name == "wrong owner" || name == "wrong slug" {
				if err == nil {
					t.Fatal("accepted wrong inherited gender identity")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var female int
			for _, a := range got.Assets {
				if a.Gender == "female" {
					female++
					if a.FormID != "standard" || a.SourceLayout != "gen8-female" {
						t.Fatal("female became alternate collectible form")
					}
				}
			}
			if (female == 1) != (name == "audited") {
				t.Fatalf("female assets=%d", female)
			}
			if len(got.Forms) != 4 {
				t.Fatal("gender created another form")
			}
			if name == "audited" && got.Forms[0].DefaultGender != "male" {
				t.Fatal("lost male default")
			}
		})
	}
	m := mappingConfig{RulesVersion: "d06-auto-8", Selection: &inventorySelection{Generations: []int{1}, AuditedInheritedGenders: true, VisualGenderSpecies: []int{1}}, SourceStandardForm: "base", StandardFormReason: "fixture"}
	if validateSelection(m) == nil {
		t.Fatal("accepted maintained list alongside automatic genders")
	}
}

func TestTransformationAliasUsesMetadataWithoutBorrowedArtwork(t *testing.T) {
	for _, name := range []string{"distinct transformation", "ordinary alias", "wrong owner", "wrong filename", "unknown target", "invalid metadata flag"} {
		t.Run(name, func(t *testing.T) {
			lock, cache, m := automaticFixture(t)
			m.RulesVersion = "d06-auto-9"
			owner, mega := "1", "1"
			if name == "wrong owner" {
				owner = "2"
			}
			if name == "ordinary alias" {
				mega = "0"
			}
			if name == "invalid metadata flag" {
				mega = "2"
			}
			fixtureInput(t, &lock, cache, "pokeapi", "data/v2/csv/pokemon.csv", []byte("id,identifier,species_id,is_default\n1,bulbasaur,1,1\n2,ivysaur,2,1\n3,venusaur,3,1\n100,bulbasaur-mega,1,0\n101,bulbasaur-alternate-mega,"+owner+",0\n"))
			fixtureInput(t, &lock, cache, "pokeapi", "data/v2/csv/pokemon_forms.csv", []byte("id,identifier,pokemon_id,is_default,is_mega\n1,bulbasaur,1,1,0\n2,ivysaur,2,1,0\n3,venusaur,3,1,0\n100,bulbasaur-mega,100,1,1\n101,bulbasaur-alternate-mega,101,1,"+mega+"\n"))
			fixtureInput(t, &lock, cache, "pokeapi", "data/v2/csv/pokemon_form_types.csv", []byte("pokemon_form_id,type_id,slot\n"))
			fixtureInput(t, &lock, cache, "pokeapi", "data/v2/csv/pokemon_types.csv", []byte("pokemon_id,type_id,slot\n1,12,1\n1,4,2\n2,12,1\n2,4,2\n3,12,1\n3,4,2\n100,12,1\n101,12,1\n"))
			editManifest(t, &lock, cache, func(manifest *sourceManifest) {
				target := "base"
				slug := "bulbasaur"
				yes, no := true, false
				if name == "wrong filename" {
					slug = "other"
				}
				if name == "unknown target" {
					target = "missing"
				}
				manifest.Pokemon[0].Forms = append(manifest.Pokemon[0].Forms, sourceForm{ID: "alternate-mega", Label: "Alternate Mega", Slug: slug, CanonicalForm: &target, HasRegular: &yes, HasShiny: &yes, IsGenerated: &no, Source: "msikma/pokesprite"})
			})
			resolved, err := deriveMappings(lock, cache, m)
			if name != "distinct transformation" && name != "ordinary alias" {
				if err == nil {
					t.Fatal("accepted invalid transformation alias evidence")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var found *formMapping
			for i := range resolved.Forms {
				if resolved.Forms[i].SpeciesID == 1 && resolved.Forms[i].ID == "alternate-mega" {
					found = &resolved.Forms[i]
				}
			}
			if name == "ordinary alias" {
				if found != nil {
					t.Fatal("ordinary alias became a transformation")
				}
				return
			}
			if found == nil || !found.MetadataOnlyAlias || found.PokemonID != 101 || found.MetadataFormID != 101 {
				t.Fatal("lost exact metadata-only transformation")
			}
			for _, a := range resolved.Assets {
				if a.FormID == "alternate-mega" || a.SourceFormID == "alternate-mega" {
					t.Fatal("borrowed ordinary artwork for transformation")
				}
			}
			normalized, _, err := normalizeCatalog(lock, cache, resolved)
			if err != nil {
				t.Fatal(err)
			}
			if len(normalized[0].Forms) != 3 {
				t.Fatal("transformation not retained as distinct metadata")
			}
			borrowed := resolved.Assets[0]
			borrowed.FormID = "alternate-mega"
			borrowed.SourceFormID = "alternate-mega"
			resolved.Assets = append(resolved.Assets, borrowed)
			if _, _, err := normalizeCatalog(lock, cache, resolved); err == nil {
				t.Fatal("metadata-only identity accepted borrowed artwork")
			}
		})
	}
}

func TestSourceEncodedGenderPairIsOneUnavailableForm(t *testing.T) {
	for _, name := range []string{"verified metadata", "wrong owner", "wrong gender identifier", "missing female", "aliased female", "different typing", "no visual difference"} {
		t.Run(name, func(t *testing.T) {
			lock, cache, m := automaticFixture(t)
			m.RulesVersion = "d06-auto-10"
			owner, formGender, difference := "1", "female", "1"
			if name == "wrong owner" {
				owner = "2"
			}
			if name == "wrong gender identifier" {
				formGender = "other"
			}
			if name == "no visual difference" {
				difference = "0"
			}
			fixtureInput(t, &lock, cache, "pokeapi", "data/v2/csv/pokemon_species.csv", []byte("id,identifier,generation_id,evolves_from_species_id,color_id,is_baby,is_legendary,is_mythical,has_gender_differences\n1,bulbasaur,1,,5,0,0,0,"+difference+"\n2,ivysaur,1,1,5,0,0,0,0\n3,venusaur,1,2,5,0,0,0,1\n"))
			fixtureInput(t, &lock, cache, "pokeapi", "data/v2/csv/pokemon.csv", []byte("id,identifier,species_id,is_default\n1,bulbasaur-male,1,1\n2,ivysaur,2,1\n3,venusaur,3,1\n101,bulbasaur-female,"+owner+",0\n"))
			fixtureInput(t, &lock, cache, "pokeapi", "data/v2/csv/pokemon_forms.csv", []byte("id,identifier,pokemon_id,is_default,is_mega,form_identifier\n1,bulbasaur-male,1,1,0,male\n2,ivysaur,2,1,0,\n3,venusaur,3,1,0,\n101,bulbasaur-female,101,1,0,"+formGender+"\n"))
			fixtureInput(t, &lock, cache, "pokeapi", "data/v2/csv/pokemon_form_types.csv", []byte("pokemon_form_id,type_id,slot\n"))
			femaleTypes := "101,12,1\n101,4,2\n"
			if name == "different typing" {
				femaleTypes = "101,4,1\n"
			}
			fixtureInput(t, &lock, cache, "pokeapi", "data/v2/csv/pokemon_types.csv", []byte("pokemon_id,type_id,slot\n1,12,1\n1,4,2\n2,12,1\n2,4,2\n3,12,1\n3,4,2\n"+femaleTypes))
			editManifest(t, &lock, cache, func(manifest *sourceManifest) {
				yes := true
				male := manifest.Pokemon[0].Forms[0]
				male.ID = "male"
				male.Slug = "bulbasaur-male"
				male.IsGenerated = &yes
				male.Source = "generated/pokeapi"
				female := male
				female.ID = "female"
				female.Slug = "bulbasaur-female"
				if name == "aliased female" {
					target := "male"
					female.CanonicalForm = &target
					female.Slug = male.Slug
				}
				manifest.Pokemon[0].DefaultForm = "male"
				manifest.Pokemon[0].Forms = []sourceForm{male, female}
				if name == "missing female" {
					manifest.Pokemon[0].Forms = manifest.Pokemon[0].Forms[:1]
				}
			})
			resolved, err := deriveMappings(lock, cache, m)
			var normalized []normalizedSpecies
			if err == nil {
				normalized, _, err = normalizeCatalog(lock, cache, resolved)
			}
			if name != "verified metadata" {
				if err == nil {
					t.Fatal("accepted malformed source gender identity")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(normalized[0].Forms) != 1 || normalized[0].Forms[0].DefaultGender != "male" || !reflect.DeepEqual(normalized[0].Forms[0].Genders, []string{"female", "male"}) {
				t.Fatalf("gender became form: %+v", normalized[0].Forms)
			}
			for _, a := range resolved.Assets {
				if a.SpeciesID == 1 {
					t.Fatal("generated gender candidate accepted")
				}
			}
			if err := validateVariantInventory(normalized, nil); err != nil {
				t.Fatal(err)
			}
			// Even with a valid owner, an asset cannot swap the source's gender identity.
			resolved.Assets = append(resolved.Assets, assetMapping{SpeciesID: 1, FormID: "standard", SourceFormID: "male", Gender: "female", Palette: "regular"})
			if _, _, err := normalizeCatalog(lock, cache, resolved); err == nil {
				t.Fatal("accepted swapped source gender")
			}
		})
	}
}
