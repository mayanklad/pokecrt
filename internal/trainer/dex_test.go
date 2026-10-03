package trainer

import (
	"encoding/json"
	"github.com/mayanklad/pokecrt/internal/catalog"
	"github.com/mayanklad/pokecrt/internal/sprite"
	"strings"
	"testing"
)

func dexAvailable(k catalog.VariantKey) bool { _, ok := sprite.Lookup(k); return ok }
func dexFixture(form, palette string) DexRecords {
	return DexRecords{Species: map[int]Discovery{6: {Count: 1, FirstMS: 10, LastMS: 10}}, Encounters: 1, Variants: []VariantDiscovery{{Key: catalog.VariantKey{SpeciesID: 6, FormID: form, Gender: "default", Palette: palette}, Discovery: Discovery{Count: 1, FirstMS: 10, LastMS: 10}, FormName: "Observed " + form, Types: []string{"fire", "dragon"}}}}
}
func TestDexUnseenDominatesAndContainsNoHiddenFields(t *testing.T) {
	d := NewDex(catalog.All(), DexRecords{}, dexAvailable)
	for _, form := range []string{"standard", "mega-x", "alola"} {
		e, err := d.Entry(6, catalog.Selection{Form: form, Shiny: true, Gender: "female"})
		if err != nil || e.Seen || e.Name != "" || len(e.Forms) > 0 || len(e.Evolution) > 0 || e.ArtworkKey != nil || e.Generation != 0 {
			t.Fatalf("leak %+v %v", e, err)
		}
		b, _ := json.Marshal(e)
		if strings.Contains(string(b), "Charizard") || strings.Contains(string(b), "Dragon") {
			t.Fatal(string(b))
		}
	}
	rows := d.List(DexFilter{})
	if len(rows) != 1025 || rows[5].Name != "?????" {
		t.Fatal("anonymous slots")
	}
}
func TestDexShinyAndAlternateFirstLocks(t *testing.T) {
	for _, form := range []string{"standard", "mega-x"} {
		r := dexFixture(form, "shiny")
		d := NewDex(catalog.All(), r, dexAvailable)
		e, err := d.Entry(6, catalog.Selection{Form: "standard"})
		if err != nil || e.ArtworkKey != nil || e.SelectedName != "" || e.Notice == "" {
			t.Fatalf("fallback %+v %v", e, err)
		}
		selected, err := d.Entry(6, catalog.Selection{Form: form, Shiny: true})
		if err != nil || selected.ArtworkKey == nil || selected.Selected.Count != 1 {
			t.Fatalf("collected %+v %v", selected, err)
		}
		for _, n := range e.Evolution {
			if n.Number != 6 && n.Name != "?????" {
				t.Fatal("evolution identity leak")
			}
		}
	}
	d := NewDex(catalog.All(), dexFixture("standard", "shiny"), func(catalog.VariantKey) bool { return false })
	e, err := d.Entry(6, catalog.Selection{Shiny: true})
	if err != nil || e.ArtworkKey != nil || !strings.Contains(e.Notice, "unavailable") || e.Selected.Count != 1 {
		t.Fatal(e, err)
	}
}
func TestDexTypesMatchOneEncounteredForm(t *testing.T) {
	r := dexFixture("mega-x", "regular")
	r.Variants[0].Types = []string{"fire", "dragon"}
	r.Variants = append(r.Variants, VariantDiscovery{Key: catalog.VariantKey{SpeciesID: 6, FormID: "standard", Gender: "default", Palette: "regular"}, Types: []string{"fire", "flying"}})
	d := NewDex(catalog.All(), r, dexAvailable)
	for _, q := range []catalog.Selection{{Types: []string{"dragon", "flying"}}, {Types: []string{"dragon"}, TypesAny: []string{"flying"}}} {
		if len(d.List(DexFilter{Selection: q})) != 0 {
			t.Fatal("cross form match")
		}
	}
	if len(d.List(DexFilter{Selection: catalog.Selection{Types: []string{"fire"}, TypesAny: []string{"dragon"}}})) != 1 {
		t.Fatal("missed exact form")
	}
	if len(d.List(DexFilter{Selection: catalog.Selection{Colors: []string{"blue"}}})) != 0 {
		t.Fatal("unseen metadata selected")
	}
}
func TestDexSummaryIntersectionAndOwnership(t *testing.T) {
	r := dexFixture("mega-x", "shiny")
	d := NewDex(catalog.All(), r, dexAvailable)
	delete(r.Species, 6)
	r.Variants[0].Types[0] = "secret"
	s := d.Summary()
	if s.Completion.Species != 1 || s.Completion.SpeciesTotal != 1017 || s.Completion.Variants != 1 || s.Completion.VariantsTotal != 2669 || s.ShinyCollections != 1 || len(s.Generations) != 9 {
		t.Fatal(s)
	}
	e, _ := d.Entry(6, catalog.Selection{Form: "mega-x", Shiny: true})
	e.Forms[0].Types[0] = "mutated"
	again, _ := d.Entry(6, catalog.Selection{Form: "mega-x", Shiny: true})
	if again.Forms[0].Types[0] != "fire" {
		t.Fatal("owned result mutated state")
	}
}

func TestDexObservedGenderOnlyAndUnsupportedSelection(t *testing.T) {
	r := DexRecords{Species: map[int]Discovery{678: {Count: 1}}, Variants: []VariantDiscovery{{Key: catalog.VariantKey{SpeciesID: 678, FormID: "standard", Gender: "female", Palette: "shiny"}, Discovery: Discovery{Count: 1}, FormName: "Standard", Types: []string{"psychic"}}}}
	d := NewDex(catalog.All(), r, dexAvailable)
	e, err := d.Entry(678, catalog.Selection{Gender: "female", Shiny: true})
	if err != nil || e.ArtworkKey == nil || len(e.Forms[0].Genders) != 1 || e.Forms[0].Genders[0].Name != "female" || e.Forms[0].UnknownGenders != 1 {
		t.Fatal(e, err)
	}
	locked, err := d.Entry(678, catalog.Selection{Gender: "male"})
	if err != nil || locked.ArtworkKey != nil || locked.SelectedName != "" {
		t.Fatal(locked, err)
	}
	if _, err = d.Entry(678, catalog.Selection{Form: "mega-x"}); err == nil {
		t.Fatal("unsupported seen selection accepted")
	}
}
