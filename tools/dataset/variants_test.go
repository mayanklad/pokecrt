package main

import (
	"bytes"
	"encoding/json"
	"image/color"
	"image/png"
	"reflect"
	"strings"
	"testing"
)

func variantFixture(t *testing.T) (sourceLock, string, mappingConfig) {
	t.Helper()
	lock, cache, m := fixtureDataset(t)
	m.RulesVersion = "d06a-1"
	for id := 1; id <= 3; id++ {
		m.Forms = append(m.Forms, formMapping{SpeciesID: id, ID: "standard", Name: "Standard", SourceFormID: "base", PokemonID: id, DefaultGender: "default", Genders: []string{"default"}, Reason: "Fixture exact standard identity"})
	}
	for i := range m.Assets {
		m.Assets[i].FormID = "standard"
		m.Assets[i].SourceFormID = "base"
		m.Assets[i].Gender = "default"
		m.Assets[i].Palette = "regular"
	}
	return lock, cache, m
}

func editManifest(t *testing.T, lock *sourceLock, cache string, edit func(*sourceManifest)) {
	t.Helper()
	data, _, _, err := readInput(*lock, cache, "pokesprite", "data/pokemon.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest sourceManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	edit(&manifest)
	data, err = json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	fixtureInput(t, lock, cache, "pokesprite", "data/pokemon.json", data)
}

func TestFormTypingAndSourceOnlyAliases(t *testing.T) {
	lock, cache, m := variantFixture(t)
	yes, no := true, false
	editManifest(t, &lock, cache, func(manifest *sourceManifest) {
		alias := "base"
		manifest.Pokemon[0].Forms = append(manifest.Pokemon[0].Forms, sourceForm{ID: "normal", Slug: "bulbasaur", CanonicalForm: &alias}, sourceForm{ID: "mega", Slug: "bulbasaur-mega", HasRegular: &yes, IsGenerated: &no, Source: "msikma/pokesprite"})
	})
	m.Forms[0].SourceAliases = []string{"normal"}
	m.Forms = append(m.Forms, formMapping{SpeciesID: 1, ID: "mega", Name: "Mega", SourceFormID: "mega", PokemonID: 100, DefaultGender: "default", Genders: []string{"default"}, Reason: "Fixture exact alternate variety"})
	for _, change := range []struct{ path, append string }{
		{"pokemon.csv", "100,1,0\n"},
		{"types.csv", "10,fire\n16,dragon\n"},
		{"pokemon_types.csv", "100,10,1\n100,16,2\n"},
	} {
		path := "data/v2/csv/" + change.path
		data, _, _, err := readInput(lock, cache, "pokeapi", path)
		if err != nil {
			t.Fatal(err)
		}
		fixtureInput(t, &lock, cache, "pokeapi", path, append(data, []byte(change.append)...))
	}
	bundle, err := buildBundle(lock, cache, m)
	if err != nil {
		t.Fatal(err)
	}
	species, _, err := normalizeCatalog(lock, cache, m)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(species[0].Forms[0].Types, []string{"grass", "poison"}) || !reflect.DeepEqual(species[0].Forms[1].Types, []string{"fire", "dragon"}) {
		t.Fatalf("form typing mixed: %+v", species[0].Forms)
	}
	if len(species[0].Forms) != 2 || len(species[0].Forms[0].SourceAliases) != 1 || bundle.Coverage.CatalogForms != 4 || bundle.Coverage.CollectibleForms != 1 {
		t.Fatalf("alias or metadata became collectible: %+v", bundle.Coverage)
	}
	m.Forms[len(m.Forms)-1].PokemonID = 2
	if _, err := buildBundle(lock, cache, m); err == nil {
		t.Fatal("accepted typing from another species")
	}
}

func TestMappedShinyNeedsRealDifferentPixels(t *testing.T) {
	lock, cache, m := variantFixture(t)
	yes := true
	editManifest(t, &lock, cache, func(manifest *sourceManifest) { manifest.Pokemon[0].Forms[0].HasShiny = &yes })
	fixtureInput(t, &lock, cache, "pokesprite", "sources/generated/asset-index.json", []byte(`{"bulbasaur":{"regular":{"source":"msikma/pokesprite","is_generated":false},"shiny":{"source":"msikma/pokesprite","is_generated":false}}}`))
	data := encodeFixturePNG(t, false)
	fixtureInput(t, &lock, cache, "pokesprite", "pokemon/shiny/bulbasaur.png", data)
	shiny := m.Assets[0]
	shiny.Palette = "shiny"
	shiny.Path = "pokemon/shiny/bulbasaur.png"
	m.Assets = append(m.Assets, shiny)
	if _, err := buildBundle(lock, cache, m); err == nil {
		t.Fatal("accepted regular pixels as shiny")
	}
	decoded, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	pixels := decoded.(interface{ Set(int, int, color.Color) })
	pixels.Set(1, 2, color.NRGBA{G: 255, A: 255})
	var different bytes.Buffer
	if err := png.Encode(&different, decoded); err != nil {
		t.Fatal(err)
	}
	fixtureInput(t, &lock, cache, "pokesprite", shiny.Path, different.Bytes())
	bundle, err := buildBundle(lock, cache, m)
	if err != nil {
		t.Fatal(err)
	}
	if bundle.Coverage.EligibleSpecies != 1 || bundle.Coverage.CollectibleForms != 1 || bundle.Coverage.ShinySprites != 1 || bundle.Coverage.ExactEligibleVariants != 2 {
		t.Fatalf("palette inflated species/forms: %+v", bundle.Coverage)
	}
	m.Assets = []assetMapping{shiny}
	if _, err := buildBundle(lock, cache, m); err == nil {
		t.Fatal("accepted shiny-only collectible")
	}
}

func TestVisualGenderInventory(t *testing.T) {
	species := []normalizedSpecies{{ID: 1, Name: "Fixture", Forms: []normalizedForm{{ID: "standard", DefaultGender: "male", Genders: []string{"female", "male"}}}}}
	assets := []normalizedAsset{{SpeciesID: 1, FormID: "standard", Gender: "male", Palette: "regular", SHA256: "first"}}
	if err := validateVariantInventory(species, assets); err == nil {
		t.Fatal("accepted missing female artwork")
	}
	female := assets[0]
	female.Gender = "female"
	assets = append(assets, female)
	if err := validateVariantInventory(species, assets); err == nil {
		t.Fatal("accepted duplicate gender pixels")
	}
	assets[1].SHA256 = "second"
	if err := validateVariantInventory(species, assets); err != nil {
		t.Fatal(err)
	}
	coverage := makeCoverage(species, assets, mappingConfig{}, "fixture")
	if coverage.EligibleSpecies != 1 || coverage.CollectibleForms != 1 || coverage.DistinctVisualGenderSlots != 2 || coverage.ExactEligibleVariants != 2 {
		t.Fatalf("bad gender inventory: %+v", coverage)
	}
}

func TestInvalidVariantMappings(t *testing.T) {
	for _, test := range []struct {
		name string
		edit func(*mappingConfig)
	}{
		{"duplicate identity", func(m *mappingConfig) { m.Assets = append(m.Assets, m.Assets[0]) }},
		{"path traversal", func(m *mappingConfig) { m.Assets[0].FormID = "../other" }},
		{"wrong palette path", func(m *mappingConfig) { m.Assets[0].Palette = "shiny" }},
		{"unmapped form", func(m *mappingConfig) { m.Forms = m.Forms[1:] }},
		{"invented biological gender", func(m *mappingConfig) { m.Forms[0].Genders = []string{"male"}; m.Forms[0].DefaultGender = "male" }},
		{"unknown appearance", func(m *mappingConfig) { m.Assets[0].FormID = "mega" }},
		{"absent gender pixels", func(m *mappingConfig) {
			m.Forms[0].Genders = []string{"male", "female"}
			m.Forms[0].DefaultGender = "male"
			m.Assets[0].Gender = "male"
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			lock, cache, m := variantFixture(t)
			test.edit(&m)
			if _, err := buildBundle(lock, cache, m); err == nil {
				t.Fatal("invalid mapping accepted")
			}
		})
	}
	lock, cache, m := variantFixture(t)
	m.Assets[0].Reason = ""
	if _, err := buildBundle(lock, cache, m); err == nil || !strings.Contains(err.Error(), "mapping") {
		t.Fatalf("missing audit reason: %v", err)
	}
}

func TestSourceGenderFormFoldsIntoOneCollectibleForm(t *testing.T) {
	lock, cache, m := variantFixture(t)
	yes, no := true, false
	editManifest(t, &lock, cache, func(manifest *sourceManifest) {
		manifest.Pokemon[0].Forms = append(manifest.Pokemon[0].Forms, sourceForm{ID: "female", Slug: "bulbasaur-female", HasRegular: &yes, IsGenerated: &no, Source: "msikma/pokesprite"})
	})
	m.Forms[0].Genders = []string{"male", "female"}
	m.Forms[0].DefaultGender = "male"
	m.Assets[0].Gender = "male"
	female := m.Assets[0]
	female.Gender = "female"
	female.SourceFormID = "female"
	female.SourceSlug = "bulbasaur-female"
	female.Path = "pokemon/regular/bulbasaur-female.png"
	m.Assets = append(m.Assets, female)
	fixtureInput(t, &lock, cache, "pokesprite", "sources/generated/asset-index.json", []byte(`{"bulbasaur":{"regular":{"source":"msikma/pokesprite","is_generated":false}},"bulbasaur-female":{"regular":{"source":"msikma/pokesprite","is_generated":false}}}`))
	pixels, err := png.Decode(bytes.NewReader(encodeFixturePNG(t, false)))
	if err != nil {
		t.Fatal(err)
	}
	pixels.(interface{ Set(int, int, color.Color) }).Set(1, 2, color.NRGBA{G: 255, A: 255})
	var data bytes.Buffer
	if err := png.Encode(&data, pixels); err != nil {
		t.Fatal(err)
	}
	fixtureInput(t, &lock, cache, "pokesprite", female.Path, data.Bytes())
	bundle, err := buildBundle(lock, cache, m)
	if err != nil {
		t.Fatal(err)
	}
	if bundle.Coverage.CatalogForms != 3 || bundle.Coverage.CollectibleForms != 1 || bundle.Coverage.DistinctVisualGenderSlots != 2 || bundle.Coverage.ExactEligibleVariants != 2 {
		t.Fatalf("gender became a collectible form: %+v", bundle.Coverage)
	}
}
