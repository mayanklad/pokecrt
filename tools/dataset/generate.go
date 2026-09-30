package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/format"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"sort"
)

type normalizedAsset struct {
	SpeciesID      int
	FormID         string
	Gender         string
	Palette        string
	Path           string
	SHA256         string
	SourceSHA256   string
	SourceURL      string
	SourceProvider string
	SourceWidth    int
	SourceHeight   int
	CropX          int
	CropY          int
	Width          int
	Height         int
}

type missingAsset struct {
	SpeciesID int    `json:"species_id"`
	Name      string `json:"name"`
	Reason    string `json:"reason"`
}

type missingVariant struct {
	SpeciesID int    `json:"species_id"`
	Name      string `json:"name"`
	FormID    string `json:"form_id"`
	Gender    string `json:"gender"`
	Palette   string `json:"palette"`
	Reason    string `json:"reason"`
}

type coverageReport struct {
	DatasetID                 string           `json:"dataset_id"`
	RulesVersion              string           `json:"rules_version"`
	CatalogSpecies            int              `json:"catalog_species"`
	CatalogForms              int              `json:"catalog_forms"`
	MissingVariants           []missingVariant `json:"missing_regular_variants"`
	EligibleSpecies           int              `json:"eligible_species"`
	StandardRegularSprites    int              `json:"standard_regular_sprites"`
	ShinySprites              int              `json:"shiny_sprites"`
	CollectibleForms          int              `json:"collectible_forms"`
	DistinctVisualGenderSlots int              `json:"distinct_visual_gender_slots"`
	ExactEligibleVariants     int              `json:"exact_eligible_variants"`
	MaxSpriteWidth            int              `json:"max_sprite_width"`
	MaxSpriteHeight           int              `json:"max_sprite_height"`
	Missing                   []missingAsset   `json:"missing_standard_regular_assets"`
	Exclusions                []string         `json:"exclusions"`
	SourceQualityFlags        []string         `json:"source_quality_flags"`
}

type generatedBundle struct {
	DatasetID string
	Coverage  coverageReport
	Files     map[string][]byte
}

func digest(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// normalizePNG crops only transparent margins and preserves visible pixels.
// Partial alpha and non-8-bit visible colors require an explicit future rule.
func normalizePNG(data []byte) ([]byte, image.Rectangle, image.Point, error) {
	config, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width <= 0 || config.Height <= 0 || config.Width > 512 || config.Height > 512 {
		return nil, image.Rectangle{}, image.Point{}, fmt.Errorf("invalid PNG or unsupported source dimensions")
	}
	decoded, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, image.Rectangle{}, image.Point{}, err
	}
	bounds := decoded.Bounds()
	visible := image.Rectangle{Min: bounds.Max, Max: bounds.Min}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := decoded.At(x, y).RGBA()
			if a == 0 {
				continue
			}
			if a != 65535 || r%257 != 0 || g%257 != 0 || b%257 != 0 {
				return nil, image.Rectangle{}, image.Point{}, fmt.Errorf("partial alpha or non-8-bit pixels require an explicit normalization rule")
			}
			visible.Min.X = min(visible.Min.X, x)
			visible.Min.Y = min(visible.Min.Y, y)
			visible.Max.X = max(visible.Max.X, x+1)
			visible.Max.Y = max(visible.Max.Y, y+1)
		}
	}
	if visible.Empty() {
		return nil, image.Rectangle{}, image.Point{}, fmt.Errorf("fully transparent asset")
	}
	cropped := image.NewNRGBA(image.Rect(0, 0, visible.Dx(), visible.Dy()))
	for y := visible.Min.Y; y < visible.Max.Y; y++ {
		for x := visible.Min.X; x < visible.Max.X; x++ {
			r, g, b, a := decoded.At(x, y).RGBA()
			if a != 0 {
				cropped.SetNRGBA(x-visible.Min.X, y-visible.Min.Y, color.NRGBA{R: uint8(r / 257), G: uint8(g / 257), B: uint8(b / 257), A: 255})
			}
		}
	}
	var output bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.BestCompression}
	if err := encoder.Encode(&output, cropped); err != nil {
		return nil, image.Rectangle{}, image.Point{}, err
	}
	return output.Bytes(), visible, bounds.Size(), nil
}

func buildBundle(lock sourceLock, cache string, mappings mappingConfig) (generatedBundle, error) {
	bundle := generatedBundle{Files: make(map[string][]byte)}
	if err := validateMappings(&mappings); err != nil {
		return bundle, err
	}
	species, sourceSpecies, err := normalizeCatalog(lock, cache, mappings)
	if err != nil {
		return bundle, err
	}
	indexData, _, _, err := readInput(lock, cache, "pokesprite-v2", "sources/generated/asset-index.json")
	if err != nil {
		return bundle, err
	}
	var index map[string]json.RawMessage
	if err := json.Unmarshal(indexData, &index); err != nil {
		return bundle, fmt.Errorf("decode asset provenance: %w", err)
	}
	var assets []normalizedAsset
	seenHashes := make(map[string]bool)
	for _, mapping := range mappings.Assets {
		var slots map[string]struct {
			Source      string `json:"source"`
			IsGenerated *bool  `json:"is_generated"`
		}
		if err := json.Unmarshal(index[mapping.SourceSlug], &slots); err != nil {
			return bundle, fmt.Errorf("missing or invalid provenance for %s: %w", mapping.SourceSlug, err)
		}
		form, ok := findNormalizedForm(species, mapping.SpeciesID, mapping.FormID)
		if !ok || !slices.Contains(form.Genders, mapping.Gender) {
			return bundle, fmt.Errorf("asset maps an undeclared appearance %s", assetIdentity(mapping))
		}
		selected, err := sourceFormByID(sourceSpecies[mapping.SpeciesID], mapping.SourceFormID)
		if err != nil {
			return bundle, err
		}
		available := selected.HasRegular
		if mapping.Palette == "shiny" {
			available = selected.HasShiny
		}
		provenance, ok := slots[mapping.Palette]
		if selected.Slug != mapping.SourceSlug || selected.CanonicalForm != nil || available == nil || !*available || selected.IsGenerated == nil || *selected.IsGenerated || !ok || provenance.IsGenerated == nil || *provenance.IsGenerated || provenance.Source != "msikma/pokesprite" || selected.Source != provenance.Source {
			return bundle, fmt.Errorf("unverified, generated, aliased, or mismatched asset %s", assetIdentity(mapping))
		}
		if mapping.Gender == "default" && selected.ID != form.SourceFormID {
			return bundle, fmt.Errorf("default asset uses a different source form %s", assetIdentity(mapping))
		}
		data, source, file, err := readInput(lock, cache, mapping.SourceID, mapping.Path)
		if err != nil {
			return bundle, err
		}
		cropped, bounds, originalSize, err := normalizePNG(data)
		if err != nil {
			return bundle, fmt.Errorf("normalize #%03d: %w", mapping.SpeciesID, err)
		}
		hash := digest(cropped)
		if mapping.Palette == "regular" && seenHashes[hash] {
			return bundle, fmt.Errorf("duplicate normalized asset; review collectible identity mappings")
		}
		if mapping.Palette == "regular" {
			seenHashes[hash] = true
		}
		assetPath := fmt.Sprintf("assets/%04d-%s-%s-%s.png", mapping.SpeciesID, mapping.FormID, mapping.Gender, mapping.Palette)
		assets = append(assets, normalizedAsset{SpeciesID: mapping.SpeciesID, FormID: mapping.FormID, Gender: mapping.Gender, Palette: mapping.Palette, Path: assetPath, SHA256: hash, SourceSHA256: file.SHA256, SourceURL: "https://raw.githubusercontent.com/" + source.Repository + "/" + source.Revision + "/" + file.Path, SourceProvider: provenance.Source, SourceWidth: originalSize.X, SourceHeight: originalSize.Y, CropX: bounds.Min.X, CropY: bounds.Min.Y, Width: bounds.Dx(), Height: bounds.Dy()})
		bundle.Files["internal/sprite/"+assetPath] = cropped
	}
	if err := validateVariantInventory(species, assets); err != nil {
		return bundle, err
	}
	identity, err := json.Marshal(struct {
		Mappings mappingConfig
		Species  []normalizedSpecies
		Assets   []normalizedAsset
	}{mappings, species, assets})
	if err != nil {
		return bundle, err
	}
	bundle.DatasetID = digest(identity)
	catalogGo, err := catalogSource(species, bundle.DatasetID)
	if err != nil {
		return bundle, err
	}
	bundle.Files["internal/catalog/generated.go"] = catalogGo
	spriteGo, err := spriteSource(assets, bundle.DatasetID)
	if err != nil {
		return bundle, err
	}
	bundle.Files["internal/sprite/manifest_generated.go"] = spriteGo
	bundle.Coverage = makeCoverage(species, assets, mappings, bundle.DatasetID)
	coverageJSON, err := json.MarshalIndent(bundle.Coverage, "", "  ")
	if err != nil {
		return bundle, err
	}
	bundle.Files["tools/dataset/coverage.json"] = append(coverageJSON, '\n')
	bundle.Files["tools/dataset/coverage.md"] = coverageMarkdown(bundle.Coverage)
	notices, err := thirdPartyNotices(lock, cache)
	if err != nil {
		return bundle, err
	}
	bundle.Files["THIRD_PARTY_NOTICES.md"] = notices
	return bundle, nil
}

func catalogSource(species []normalizedSpecies, datasetID string) ([]byte, error) {
	var output bytes.Buffer
	fmt.Fprintln(&output, "// Code generated by tools/dataset; DO NOT EDIT.\npackage catalog")
	fmt.Fprintf(&output, "\nconst DatasetID = %q\n\nvar generatedSpecies = []Species{\n", datasetID)
	for _, s := range species {
		fmt.Fprintf(&output, "{ID:%d,Name:%q,Slug:%q,Aliases:%#v,Generation:%d,Color:%q,Stage:%d,Baby:%t,Legendary:%t,Mythical:%t,EvolvesFrom:%d,EvolvesTo:%#v,Forms:[]Form{", s.ID, s.Name, s.Slug, s.Aliases, s.Generation, s.Color, s.Stage, s.Baby, s.Legendary, s.Mythical, s.EvolvesFrom, s.EvolvesTo)
		for _, f := range s.Forms {
			fmt.Fprintf(&output, "{ID:%q,Name:%q,Types:%#v,DefaultGender:%q,Genders:%#v,SourceAliases:%#v},", f.ID, f.Name, f.Types, f.DefaultGender, f.Genders, f.SourceAliases)
		}
		fmt.Fprintln(&output, "}},")
	}
	fmt.Fprintln(&output, "}\n\nvar generatedAliases = map[string]int{")
	for i, s := range species {
		for _, alias := range s.Aliases {
			fmt.Fprintf(&output, "%q:%d,\n", alias, i)
		}
	}
	fmt.Fprintln(&output, "}")
	return format.Source(output.Bytes())
}

func spriteSource(assets []normalizedAsset, datasetID string) ([]byte, error) {
	var output bytes.Buffer
	fmt.Fprintln(&output, "// Code generated by tools/dataset; DO NOT EDIT.\npackage sprite\n\nimport \"github.com/mayanklad/pokecrt/internal/catalog\"")
	fmt.Fprintf(&output, "\nconst DatasetID = %q\n\nvar generatedAssets = []Asset{\n", datasetID)
	for _, a := range assets {
		fmt.Fprintf(&output, "{Key:catalog.VariantKey{SpeciesID:%d, FormID:%q, Gender:%q, Palette:%q}, Path:%q, SHA256:%q, SourceSHA256:%q, SourceURL:%q, SourceProvider:%q, Width:%d, Height:%d},\n", a.SpeciesID, a.FormID, a.Gender, a.Palette, a.Path, a.SHA256, a.SourceSHA256, a.SourceURL, a.SourceProvider, a.Width, a.Height)
	}
	fmt.Fprintln(&output, "}")
	return format.Source(output.Bytes())
}

func makeCoverage(species []normalizedSpecies, assets []normalizedAsset, mappings mappingConfig, datasetID string) coverageReport {
	coverage := coverageReport{DatasetID: datasetID, RulesVersion: mappings.RulesVersion, CatalogSpecies: len(species), ExactEligibleVariants: len(assets), Missing: []missingAsset{}, MissingVariants: []missingVariant{}, Exclusions: mappings.Exclusions, SourceQualityFlags: []string{"Only explicitly mapped inherited artwork with matching source provenance is accepted.", "Generated candidates, source-only aliases, shiny fallbacks, and shiny-only collectibles are rejected.", "Image copyrights are separate from repository code licenses; no underlying-rights clearance is claimed."}}
	eligible := map[int]bool{}
	forms := map[string]bool{}
	genders := map[string]bool{}
	available := map[string]bool{}
	for _, asset := range assets {
		key := fmt.Sprintf("%d/%s/%s/%s", asset.SpeciesID, asset.FormID, asset.Gender, asset.Palette)
		available[key] = true
		if asset.Palette == "regular" {
			eligible[asset.SpeciesID] = true
			forms[fmt.Sprintf("%d/%s", asset.SpeciesID, asset.FormID)] = true
			if asset.FormID == "standard" {
				coverage.StandardRegularSprites++
			}
			if asset.Gender != "default" {
				genders[fmt.Sprintf("%d/%s/%s", asset.SpeciesID, asset.FormID, asset.Gender)] = true
			}
		} else {
			coverage.ShinySprites++
		}
		coverage.MaxSpriteWidth = max(coverage.MaxSpriteWidth, asset.Width)
		coverage.MaxSpriteHeight = max(coverage.MaxSpriteHeight, asset.Height)
	}
	coverage.EligibleSpecies = len(eligible)
	coverage.CollectibleForms = len(forms)
	coverage.DistinctVisualGenderSlots = len(genders)
	for _, s := range species {
		coverage.CatalogForms += len(s.Forms)
		for _, f := range s.Forms {
			if f.ID == "standard" && !available[fmt.Sprintf("%d/%s/%s/regular", s.ID, f.ID, f.DefaultGender)] {
				coverage.Missing = append(coverage.Missing, missingAsset{SpeciesID: s.ID, Name: s.Name, Reason: "No standard regular default asset is mapped; no fallback applied."})
			}
			for _, gender := range f.Genders {
				if !available[fmt.Sprintf("%d/%s/%s/regular", s.ID, f.ID, gender)] {
					coverage.MissingVariants = append(coverage.MissingVariants, missingVariant{SpeciesID: s.ID, Name: s.Name, FormID: f.ID, Gender: gender, Palette: "regular", Reason: "No audited asset is mapped for this exact appearance."})
				}
			}
		}
	}
	return coverage
}

func coverageMarkdown(coverage coverageReport) []byte {
	var output bytes.Buffer
	fmt.Fprintln(&output, "<!-- Code generated by tools/dataset; DO NOT EDIT. -->\n# Dataset coverage")
	fmt.Fprintf(&output, "\nDataset ID: `%s`\nRules: `%s`\n\n", coverage.DatasetID, coverage.RulesVersion)
	fmt.Fprintln(&output, "| Inventory | Count |\n| --- | --- |")
	for _, item := range []struct {
		name  string
		count int
	}{{"Catalog species", coverage.CatalogSpecies}, {"Catalog forms", coverage.CatalogForms}, {"Eligible encounter species", coverage.EligibleSpecies}, {"Standard regular sprites", coverage.StandardRegularSprites}, {"Shiny sprites", coverage.ShinySprites}, {"Collectible forms", coverage.CollectibleForms}, {"Distinct visual gender slots", coverage.DistinctVisualGenderSlots}, {"Exact eligible variants", coverage.ExactEligibleVariants}} {
		fmt.Fprintf(&output, "| %s | %d |\n", item.name, item.count)
	}
	fmt.Fprintf(&output, "\nMaximum cropped dimensions: %d × %d source pixels.\n", coverage.MaxSpriteWidth, coverage.MaxSpriteHeight)
	fmt.Fprintln(&output, "\n## Missing standard regular artwork\n\n| Number | Species | Reason |\n| --- | --- | --- |")
	for _, missing := range coverage.Missing {
		fmt.Fprintf(&output, "| #%03d | %s | %s |\n", missing.SpeciesID, missing.Name, missing.Reason)
	}
	fmt.Fprintln(&output, "\n## Missing regular appearances\n\n| Number | Species | Form | Gender | Reason |\n| --- | --- | --- | --- | --- |")
	for _, missing := range coverage.MissingVariants {
		fmt.Fprintf(&output, "| #%03d | %s | %s | %s | %s |\n", missing.SpeciesID, missing.Name, missing.FormID, missing.Gender, missing.Reason)
	}
	fmt.Fprint(&output, "\n## Exclusions\n\n")
	for _, note := range coverage.Exclusions {
		fmt.Fprintf(&output, "- %s\n", note)
	}
	fmt.Fprint(&output, "\n## Source quality flags\n\n")
	for _, note := range coverage.SourceQualityFlags {
		fmt.Fprintf(&output, "- %s\n", note)
	}
	return output.Bytes()
}

func thirdPartyNotices(lock sourceLock, cache string) ([]byte, error) {
	var output bytes.Buffer
	fmt.Fprintln(&output, "<!-- Code generated by tools/dataset; DO NOT EDIT. -->\n# Third-party notices\n\nPokéCRT is an independent fan project. Pokémon names and artwork retain their source rights and trademarks. Source repository code licenses do not transfer ownership of Pokémon images. The owner has approved attributed fan-project distribution under docs/release-policy.md. This records the distribution decision, not a grant or clearance of underlying image rights. Original PokéCRT code is MIT licensed; third-party artwork is not relicensed by PokéCRT.")
	for _, item := range []struct{ sourceID, path, heading string }{{"pokeapi", "LICENSE.md", "PokéAPI metadata terms"}, {"pokesprite-v2", "license.md", "PokéSprite-v2 code and non-image material terms"}, {"pokesprite-v2", "contributors.md", "PokéSprite-v2 source credits"}} {
		data, source, _, err := readInput(lock, cache, item.sourceID, item.path)
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(&output, "\n## %s\n\nSource: https://github.com/%s/tree/%s\n\n%s\n\n", item.heading, source.Repository, source.Revision, source.Terms)
		output.Write(data)
		output.WriteByte('\n')
	}
	return output.Bytes(), nil
}

func outputPaths(bundle generatedBundle) []string {
	paths := make([]string, 0, len(bundle.Files))
	for path := range bundle.Files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func verifyOutputInventory(bundle generatedBundle, root string) error {
	files, err := filepath.Glob(filepath.Join(root, "internal/sprite/assets/*.png"))
	if err != nil {
		return err
	}
	for _, filename := range files {
		relative, err := filepath.Rel(root, filename)
		if err != nil {
			return err
		}
		if _, ok := bundle.Files[filepath.ToSlash(relative)]; !ok {
			return fmt.Errorf("unexpected sprite asset %s; review and remove it explicitly before generating", relative)
		}
	}
	return nil
}

func writeBundle(bundle generatedBundle, root string) error {
	if err := verifyOutputInventory(bundle, root); err != nil {
		return err
	}
	for _, path := range outputPaths(bundle) {
		filename := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
			return err
		}
		file, err := os.CreateTemp(filepath.Dir(filename), ".generated-*")
		if err != nil {
			return err
		}
		if err := file.Chmod(0o644); err != nil {
			file.Close()
			os.Remove(file.Name())
			return err
		}
		_, writeErr := file.Write(bundle.Files[path])
		closeErr := file.Close()
		if writeErr != nil || closeErr != nil {
			os.Remove(file.Name())
			return fmt.Errorf("write generated output %s: write=%v close=%v", path, writeErr, closeErr)
		}
		if err := os.Rename(file.Name(), filename); err != nil {
			os.Remove(file.Name())
			return err
		}
	}
	return nil
}

func checkBundle(bundle generatedBundle, root string) error {
	if err := verifyOutputInventory(bundle, root); err != nil {
		return err
	}
	for _, path := range outputPaths(bundle) {
		actual, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			return fmt.Errorf("generated output missing/unreadable: %s: %w", path, err)
		}
		if !bytes.Equal(actual, bundle.Files[path]) {
			return fmt.Errorf("generated output drift: %s (check did not overwrite it)", path)
		}
	}
	return nil
}

// prepareAssets checks every tracked generated output before writing only sprites.
// It permits a clean source checkout without images and never hides metadata drift.
func prepareAssets(bundle generatedBundle, root string) error {
	assets := generatedBundle{Files: make(map[string][]byte)}
	metadata := generatedBundle{Files: make(map[string][]byte)}
	for path, data := range bundle.Files {
		if filepath.Ext(path) == ".png" {
			assets.Files[path] = data
		} else {
			metadata.Files[path] = data
		}
	}
	if err := verifyOutputInventory(bundle, root); err != nil {
		return err
	}
	// checkBundle also checks sprite inventory; retain the full inventory here.
	for _, path := range outputPaths(metadata) {
		actual, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			return fmt.Errorf("generated output missing/unreadable: %s: %w", path, err)
		}
		if !bytes.Equal(actual, metadata.Files[path]) {
			return fmt.Errorf("generated output drift: %s (prepare-assets did not overwrite it)", path)
		}
	}
	return writeBundle(assets, root)
}
