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

func Lookup(key catalog.VariantKey) (Asset, bool) {
	for _, asset := range generatedAssets {
		if asset.Key == key {
			return asset, true
		}
	}
	return Asset{}, false
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
