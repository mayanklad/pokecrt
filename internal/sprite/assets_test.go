package sprite

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"slices"
	"testing"

	"github.com/mayanklad/pokecrt/internal/catalog"
)

func TestEmbeddedInventoryMatchesManifest(t *testing.T) {
	if DatasetID != catalog.DatasetID || len(Inventory()) == 0 {
		t.Fatal("missing or inconsistent embedded dataset")
	}
	for _, asset := range Inventory() {
		data, err := files.ReadFile(asset.Path)
		if err != nil {
			t.Fatal(err)
		}
		hash := sha256.Sum256(data)
		if hex.EncodeToString(hash[:]) != asset.SHA256 {
			t.Fatalf("content hash mismatch: %s", asset.Path)
		}
		decoded, err := Decode(asset.Key)
		if err != nil || decoded.Bounds().Dx() != asset.Width || decoded.Bounds().Dy() != asset.Height {
			t.Fatalf("decode/dimension mismatch: %s: %v", asset.Path, err)
		}
		if _, ok := catalog.ByNumber(asset.Key.SpeciesID); !ok {
			t.Fatalf("asset has unknown species: %+v", asset.Key)
		}
	}
}

func TestAbsentExactVariantHasNoFallback(t *testing.T) {
	key := catalog.VariantKey{SpeciesID: 6, FormID: "unsupported-test-form", Gender: "default", Palette: "regular"}
	if _, ok := Lookup(key); ok {
		t.Fatal("absent exact variant resolved")
	}
	if _, err := Decode(key); err == nil {
		t.Fatal("absent variant received fallback artwork")
	}
}

// Validate the public consumer contract between three independently generated
// artifacts: catalog, embedded manifest and the machine-readable coverage report.
func TestCoverageAgreesWithEmbeddedPublicInventory(t *testing.T) {
	data, err := os.ReadFile("../../tools/dataset/coverage.json")
	if err != nil {
		t.Fatal(err)
	}
	var report struct {
		DatasetID       string            `json:"dataset_id"`
		CatalogSpecies  int               `json:"catalog_species"`
		CatalogForms    int               `json:"catalog_forms"`
		EligibleSpecies int               `json:"eligible_species"`
		StandardRegular int               `json:"standard_regular_sprites"`
		Shiny           int               `json:"shiny_sprites"`
		Forms           int               `json:"collectible_forms"`
		Genders         int               `json:"distinct_visual_gender_slots"`
		Variants        int               `json:"exact_eligible_variants"`
		Missing         []json.RawMessage `json:"missing_standard_regular_assets"`
		MissingVariants []json.RawMessage `json:"missing_regular_variants"`
		Achievements    struct {
			National    int         `json:"national_species"`
			Generations map[int]int `json:"generation_species"`
			Families    [][]int     `json:"branching_families"`
		} `json:"achievement_inventory"`
	}
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if report.DatasetID != DatasetID {
		t.Fatal("coverage belongs to a different embedded dataset")
	}
	assets := Inventory()
	keys := map[catalog.VariantKey]Asset{}
	eligible := map[int]bool{}
	forms := map[catalog.VariantKey]bool{}
	standard, shiny, genders := 0, 0, 0
	for _, a := range assets {
		if _, duplicate := keys[a.Key]; duplicate {
			t.Fatalf("duplicate collectible key: %+v", a.Key)
		}
		keys[a.Key] = a
		species, ok := catalog.ByNumber(a.Key.SpeciesID)
		if !ok {
			t.Fatal("unknown asset species")
		}
		owned := false
		for _, f := range species.Forms {
			if f.ID == a.Key.FormID && slices.Contains(f.Genders, a.Key.Gender) {
				owned = true
			}
		}
		if !owned {
			t.Fatalf("asset outside declared form/gender inventory: %+v", a.Key)
		}
		switch a.Key.Palette {
		case "regular":
			eligible[a.Key.SpeciesID] = true
			forms[catalog.VariantKey{SpeciesID: a.Key.SpeciesID, FormID: a.Key.FormID}] = true
			if a.Key.FormID == "standard" {
				standard++
			}
			if a.Key.Gender != "default" {
				genders++
			}
		case "shiny":
			shiny++
		default:
			t.Fatalf("unknown palette: %+v", a.Key)
		}
	}
	for key := range keys {
		if key.Palette == "shiny" {
			regular := key
			regular.Palette = "regular"
			if _, ok := keys[regular]; !ok {
				t.Fatalf("shiny-only collectible: %+v", key)
			}
		}
	}
	species := catalog.All()
	metadataForms, missingVariants := 0, 0
	generationCounts := map[int]int{}
	for _, s := range species {
		if eligible[s.ID] {
			generationCounts[s.Generation]++
		}
		metadataForms += len(s.Forms)
		for _, f := range s.Forms {
			for _, gender := range f.Genders {
				if _, ok := keys[catalog.VariantKey{SpeciesID: s.ID, FormID: f.ID, Gender: gender, Palette: "regular"}]; !ok {
					missingVariants++
				}
			}
			if len(f.Genders) == 2 {
				male, maleOK := keys[catalog.VariantKey{SpeciesID: s.ID, FormID: f.ID, Gender: "male", Palette: "regular"}]
				female, femaleOK := keys[catalog.VariantKey{SpeciesID: s.ID, FormID: f.ID, Gender: "female", Palette: "regular"}]
				if maleOK != femaleOK || maleOK && male.SHA256 == female.SHA256 {
					t.Fatalf("partial or fabricated gender pair #%d/%s", s.ID, f.ID)
				}
			}
		}
	}
	matches, err := catalog.Query(catalog.Selection{}, func(k catalog.VariantKey) bool { _, ok := keys[k]; return ok })
	if err != nil {
		t.Fatal(err)
	}
	printable := 0
	for _, m := range matches {
		if m.Available {
			printable++
		}
	}
	for _, tc := range []struct {
		name             string
		reported, actual int
	}{
		{"catalog species", report.CatalogSpecies, len(species)}, {"public rows", report.CatalogSpecies, len(matches)},
		{"catalog forms", report.CatalogForms, metadataForms}, {"eligible species", report.EligibleSpecies, len(eligible)},
		{"standard regular slots", report.StandardRegular, standard}, {"shiny slots", report.Shiny, shiny},
		{"collectible forms", report.Forms, len(forms)}, {"gender slots", report.Genders, genders}, {"variants", report.Variants, len(assets)},
		{"missing standards", len(report.Missing), len(matches) - printable}, {"missing variants", len(report.MissingVariants), missingVariants},
		{"national target", report.Achievements.National, len(eligible)},
	} {
		if tc.reported != tc.actual {
			t.Errorf("%s: coverage=%d embedded/query=%d", tc.name, tc.reported, tc.actual)
		}
	}
	if len(report.Achievements.Generations) != len(generationCounts) {
		t.Fatal("generation denominator scope differs")
	}
	for gen, n := range generationCounts {
		if report.Achievements.Generations[gen] != n {
			t.Fatalf("generation %d denominator differs", gen)
		}
	}
	for _, family := range report.Achievements.Families {
		members := map[int]bool{}
		branch := false
		for _, id := range family {
			if members[id] || !eligible[id] {
				t.Fatalf("duplicate or unavailable family member #%d", id)
			}
			members[id] = true
		}
		for _, id := range family {
			s, _ := catalog.ByNumber(id)
			if len(s.EvolvesTo) > 1 {
				branch = true
			}
			if s.EvolvesFrom > 0 && !members[s.EvolvesFrom] {
				t.Fatal("branching family trimmed a predecessor")
			}
			for _, child := range s.EvolvesTo {
				if !members[child] {
					t.Fatal("branching family trimmed a descendant")
				}
			}
		}
		if !branch {
			t.Fatal("completion target has no species branch")
		}
	}
}

func BenchmarkLookup(b *testing.B) {
	inventory := Inventory()
	for _, tc := range []struct {
		name  string
		key   catalog.VariantKey
		found bool
	}{
		{"First", inventory[0].Key, true},
		{"Last", inventory[len(inventory)-1].Key, true},
		{"Missing", catalog.VariantKey{SpeciesID: -1}, false},
	} {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, ok := Lookup(tc.key); ok != tc.found {
					b.Fatal("lookup result changed")
				}
			}
		})
	}
}

func BenchmarkDecode(b *testing.B) {
	species, ok := catalog.ByName("charizard")
	if !ok {
		b.Fatal("missing benchmark species")
	}
	key, ok := catalog.StandardKey(species)
	if !ok {
		b.Fatal("missing benchmark form")
	}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := Decode(key); err != nil {
			b.Fatal(err)
		}
	}
}
