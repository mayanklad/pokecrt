package sprite

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	"image/png"
	"slices"

	"github.com/mayanklad/pokecrt/internal/catalog"
)

//go:embed assets/*.png
var files embed.FS

type Asset struct {
	Key            catalog.VariantKey
	Path           string
	SHA256         string
	SourceSHA256   string
	SourceURL      string
	SourceProvider string
	Width          int
	Height         int
}

func Inventory() []Asset {
	return slices.Clone(generatedAssets)
}

// Derive exact-key positions from the generated manifest once. The index is
// read-only after initialization; positions avoid copying asset metadata into it.
// Inventory continues to expose manifest order independently of map iteration.
var assetIndex = func() map[catalog.VariantKey]int {
	index := make(map[catalog.VariantKey]int, len(generatedAssets))
	for i, asset := range generatedAssets {
		index[asset.Key] = i
	}
	return index
}()

func Lookup(key catalog.VariantKey) (Asset, bool) {
	index, ok := assetIndex[key]
	if !ok {
		return Asset{}, false
	}
	return generatedAssets[index], true
}

// Decode performs exact lookup; it never substitutes a different appearance.
func Decode(key catalog.VariantKey) (image.Image, error) {
	asset, ok := Lookup(key)
	if !ok {
		return nil, fmt.Errorf("artwork unavailable for species #%03d and selected variant", key.SpeciesID)
	}
	data, err := files.ReadFile(asset.Path)
	if err != nil {
		return nil, fmt.Errorf("read bundled artwork: %w", err)
	}
	decoded, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode bundled artwork: %w", err)
	}
	return decoded, nil
}
