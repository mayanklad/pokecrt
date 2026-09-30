package sprite

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/mayanklad/pokecrt/internal/catalog"
)

func TestEmbeddedInventoryMatchesManifest(t *testing.T) {
	if DatasetID != catalog.DatasetID || len(Inventory()) == 0 {
		t.Fatal("missing or inconsistent embedded dataset")
	}
	for _, asset := range Inventory() {
		data, err := files.ReadFile(asset.Path)
		if err != nil {
			t.Fatal(err)
		}
		hash := sha256.Sum256(data)
		if hex.EncodeToString(hash[:]) != asset.SHA256 {
			t.Fatalf("content hash mismatch: %s", asset.Path)
		}
		decoded, err := Decode(asset.Key)
		if err != nil || decoded.Bounds().Dx() != asset.Width || decoded.Bounds().Dy() != asset.Height {
			t.Fatalf("decode/dimension mismatch: %s: %v", asset.Path, err)
		}
		if _, ok := catalog.ByNumber(asset.Key.SpeciesID); !ok {
			t.Fatalf("asset has unknown species: %+v", asset.Key)
		}
	}
}

func TestAbsentExactVariantHasNoFallback(t *testing.T) {
	key := catalog.VariantKey{SpeciesID: 6, FormID: "unsupported-test-form", Gender: "default", Palette: "regular"}
	if _, ok := Lookup(key); ok {
		t.Fatal("absent exact variant resolved")
	}
	if _, err := Decode(key); err == nil {
		t.Fatal("absent variant received fallback artwork")
	}
}
