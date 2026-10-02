package catalog_test

import (
	"errors"
	"github.com/mayanklad/pokecrt/internal/catalog"
	"github.com/mayanklad/pokecrt/internal/sprite"
	"testing"
)

func TestStandardCandidateBoundaries(t *testing.T) {
	for _, test := range []struct{ index, want int }{{0, 1}, {24, 25}, {150, 151}, {151, 152}, {250, 251}, {251, 252}, {385, 386}, {386, 387}, {492, 493}, {493, 494}, {648, 649}, {649, 650}, {720, 721}, {721, 722}, {808, 809}, {809, 810}, {897, 898}, {898, 906}, {1012, 1025}} {
		index, want := test.index, test.want
		species, key, err := chooseStandard(func(n int) (int, error) {
			if n != 1013 {
				t.Fatalf("candidate count=%d; want 1013", n)
			}
			return index, nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if species.ID != want || key.SpeciesID != want || key.FormID != "standard" || key.Palette != "regular" {
			t.Fatalf("unexpected selection: %+v %+v", species, key)
		}
	}
}

func TestSelectorFailures(t *testing.T) {
	failure := errors.New("entropy unavailable")
	_, _, err := chooseStandard(func(int) (int, error) { return 0, failure })
	if !errors.Is(err, failure) {
		t.Fatalf("got %v", err)
	}
	for _, index := range []int{-1, 1013} {
		_, _, err := chooseStandard(func(int) (int, error) { return index, nil })
		if err == nil {
			t.Fatal("accepted out-of-range selector")
		}
	}
	if _, err := catalog.CryptoIndex(0); err == nil {
		t.Fatal("accepted empty range")
	}
	value, err := catalog.CryptoIndex(1)
	if err != nil || value != 0 {
		t.Fatalf("singleton selection: %d, %v", value, err)
	}
}

func chooseStandard(selectIndex catalog.IndexSelector) (catalog.Species, catalog.VariantKey, error) {
	return catalog.ChooseStandard(selectIndex, func(k catalog.VariantKey) bool { _, ok := sprite.Lookup(k); return ok })
}

func TestMetadataOnlyTransformationHasNoSpriteFallback(t *testing.T) {
	for _, form := range []string{"curly-mega", "droopy-mega", "stretchy-mega"} {
		for _, palette := range []string{"regular", "shiny"} {
			key := catalog.VariantKey{SpeciesID: 978, FormID: form, Gender: "default", Palette: palette}
			if _, ok := sprite.Lookup(key); ok {
				t.Fatalf("unsupported transformation has artwork: %+v", key)
			}
		}
	}
	if _, ok := sprite.Lookup(catalog.VariantKey{SpeciesID: 978, FormID: "droopy", Gender: "default", Palette: "regular"}); !ok {
		t.Fatal("lost ordinary artwork")
	}
}

func TestUnavailableOinkologneGendersHaveNoSprite(t *testing.T) {
	for _, gender := range []string{"male", "female"} {
		for _, palette := range []string{"regular", "shiny"} {
			key := catalog.VariantKey{SpeciesID: 916, FormID: "standard", Gender: gender, Palette: palette}
			if _, ok := sprite.Lookup(key); ok {
				t.Fatalf("generated Oinkologne gender candidate accepted: %+v", key)
			}
		}
	}
}

func query(t *testing.T, q catalog.Selection) []catalog.Match {
	t.Helper()
	matches, err := catalog.Query(q, func(k catalog.VariantKey) bool { _, ok := sprite.Lookup(k); return ok })
	if err != nil {
		t.Fatal(err)
	}
	return matches
}

func TestComposableSelectionUsesOneExactAppearance(t *testing.T) {
	for _, tc := range []struct {
		q     catalog.Selection
		count int
	}{
		{catalog.Selection{Name: "charizard", Types: []string{"fire"}, TypesAny: []string{"flying", "dragon"}}, 1},
		{catalog.Selection{Name: "charizard", Types: []string{"fire", "dragon"}}, 0},
		{catalog.Selection{Name: "charizard", Form: "mega-x", Types: []string{"fire", "dragon"}, Generations: []int{1, 2}, Stages: []int{3}, Colors: []string{"red"}}, 1},
		{catalog.Selection{Name: "charizard", Generations: []int{2}}, 0},
		{catalog.Selection{Name: "charizard", Form: "hisui"}, 0},
		{catalog.Selection{Name: "charizard", Gender: "female"}, 0},
		{catalog.Selection{Name: "mew", Mythical: true}, 1},
		{catalog.Selection{Name: "mew", Legendary: true, Mythical: true}, 0},
		{catalog.Selection{Name: "pichu", Baby: true, Stages: []int{1}}, 1},
		{catalog.Selection{Name: "mew", Legendary: false}, 1},
	} {
		got := query(t, tc.q)
		if len(got) != tc.count {
			t.Fatalf("%+v: got %d, want %d", tc.q, len(got), tc.count)
		}
		for _, m := range got {
			if m.Key.FormID != m.Form.ID || m.Key.SpeciesID != m.Species.ID {
				t.Fatal("mixed appearance identity")
			}
		}
	}
}

func TestSelectionVocabularyNormalizationAndUnknownValues(t *testing.T) {
	q, err := catalog.ValidateSelection(catalog.Selection{Name: " CHARIZARD ", Form: " MEGA-X ", Types: []string{"FIRE", " fire ", "Dragon"}, Generations: []int{2, 1, 1}})
	if err != nil || q.Form != "mega-x" || len(q.Types) != 2 || len(q.Generations) != 2 {
		t.Fatalf("normalized=%+v err=%v", q, err)
	}
	for _, q := range []catalog.Selection{
		{Name: "char"}, {Form: "base"}, {Gender: "default"}, {Generations: []int{10}}, {Stages: []int{99}}, {Types: []string{"stellar"}}, {Colors: []string{"silver"}}, {TypesAny: []string{""}},
	} {
		if _, err := catalog.ValidateSelection(q); err == nil {
			t.Fatalf("accepted %+v", q)
		}
	}
}

func TestQueryAvailabilityAndUniformSpeciesCandidates(t *testing.T) {
	got := query(t, catalog.Selection{Name: "oinkologne", Gender: "female"})
	if len(got) != 1 || got[0].Available || got[0].Key.Gender != "female" {
		t.Fatalf("metadata-only gender=%+v", got)
	}
	if len(query(t, catalog.Selection{Name: "oinkologne", Gender: "female", Shiny: true})) != 0 {
		t.Fatal("shiny query retained missing artwork")
	}
	got = query(t, catalog.Selection{Name: "meowstic", Gender: "female", Shiny: true})
	if len(got) != 1 || !got[0].Available || got[0].Key.Palette != "shiny" {
		t.Fatalf("female shiny=%+v", got)
	}
	matches := query(t, catalog.Selection{Types: []string{"fire"}})
	available := 0
	seen := map[int]bool{}
	for _, m := range matches {
		if seen[m.Species.ID] {
			t.Fatal("species multiplied by appearances")
		}
		seen[m.Species.ID] = true
		if m.Available {
			available++
		}
	}
	calls := 0
	for i := 0; i < available; i++ {
		index := i
		m, err := catalog.Choose(matches, func(n int) (int, error) {
			calls++
			if n != available {
				t.Fatalf("n=%d want %d", n, available)
			}
			return index, nil
		})
		if err != nil || !m.Available || m.Key.FormID != "standard" {
			t.Fatalf("selection=%+v err=%v", m, err)
		}
	}
	if calls != available {
		t.Fatal("uniform candidate selector bypassed")
	}
	matches = query(t, catalog.Selection{Name: "charizard", Form: "mega-x"})
	matches[0].Form.Types[0] = "changed"
	if query(t, catalog.Selection{Name: "charizard", Form: "mega-x"})[0].Form.Types[0] == "changed" {
		t.Fatal("query mutation changed bundled catalog")
	}
}

func BenchmarkQuery(b *testing.B) {
	for _, tc := range []struct {
		name      string
		selection catalog.Selection
	}{
		{"Standard", catalog.Selection{}},
		{"Named", catalog.Selection{Name: "charizard"}},
		{"Filtered", catalog.Selection{Generations: []int{1, 2}, Types: []string{"fire"}, TypesAny: []string{"flying", "dragon"}}},
	} {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				matches, err := catalog.Query(tc.selection, func(key catalog.VariantKey) bool { _, ok := sprite.Lookup(key); return ok })
				if err != nil || len(matches) == 0 {
					b.Fatalf("query: %v; %d matches", err, len(matches))
				}
			}
		})
	}
}
