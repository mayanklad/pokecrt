package query

import (
	"errors"
	"testing"
)

func TestStandardCandidateBoundaries(t *testing.T) {
	for index, want := range []int{1, 6, 7} {
		species, key, err := ChooseStandard(func(n int) (int, error) {
			if n != 3 {
				t.Fatalf("candidate count=%d; want 3", n)
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
	_, _, err := ChooseStandard(func(int) (int, error) { return 0, failure })
	if !errors.Is(err, failure) {
		t.Fatalf("got %v", err)
	}
	for _, index := range []int{-1, 3} {
		_, _, err := ChooseStandard(func(int) (int, error) { return index, nil })
		if err == nil {
			t.Fatal("accepted out-of-range selector")
		}
	}
	if _, err := CryptoIndex(0); err == nil {
		t.Fatal("accepted empty range")
	}
	value, err := CryptoIndex(1)
	if err != nil || value != 0 {
		t.Fatalf("singleton selection: %d, %v", value, err)
	}
}
