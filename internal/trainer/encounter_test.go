package trainer

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/mayanklad/pokecrt/internal/catalog"
	"github.com/mayanklad/pokecrt/internal/sprite"
)

func fixturePool(t *testing.T) (*Pool, []catalog.Species) {
	t.Helper()
	species := []catalog.Species{
		{ID: 1, Name: "One", Forms: []catalog.Form{{ID: "standard", Name: "Standard", Types: []string{"grass"}, Genders: []string{"default"}}}},
		{ID: 2, Name: "Many", Forms: []catalog.Form{
			{ID: "standard", Name: "Standard", Types: []string{"fire"}, Genders: []string{"default"}},
			{ID: "mega", Name: "Mega", Types: []string{"fire", "dragon"}, Tags: []string{"mega"}, Genders: []string{"default"}},
			{ID: "regional", Name: "Regional", Types: []string{"ice"}, Tags: []string{"regional"}, Genders: []string{"male", "female"}},
		}},
	}
	p, err := NewPool(species, func(key catalog.VariantKey) bool {
		// One standard has both palettes; a shiny-only female is not eligible.
		if key.SpeciesID == 1 {
			return true
		}
		if key.Gender == "female" {
			return key.Palette == "shiny"
		}
		return key.Palette == "regular"
	})
	if err != nil {
		t.Fatal(err)
	}
	return p, species
}

func controlled(values []int, bounds *[]int) func(int) (int, error) {
	i := 0
	return func(n int) (int, error) {
		*bounds = append(*bounds, n)
		if i >= len(values) {
			return 0, errors.New("unexpected random call")
		}
		v := values[i]
		i++
		return v, nil
	}
}

func TestHierarchicalSelectionAndShinyBoundaries(t *testing.T) {
	p, _ := fixturePool(t)
	if p.SpeciesCount() != 2 || p.FormCount() != 4 || p.VariantCount() != 5 {
		t.Fatal("pool totals")
	}
	for _, roll := range []int{0, 1, 4095} {
		bounds := []int{}
		choice, err := p.Select(controlled([]int{0, 0, 0, roll}, &bounds))
		palette := "regular"
		if roll == 0 {
			palette = "shiny"
		}
		if err != nil || choice.Key().SpeciesID != 1 || choice.Key().Palette != palette || !reflect.DeepEqual(bounds, []int{2, 1, 1, 4096}) {
			t.Fatalf("shiny %d: %+v %v %v", roll, choice, err, bounds)
		}
	}
	for form := 0; form < 3; form++ {
		bounds := []int{}
		c, err := p.Select(controlled([]int{1, form, 0}, &bounds))
		if err != nil || c.Key().SpeciesID != 2 || c.Key().Palette != "regular" || c.Key().Gender == "female" || !reflect.DeepEqual(bounds, []int{2, 3, 1}) {
			t.Fatalf("form draw: %+v %v %v", c, err, bounds)
		}
		if c.Key().FormID == "mega" && !c.Snapshot().Transformation {
			t.Fatal("transformation snapshot")
		}
		if c.Key().FormID == "regional" && !c.Snapshot().Regional {
			t.Fatal("regional snapshot")
		}
	}
	for _, values := range [][]int{{-1}, {2}, {0, 1}, {0, 0, -1}, {0, 0, 0, 4096}} {
		bounds := []int{}
		if _, err := p.Select(controlled(values, &bounds)); err == nil {
			t.Fatalf("invalid RNG accepted: %v", values)
		}
	}
	if _, err := p.Select(func(int) (int, error) { return 0, errors.New("RNG failure") }); err == nil {
		t.Fatal("RNG error ignored")
	}
}

func TestPoolOwnsMetadataAndExactEligibleInventory(t *testing.T) {
	p, species := fixturePool(t)
	species[0].Name = "mutated"
	species[0].Forms[0].Name = "mutated"
	species[0].Forms[0].Types[0] = "water"
	bounds := []int{}
	c, err := p.Select(controlled([]int{0, 0, 0, 1}, &bounds))
	if err != nil || c.Snapshot().SpeciesName != "One" || c.Snapshot().Type1 != "grass" {
		t.Fatal("metadata ownership")
	}
	p, err = BundledPool()
	if err != nil {
		t.Fatal(err)
	}
	if p.SpeciesCount() != 1017 || p.FormCount() != 1327 || p.VariantCount() != 2669 {
		t.Fatalf("bundled totals: %d %d %d", p.SpeciesCount(), p.FormCount(), p.VariantCount())
	}
	seen := map[catalog.VariantKey]bool{}
	for _, s := range p.species {
		for _, f := range s.forms {
			for _, g := range f.genders {
				key := g.regular.Key()
				if _, ok := sprite.Lookup(key); !ok {
					t.Fatal("missing regular eligible")
				}
				if seen[key] {
					t.Fatal("duplicate eligible key")
				}
				seen[key] = true
				if g.shiny {
					key.Palette = "shiny"
					if _, ok := sprite.Lookup(key); !ok {
						t.Fatal("missing shiny eligible")
					}
					seen[key] = true
				}
			}
		}
	}
	for _, asset := range sprite.Inventory() {
		if !seen[asset.Key] {
			t.Fatalf("accepted asset excluded: %+v", asset.Key)
		}
	}
}

type encounterFake struct {
	active   Profile
	captured int64
	records  int
	fail     error
}

func (r *encounterFake) ActiveProfile(context.Context) (Profile, error) {
	if r.active.ID == 0 {
		return Profile{}, ErrNoActive
	}
	return r.active, nil
}
func (r *encounterFake) RecordEncounter(_ context.Context, id int64, c Choice, when int64) (Record, error) {
	r.records++
	r.captured = id
	if r.fail != nil {
		return Record{}, r.fail
	}
	return Record{ID: 1, TrainerID: id, Choice: c, EncounteredAtMS: when}, nil
}

func TestServicePreparationCaptureAndFailureBoundaries(t *testing.T) {
	p, _ := fixturePool(t)
	repo := &encounterFake{active: Profile{ID: 1}}
	now := time.Date(2026, 10, 3, 1, 2, 3, 456000000, time.FixedZone("offset", 19800))
	artwork := []byte("prepared sprite")
	service := EncounterService{Repository: repo, Pool: p, Index: func(int) (int, error) { return 0, nil }, Clock: func() time.Time { return now }, Prepare: func(Choice) ([]byte, error) { repo.active.ID = 2; return artwork, nil }}
	result, err := service.Encounter(context.Background())
	if err != nil || repo.captured != 1 || result.Record.EncounteredAtMS != now.UnixMilli() {
		t.Fatalf("capture/time: %+v %v", result, err)
	}
	artwork[0] = 'X'
	out := result.Artwork()
	out[0] = 'Y'
	if string(result.Artwork()) != "prepared sprite" {
		t.Fatal("prepared buffer was shared")
	}
	repo.records = 0
	service.Prepare = func(Choice) ([]byte, error) { return nil, errors.New("decode/render failed") }
	if _, err := service.Encounter(context.Background()); err == nil || repo.records != 0 {
		t.Fatal("preparation failure recorded")
	}
	service.Prepare = func(Choice) ([]byte, error) { return nil, nil }
	if _, err := service.Encounter(context.Background()); err == nil || repo.records != 0 {
		t.Fatal("empty preparation recorded")
	}
	cancelDuringPrepare, cancelPrepare := context.WithCancel(context.Background())
	service.Prepare = func(Choice) ([]byte, error) { cancelPrepare(); return []byte("art"), nil }
	if _, err := service.Encounter(cancelDuringPrepare); !errors.Is(err, context.Canceled) || repo.records != 0 {
		t.Fatal("cancellation after preparation recorded")
	}
	repo.active.ID = 0
	service.Prepare = func(Choice) ([]byte, error) { t.Fatal("prepared without active trainer"); return nil, nil }
	if _, err := service.Encounter(context.Background()); !errors.Is(err, ErrNoActive) {
		t.Fatalf("missing active: %v", err)
	}
	repo.active.ID = 1
	service.Prepare = func(Choice) ([]byte, error) { return []byte("art"), nil }
	repo.fail = errors.New("storage failure")
	result, err = service.Encounter(context.Background())
	if err == nil || result.Record.ID != 0 || len(result.Artwork()) != 0 {
		t.Fatal("failed recording returned success")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	repo.records = 0
	if _, err := service.Encounter(ctx); !errors.Is(err, context.Canceled) || repo.records != 0 {
		t.Fatal("canceled action recorded")
	}
}
