// Package query selects catalog identities without trainer storage.
package query

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/mayanklad/pokecrt/internal/catalog"
	"github.com/mayanklad/pokecrt/internal/sprite"
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

// StandardKey uses the form's declared default artwork, never a fallback.
func StandardKey(species catalog.Species) (catalog.VariantKey, bool) {
	for _, form := range species.Forms {
		if form.ID == "standard" {
			return catalog.VariantKey{SpeciesID: species.ID, FormID: form.ID, Gender: form.DefaultGender, Palette: "regular"}, true
		}
	}
	return catalog.VariantKey{}, false
}

// ChooseStandard gives each eligible species exactly one candidate.
func ChooseStandard(selectIndex IndexSelector) (catalog.Species, catalog.VariantKey, error) {
	var candidates []catalog.Species
	for _, species := range catalog.All() {
		key, ok := StandardKey(species)
		if !ok {
			continue
		}
		if _, ok := sprite.Lookup(key); ok {
			candidates = append(candidates, species)
		}
	}
	if len(candidates) == 0 {
		return catalog.Species{}, catalog.VariantKey{}, fmt.Errorf("No Pokémon match the specified filters.")
	}
	index, err := selectIndex(len(candidates))
	if err != nil {
		return catalog.Species{}, catalog.VariantKey{}, err
	}
	if index < 0 || index >= len(candidates) {
		return catalog.Species{}, catalog.VariantKey{}, fmt.Errorf("selector returned an out-of-range index")
	}
	species := candidates[index]
	key, _ := StandardKey(species)
	return species, key, nil
}
