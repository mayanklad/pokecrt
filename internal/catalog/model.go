package catalog

import (
	"slices"
	"sort"
	"strings"
)

// VariantKey identifies one exact appearance independently of an asset filename.
type VariantKey struct {
	SpeciesID int
	FormID    string
	Gender    string
	Palette   string
}

type Species struct {
	ID          int
	Name        string
	Slug        string
	Aliases     []string
	Generation  int
	Color       string
	Stage       int
	Baby        bool
	Legendary   bool
	Mythical    bool
	EvolvesFrom int
	EvolvesTo   []int
	Forms       []Form
}

type Form struct {
	ID            string
	Name          string
	Types         []string
	DefaultGender string
	Genders       []string
	// SourceAliases document upstream identities; they are not public selectors.
	SourceAliases []string
}

// All returns an independent copy in ascending National number order.
func All() []Species {
	result := make([]Species, len(generatedSpecies))
	for i, species := range generatedSpecies {
		result[i] = cloneSpecies(species)
	}
	return result
}

func ByName(name string) (Species, bool) {
	index, ok := generatedAliases[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		return Species{}, false
	}
	return cloneSpecies(generatedSpecies[index]), true
}

func ByNumber(number int) (Species, bool) {
	index := sort.Search(len(generatedSpecies), func(i int) bool {
		return generatedSpecies[i].ID >= number
	})
	if index == len(generatedSpecies) || generatedSpecies[index].ID != number {
		return Species{}, false
	}
	return cloneSpecies(generatedSpecies[index]), true
}

func cloneSpecies(species Species) Species {
	species.Aliases = slices.Clone(species.Aliases)
	species.EvolvesTo = slices.Clone(species.EvolvesTo)
	species.Forms = slices.Clone(species.Forms)
	for i := range species.Forms {
		species.Forms[i].Types = slices.Clone(species.Forms[i].Types)
		species.Forms[i].Genders = slices.Clone(species.Forms[i].Genders)
		species.Forms[i].SourceAliases = slices.Clone(species.Forms[i].SourceAliases)
	}
	return species
}
