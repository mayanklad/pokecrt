package catalog

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"slices"
	"sort"
	"strings"
)

// IndexSelector returns an unbiased index in [0, n).
type IndexSelector func(n int) (int, error)

func CryptoIndex(n int) (int, error) {
	if n <= 0 {
		return 0, fmt.Errorf("selection requires a positive candidate count")
	}
	value, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		return 0, fmt.Errorf("random selection: %w", err)
	}
	return int(value.Int64()), nil
}

// Selection evaluates all metadata against one exact selected form.
// False status flags impose no restriction; omitted gender uses the form default.
type Selection struct {
	Name                             string
	Generations, Stages              []int
	Types, TypesAny, Colors          []string
	Form, Gender                     string
	Shiny, Legendary, Mythical, Baby bool
}

type Match struct {
	Species   Species
	Form      Form
	Key       VariantKey
	Available bool
}

// ValidateSelection uses bundled metadata as the static selector vocabulary.
func ValidateSelection(q Selection) (Selection, error) {
	q.Name = strings.TrimSpace(q.Name)
	if q.Name != "" {
		if _, ok := ByName(q.Name); !ok {
			return q, fmt.Errorf("unknown species %q", q.Name)
		}
	}
	q.Form = strings.ToLower(strings.TrimSpace(q.Form))
	if q.Form == "" {
		q.Form = "standard"
	}
	q.Gender = strings.ToLower(strings.TrimSpace(q.Gender))
	if q.Gender != "" && q.Gender != "male" && q.Gender != "female" {
		return q, fmt.Errorf("unsupported gender %q", q.Gender)
	}
	generations, stages := map[int]bool{}, map[int]bool{}
	forms, types, colors := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, s := range generatedSpecies {
		generations[s.Generation] = true
		stages[s.Stage] = true
		colors[s.Color] = true
		for _, f := range s.Forms {
			forms[f.ID] = true
			for _, typ := range f.Types {
				types[typ] = true
			}
		}
	}
	if !forms[q.Form] {
		return q, fmt.Errorf("unknown form %q", q.Form)
	}
	for _, group := range []struct {
		name      string
		values    *[]int
		supported map[int]bool
	}{{"gen", &q.Generations, generations}, {"stage", &q.Stages, stages}} {
		values := slices.Clone(*group.values)
		sort.Ints(values)
		values = slices.Compact(values)
		for _, v := range values {
			if v <= 0 || !group.supported[v] {
				return q, fmt.Errorf("unsupported --%s value %d", group.name, v)
			}
		}
		*group.values = values
	}
	for _, group := range []struct {
		name      string
		values    *[]string
		supported map[string]bool
	}{{"type", &q.Types, types}, {"type-any", &q.TypesAny, types}, {"color", &q.Colors, colors}} {
		values := slices.Clone(*group.values)
		for i, v := range values {
			values[i] = strings.ToLower(strings.TrimSpace(v))
			if !group.supported[values[i]] {
				return q, fmt.Errorf("unsupported --%s value %q", group.name, v)
			}
		}
		sort.Strings(values)
		*group.values = slices.Compact(values)
	}
	return q, nil
}

// Query retains unavailable regular metadata entries for public listing.
// Shiny selection requires actual shiny artwork. Each species appears once.
func Query(q Selection, available func(VariantKey) bool) ([]Match, error) {
	q, err := ValidateSelection(q)
	if err != nil {
		return nil, err
	}
	namedID := 0
	if q.Name != "" {
		s, _ := ByName(q.Name)
		namedID = s.ID
	}
	result := []Match{}
	for _, s := range generatedSpecies {
		if namedID != 0 && s.ID != namedID {
			continue
		}
		if len(q.Generations) > 0 && !slices.Contains(q.Generations, s.Generation) {
			continue
		}
		if len(q.Stages) > 0 && !slices.Contains(q.Stages, s.Stage) {
			continue
		}
		if len(q.Colors) > 0 && !slices.Contains(q.Colors, s.Color) {
			continue
		}
		if q.Legendary && !s.Legendary || q.Mythical && !s.Mythical || q.Baby && !s.Baby {
			continue
		}
		for _, f := range s.Forms {
			if f.ID != q.Form {
				continue
			}
			containsAll := true
			for _, typ := range q.Types {
				if !slices.Contains(f.Types, typ) {
					containsAll = false
				}
			}
			if !containsAll {
				continue
			}
			if len(q.TypesAny) > 0 {
				containsAny := false
				for _, typ := range q.TypesAny {
					if slices.Contains(f.Types, typ) {
						containsAny = true
					}
				}
				if !containsAny {
					continue
				}
			}
			gender := f.DefaultGender
			if q.Gender != "" {
				if len(f.Genders) < 2 || !slices.Contains(f.Genders, q.Gender) {
					continue
				}
				gender = q.Gender
			}
			palette := "regular"
			if q.Shiny {
				palette = "shiny"
			}
			key := VariantKey{SpeciesID: s.ID, FormID: f.ID, Gender: gender, Palette: palette}
			exists := available != nil && available(key)
			if q.Shiny && !exists {
				continue
			}
			copied := cloneSpecies(s)
			var form Form
			for _, candidate := range copied.Forms {
				if candidate.ID == f.ID {
					form = candidate
					break
				}
			}
			result = append(result, Match{Species: copied, Form: form, Key: key, Available: exists})
			break
		}
	}
	return result, nil
}

// Choose gives each renderable matching species exactly one candidate.
func Choose(matches []Match, selectIndex IndexSelector) (Match, error) {
	candidates := []Match{}
	for _, m := range matches {
		if m.Available {
			candidates = append(candidates, m)
		}
	}
	if len(candidates) == 0 {
		return Match{}, fmt.Errorf("No Pokémon match the specified filters.")
	}
	if len(candidates) == 1 {
		return candidates[0], nil
	}
	index, err := selectIndex(len(candidates))
	if err != nil {
		return Match{}, err
	}
	if index < 0 || index >= len(candidates) {
		return Match{}, fmt.Errorf("selector returned an out-of-range index")
	}
	return candidates[index], nil
}

// StandardKey uses the form's declared default artwork, never a fallback.
func StandardKey(species Species) (VariantKey, bool) {
	for _, form := range species.Forms {
		if form.ID == "standard" {
			return VariantKey{SpeciesID: species.ID, FormID: form.ID, Gender: form.DefaultGender, Palette: "regular"}, true
		}
	}
	return VariantKey{}, false
}

func ChooseStandard(selectIndex IndexSelector, available func(VariantKey) bool) (Species, VariantKey, error) {
	matches, err := Query(Selection{}, available)
	if err != nil {
		return Species{}, VariantKey{}, err
	}
	chosen, err := Choose(matches, selectIndex)
	return chosen.Species, chosen.Key, err
}
