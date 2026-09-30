package catalog_test

import (
	"errors"
	"github.com/mayanklad/pokecrt/internal/catalog"
	"github.com/mayanklad/pokecrt/internal/sprite"
	"testing"
)

func TestStandardCandidateBoundaries(t *testing.T) {
	for _, test := range []struct{ index, want int }{{0, 1}, {24, 25}, {150, 151}, {151, 152}, {250, 251}, {251, 252}, {385, 386}, {424, 866}} {
		index, want := test.index, test.want
		species, key, err := chooseStandard(func(n int) (int, error) {
			if n != 425 {
				t.Fatalf("candidate count=%d; want 425", n)
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
	for _, index := range []int{-1, 425} {
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
