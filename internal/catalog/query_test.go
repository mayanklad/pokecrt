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
