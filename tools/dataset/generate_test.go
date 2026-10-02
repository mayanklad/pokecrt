package main

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func encodeFixturePNG(t *testing.T, partial bool) []byte {
	t.Helper()
	pixels := image.NewNRGBA(image.Rect(0, 0, 6, 7))
	pixels.SetNRGBA(1, 2, color.NRGBA{R: 255, A: 255})
	pixels.SetNRGBA(3, 4, color.NRGBA{B: 255, A: 255})
	if partial {
		pixels.SetNRGBA(2, 3, color.NRGBA{G: 255, A: 127})
	}
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, pixels); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func fixtureDataset(t *testing.T) (sourceLock, string, mappingConfig) {
	t.Helper()
	cache := t.TempDir()
	lock := sourceLock{SchemaVersion: 1, Sources: []source{
		{ID: "pokeapi", Repository: "example/metadata", Revision: strings.Repeat("a", 40), Terms: "Fixture metadata terms", Attribution: "Fixture"},
		{ID: "pokesprite-v2", Repository: "example/artwork", Revision: strings.Repeat("b", 40), Terms: "Fixture artwork terms", Attribution: "Fixture"},
	}}
	inputs := map[string][]byte{
		"pokeapi/data/v2/csv/generations.csv":              []byte("id,identifier\n1,generation-i\n"),
		"pokeapi/data/v2/csv/pokemon_colors.csv":           []byte("id,identifier\n5,green\n"),
		"pokeapi/data/v2/csv/types.csv":                    []byte("id,identifier\n12,grass\n4,poison\n"),
		"pokeapi/data/v2/csv/pokemon_species.csv":          []byte("id,identifier,generation_id,evolves_from_species_id,color_id,is_baby,is_legendary,is_mythical,has_gender_differences\n1,bulbasaur,1,,5,0,0,0,0\n2,ivysaur,1,1,5,0,0,0,0\n3,venusaur,1,2,5,0,0,0,1\n"),
		"pokeapi/data/v2/csv/pokemon_species_names.csv":    []byte("pokemon_species_id,local_language_id,name\n1,9,Bulbasaur\n2,9,Ivysaur\n3,9,Venusaur\n"),
		"pokeapi/data/v2/csv/pokemon.csv":                  []byte("id,species_id,is_default\n1,1,1\n2,2,1\n3,3,1\n"),
		"pokeapi/data/v2/csv/pokemon_types.csv":            []byte("pokemon_id,type_id,slot\n1,12,1\n1,4,2\n2,12,1\n2,4,2\n3,12,1\n3,4,2\n"),
		"pokeapi/LICENSE.md":                               []byte("Fixture metadata notice\n"),
		"pokesprite-v2/license.md":                         []byte("Fixture code notice\n"),
		"pokesprite-v2/contributors.md":                    []byte("Fixture credits\n"),
		"pokesprite-v2/pokemon/regular/bulbasaur.png":      encodeFixturePNG(t, false),
		"pokesprite-v2/sources/generated/asset-index.json": []byte(`{"_meta":{"generated_at":"excluded from identity"},"bulbasaur":{"regular":{"source":"msikma/pokesprite","is_generated":false}}}`),
	}
	manifest := sourceManifest{}
	for i, name := range []string{"Bulbasaur", "Ivysaur", "Venusaur"} {
		yes, no := true, false
		slug := strings.ToLower(name)
		manifest.Pokemon = append(manifest.Pokemon, sourceSpecies{ID: i + 1, Name: name, Slug: slug, Aliases: []string{name}, DefaultForm: "base", Forms: []sourceForm{{ID: "base", Slug: slug, HasRegular: &yes, IsGenerated: &no, Source: "msikma/pokesprite"}}})
	}
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	inputs["pokesprite-v2/data/pokemon.json"] = manifestJSON
	for path, data := range inputs {
		parts := strings.SplitN(path, "/", 2)
		fixtureInput(t, &lock, cache, parts[0], parts[1], data)
	}
	mappings := mappingConfig{RulesVersion: "d02b-1", CatalogSpecies: []int{1, 2, 3}, SourceStandardForm: "base", StandardFormReason: "Fixture base-to-standard mapping", Assets: []assetMapping{{SpeciesID: 1, SourceID: "pokesprite-v2", Path: "pokemon/regular/bulbasaur.png", SourceSlug: "bulbasaur", Reason: "Fixture regular asset"}}, Exclusions: []string{"Fixture excludes other appearances"}}
	return lock, cache, mappings
}

func fixtureInput(t *testing.T, lock *sourceLock, cache, sourceID, path string, data []byte) {
	t.Helper()
	filename := filepath.Join(cache, sourceID, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, data, 0o600); err != nil {
		t.Fatal(err)
	}
	file := sourceFile{Path: path, Size: int64(len(data)), SHA256: digest(data)}
	for i := range lock.Sources {
		if lock.Sources[i].ID != sourceID {
			continue
		}
		for j := range lock.Sources[i].Files {
			if lock.Sources[i].Files[j].Path == path {
				lock.Sources[i].Files[j] = file
				return
			}
		}
		lock.Sources[i].Files = append(lock.Sources[i].Files, file)
		return
	}
	t.Fatalf("fixture source missing: %s", sourceID)
}

func TestTransparentCropPreservesPixelsAndOddHeight(t *testing.T) {
	data, bounds, size, err := normalizePNG(encodeFixturePNG(t, false))
	if err != nil {
		t.Fatal(err)
	}
	if bounds != image.Rect(1, 2, 4, 5) || size != image.Pt(6, 7) {
		t.Fatalf("bounds=%v source size=%v", bounds, size)
	}
	decoded, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Bounds().Size() != image.Pt(3, 3) {
		t.Fatalf("cropped size=%v", decoded.Bounds().Size())
	}
	if got := color.NRGBAModel.Convert(decoded.At(0, 0)); got != (color.NRGBA{R: 255, A: 255}) {
		t.Fatalf("top-left pixel changed: %v", got)
	}
	if got := color.NRGBAModel.Convert(decoded.At(2, 2)); got != (color.NRGBA{B: 255, A: 255}) {
		t.Fatalf("bottom-right pixel changed: %v", got)
	}
	if _, _, _, err := normalizePNG(encodeFixturePNG(t, true)); err == nil {
		t.Fatal("partial alpha accepted without a normalization rule")
	}
	var transparent bytes.Buffer
	if err := png.Encode(&transparent, image.NewNRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := normalizePNG(transparent.Bytes()); err == nil {
		t.Fatal("fully transparent PNG accepted")
	}
}

func TestGenerationIsDeterministicAndChecksDoNotOverwrite(t *testing.T) {
	lock, cache, mappings := fixtureDataset(t)
	first, err := buildBundle(lock, cache, mappings)
	if err != nil {
		t.Fatal(err)
	}
	second, err := buildBundle(lock, cache, mappings)
	if err != nil {
		t.Fatal(err)
	}
	if first.DatasetID != second.DatasetID || !reflect.DeepEqual(first.Files, second.Files) {
		t.Fatal("repeated generation changed identity or outputs")
	}
	fixtureInput(t, &lock, cache, "pokesprite-v2", "sources/generated/asset-index.json", []byte(`{"_meta":{"generated_at":"different informational timestamp"},"bulbasaur":{"regular":{"source":"msikma/pokesprite","is_generated":false}}}`))
	withoutTimestampDrift, err := buildBundle(lock, cache, mappings)
	if err != nil || withoutTimestampDrift.DatasetID != first.DatasetID {
		t.Fatalf("informational timestamp changed dataset identity: %v", err)
	}
	if first.Coverage.CatalogSpecies != 3 || first.Coverage.EligibleSpecies != 1 || len(first.Coverage.Missing) != 2 {
		t.Fatalf("coverage conflates catalog and eligible species: %+v", first.Coverage)
	}
	root := t.TempDir()
	if err := writeBundle(first, root); err != nil {
		t.Fatal(err)
	}
	if err := checkBundle(first, root); err != nil {
		t.Fatal(err)
	}
	filename := filepath.Join(root, "internal/catalog/generated.go")
	tampered := []byte("deliberate output drift\n")
	if err := os.WriteFile(filename, tampered, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := checkBundle(first, root); err == nil || !strings.Contains(err.Error(), "drift") {
		t.Fatalf("drift accepted: %v", err)
	}
	after, err := os.ReadFile(filename)
	if err != nil || !bytes.Equal(after, tampered) {
		t.Fatal("check overwrote the drifted output")
	}
	if err := os.WriteFile(filepath.Join(root, "internal/sprite/assets/unmapped.png"), []byte("unexpected"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeBundle(first, root); err == nil {
		t.Fatal("unmapped asset silently accepted")
	}
}

func TestGenerationRejectsBadSourceData(t *testing.T) {
	tests := []struct {
		name string
		edit func(*testing.T, *sourceLock, string, *mappingConfig)
	}{
		{"duplicate species", func(t *testing.T, l *sourceLock, c string, _ *mappingConfig) {
			filename := "data/v2/csv/pokemon_species.csv"
			data, _, _, err := readInput(*l, c, "pokeapi", filename)
			if err != nil {
				t.Fatal(err)
			}
			fixtureInput(t, l, c, "pokeapi", filename, append(data, []byte("1,bulbasaur,1,,5,0,0,0,0\n")...))
		}},
		{"cycle", func(t *testing.T, l *sourceLock, c string, _ *mappingConfig) {
			filename := "data/v2/csv/pokemon_species.csv"
			data, _, _, err := readInput(*l, c, "pokeapi", filename)
			if err != nil {
				t.Fatal(err)
			}
			fixtureInput(t, l, c, "pokeapi", filename, bytes.Replace(data, []byte("1,bulbasaur,1,,"), []byte("1,bulbasaur,1,3,"), 1))
		}},
		{"generated provenance", func(t *testing.T, l *sourceLock, c string, _ *mappingConfig) {
			fixtureInput(t, l, c, "pokesprite-v2", "sources/generated/asset-index.json", []byte(`{"bulbasaur":{"regular":{"source":"msikma/pokesprite","is_generated":true}}}`))
		}},
		{"invalid PNG", func(t *testing.T, l *sourceLock, c string, _ *mappingConfig) {
			fixtureInput(t, l, c, "pokesprite-v2", "pokemon/regular/bulbasaur.png", []byte("not a PNG"))
		}},
		{"ambiguous alias", func(t *testing.T, l *sourceLock, c string, _ *mappingConfig) {
			data, _, _, err := readInput(*l, c, "pokesprite-v2", "data/pokemon.json")
			if err != nil {
				t.Fatal(err)
			}
			var manifest sourceManifest
			if err := json.Unmarshal(data, &manifest); err != nil {
				t.Fatal(err)
			}
			manifest.Pokemon[1].Aliases = []string{"Bulbasaur"}
			data, err = json.Marshal(manifest)
			if err != nil {
				t.Fatal(err)
			}
			fixtureInput(t, l, c, "pokesprite-v2", "data/pokemon.json", data)
		}},
		{"missing exact asset", func(t *testing.T, _ *sourceLock, c string, _ *mappingConfig) {
			if err := os.Remove(filepath.Join(c, "pokesprite-v2/pokemon/regular/bulbasaur.png")); err != nil {
				t.Fatal(err)
			}
		}},
		{"invalid type cardinality", func(t *testing.T, l *sourceLock, c string, _ *mappingConfig) {
			path := "data/v2/csv/pokemon_types.csv"
			data, _, _, err := readInput(*l, c, "pokeapi", path)
			if err != nil {
				t.Fatal(err)
			}
			fixtureInput(t, l, c, "pokeapi", path, append(data, []byte("1,4,3\n")...))
		}},
		{"unclosed evolution family", func(_ *testing.T, _ *sourceLock, _ string, m *mappingConfig) { m.CatalogSpecies = []int{1} }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lock, cache, mappings := fixtureDataset(t)
			test.edit(t, &lock, cache, &mappings)
			if _, err := buildBundle(lock, cache, mappings); err == nil {
				t.Fatal("invalid fixture generated a bundle")
			}
		})
	}
}

func TestEvolutionStageBranchingAndMissingReference(t *testing.T) {
	stages, err := evolutionStages(map[int]int{1: 0, 2: 1, 3: 1, 4: 2})
	if err != nil || !reflect.DeepEqual(stages, map[int]int{1: 1, 2: 2, 3: 2, 4: 3}) {
		t.Fatalf("stages=%v error=%v", stages, err)
	}
	if _, err := evolutionStages(map[int]int{1: 99}); err == nil {
		t.Fatal("missing evolution reference accepted")
	}
}

func TestPrepareAssetsFreshCheckoutAndDrift(t *testing.T) {
	root := t.TempDir()
	bundle := generatedBundle{Files: map[string][]byte{
		"internal/catalog/generated.go":      []byte("metadata"),
		"internal/sprite/assets/fixture.png": []byte("image"),
	}}
	metadata := filepath.Join(root, "internal/catalog/generated.go")
	if err := os.MkdirAll(filepath.Dir(metadata), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(metadata, []byte("drift"), 0644); err != nil {
		t.Fatal(err)
	}
	asset := filepath.Join(root, "internal/sprite/assets/fixture.png")
	if err := prepareAssets(bundle, root); err == nil {
		t.Fatal("accepted metadata drift")
	}
	if _, err := os.Stat(asset); !os.IsNotExist(err) {
		t.Fatal("wrote sprite before metadata verification")
	}
	if err := os.WriteFile(metadata, []byte("metadata"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := prepareAssets(bundle, root); err != nil {
		t.Fatal(err)
	}
	if err := checkBundle(bundle, root); err != nil {
		t.Fatal(err)
	}
	extra := filepath.Join(root, "internal/sprite/assets/unexpected.png")
	if err := os.WriteFile(extra, []byte("unexpected"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := prepareAssets(bundle, root); err == nil {
		t.Fatal("accepted unexpected sprite")
	}
}

func TestAchievementInventoryUsesEligibleFormsAndCompleteFamilies(t *testing.T) {
	species := []normalizedSpecies{
		{ID: 1, Generation: 1, Forms: []normalizedForm{{ID: "standard", Types: []string{"normal"}}, {ID: "mega", Types: []string{"dragon"}, Tags: []string{"mega"}}}},
		{ID: 2, Generation: 2, EvolvesFrom: 1, Forms: []normalizedForm{{ID: "regional", Types: []string{"fire"}, Tags: []string{"regional"}}}},
		{ID: 3, Generation: 2, EvolvesFrom: 1, Forms: []normalizedForm{{ID: "standard", Types: []string{"water"}}}},
		{ID: 4, Generation: 3, Forms: []normalizedForm{{ID: "gmax", Types: []string{"grass"}, Tags: []string{"gigantamax"}}}},
		{ID: 5, Generation: 3, EvolvesFrom: 4},
	}
	eligible := map[int]bool{1: true, 2: true, 3: true, 4: true, 5: true}
	forms := map[string]bool{"1/standard": true, "2/regional": true, "3/standard": true, "4/gmax": true}
	got := deriveAchievementInventory(species, eligible, forms, 1)
	if !reflect.DeepEqual(got.Types, []string{"fire", "grass", "normal", "water"}) || got.RegionalForms != 1 || got.TransformationForms != 1 || got.NationalSpecies != 5 || !reflect.DeepEqual(got.BranchingFamilies, [][]int{{1, 2, 3}}) || len(got.Limitations) != 0 {
		t.Fatalf("inventory=%+v", got)
	}
	if !reflect.DeepEqual(got.GenerationSpecies, map[int]int{1: 1, 2: 2, 3: 2}) {
		t.Fatalf("generations=%v", got.GenerationSpecies)
	}
	// Multiple palettes/genders do not add form or species targets; asset sets
	// are deduplicated before this stage. An unavailable branch excludes the
	// entire family, rather than shrinking its completion denominator.
	delete(eligible, 3)
	got = deriveAchievementInventory(species, eligible, forms, 1)
	if len(got.BranchingFamilies) != 0 || got.NationalSpecies != 4 {
		t.Fatalf("partial family qualified: %+v", got)
	}
	// Preserve ordering regardless of metadata input order.
	for i, j := 0, len(species)-1; i < j; i, j = i+1, j-1 {
		species[i], species[j] = species[j], species[i]
	}
	if other := deriveAchievementInventory(species, eligible, forms, 1); !reflect.DeepEqual(got, other) {
		t.Fatalf("order changed targets: %+v", other)
	}
	empty := deriveAchievementInventory(nil, nil, nil, 0)
	if len(empty.Limitations) != 6 || len(empty.Types) != 0 || empty.NationalSpecies != 0 {
		t.Fatalf("impossible goals not reported: %+v", empty)
	}
}

func TestCoverageDoesNotCountShinyOnlyOrMetadataOnlyFormsAsTargets(t *testing.T) {
	species := []normalizedSpecies{{ID: 1, Generation: 1, Forms: []normalizedForm{{ID: "standard", DefaultGender: "default", Genders: []string{"default"}, Types: []string{"normal"}}, {ID: "mega", Tags: []string{"mega"}, Types: []string{"dragon"}}, {ID: "regional", Tags: []string{"regional"}, Types: []string{"fire"}}}}}
	assets := []normalizedAsset{{SpeciesID: 1, FormID: "standard", Gender: "default", Palette: "regular"}, {SpeciesID: 1, FormID: "standard", Gender: "default", Palette: "shiny"}, {SpeciesID: 1, FormID: "regional", Gender: "default", Palette: "shiny"}}
	got := makeCoverage(species, assets, mappingConfig{}, "fixture").AchievementInventory
	if got.RegionalForms != 0 || got.TransformationForms != 0 || got.NationalSpecies != 1 || !reflect.DeepEqual(got.Types, []string{"normal"}) {
		t.Fatalf("unavailable appearance became a target: %+v", got)
	}
}
