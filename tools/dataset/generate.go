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
	"strings"
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

type achievementInventory struct {
	Types               []string    `json:"types"`
	RegionalForms       int         `json:"regional_forms"`
	TransformationForms int         `json:"transformation_forms"`
	BranchingFamilies   [][]int     `json:"branching_families"`
	GenerationSpecies   map[int]int `json:"generation_species"`
	NationalSpecies     int         `json:"national_species"`
	Limitations         []string    `json:"limitations"`
}

type metadataVarietyGap struct {
	SpeciesID  int    `json:"species_id"`
	PokemonID  int    `json:"pokemon_id"`
	Identifier string `json:"identifier"`
	Reason     string `json:"reason"`
}

type inventoryAudit struct {
	FoldedSourceAliases int                  `json:"folded_source_aliases"`
	MetadataVarietyGaps []metadataVarietyGap `json:"metadata_varieties_without_identity"`
}

type coverageReport struct {
	InventoryAudit inventoryAudit `json:"inventory_audit"`

	AchievementInventory      achievementInventory `json:"achievement_inventory"`
	DatasetID                 string               `json:"dataset_id"`
	RulesVersion              string               `json:"rules_version"`
	CatalogSpecies            int                  `json:"catalog_species"`
	CatalogForms              int                  `json:"catalog_forms"`
	MissingVariants           []missingVariant     `json:"missing_regular_variants"`
	EligibleSpecies           int                  `json:"eligible_species"`
	StandardRegularSprites    int                  `json:"standard_regular_sprites"`
	ShinySprites              int                  `json:"shiny_sprites"`
	CollectibleForms          int                  `json:"collectible_forms"`
	DistinctVisualGenderSlots int                  `json:"distinct_visual_gender_slots"`
	ExactEligibleVariants     int                  `json:"exact_eligible_variants"`
	MaxSpriteWidth            int                  `json:"max_sprite_width"`
	MaxSpriteHeight           int                  `json:"max_sprite_height"`
	Missing                   []missingAsset       `json:"missing_standard_regular_assets"`
	Exclusions                []string             `json:"exclusions"`
	SourceQualityFlags        []string             `json:"source_quality_flags"`
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
	var err error
	if err = verifyReviewedAliasPixels(lock, cache, mappings); err != nil {
		return bundle, err
	}
	mappings, err = deriveMappings(lock, cache, mappings)
	if err != nil {
		return bundle, err
	}
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
	femaleSources := false
	previousFemale := make(map[int]bool)
	var inherited inheritedInventory
	previousIcons := make(map[string]bool)
	if mappings.RulesVersion == "d06b-gen1-1" || automaticRules(mappings.RulesVersion) {
		inherited, err = loadInheritedInventory(lock, cache)
		if err != nil {
			return bundle, err
		}
	}
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
		if selected.Slug != mapping.SourceSlug || selected.CanonicalForm != nil || available == nil || !*available || selected.IsGenerated == nil || *selected.IsGenerated || !ok || provenance.IsGenerated == nil || *provenance.IsGenerated || !providerReviewed(mappings, provenance.Source) || selected.Source != provenance.Source || (mapping.Provider != "" && mapping.Provider != provenance.Source) {
			return bundle, fmt.Errorf("unverified, generated, aliased, or mismatched asset %s", assetIdentity(mapping))
		}
		if mapping.Gender == "default" && selected.ID != form.SourceFormID {
			return bundle, fmt.Errorf("default asset uses a different source form %s", assetIdentity(mapping))
		}
		if inherited != nil {
			var previous bool
			var err error
			if mapping.Provider == "bamq/pokemon-sprites" {
				previous, err = verifyAssetProvider(lock, cache, mappings, mapping, sourceSpecies[mapping.SpeciesID].Slug, true)
			} else {
				previous, err = inherited.verify(mapping, sourceSpecies[mapping.SpeciesID].Slug)
			}
			if err != nil {
				return bundle, err
			}
			if previous {
				previousIcons[fmt.Sprintf("#%03d/%s", mapping.SpeciesID, mapping.FormID)] = true
			}
		}
		if mapping.SourceLayout == "gen8-female" {
			previous, err := verifyFemaleProvenance(lock, cache, mapping)
			if err != nil {
				return bundle, err
			}
			femaleSources = true
			if previous {
				previousFemale[mapping.SpeciesID] = true
			}
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

	kept, reasons, err := excludeDuplicatePalettes(assets, mappings.DuplicatePaletteExclusions)
	if err != nil {
		return bundle, err
	}
	keptPaths := map[string]bool{}
	for _, a := range kept {
		keptPaths[a.Path] = true
	}
	for _, a := range assets {
		if !keptPaths[a.Path] {
			delete(bundle.Files, "internal/sprite/"+a.Path)
		}
	}
	assets = kept
	mappings.Exclusions = append(mappings.Exclusions, reasons...)
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
	if automaticRuleLevel(mappings.RulesVersion) >= 12 {
		bundle.Coverage.InventoryAudit, err = auditMetadataInventory(lock, cache, mappings, species)
		if err != nil {
			return bundle, err
		}
	}
	if femaleSources {
		bundle.Coverage.SourceQualityFlags = append(bundle.Coverage.SourceQualityFlags, "Explicit gen8-female artwork verified against pinned inherited inventory; unofficial female candidates excluded.")
	}
	previousIDs := make([]int, 0, len(previousFemale))
	for id := range previousFemale {
		previousIDs = append(previousIDs, id)
	}
	sort.Ints(previousIDs)
	for _, id := range previousIDs {
		bundle.Coverage.SourceQualityFlags = append(bundle.Coverage.SourceQualityFlags, fmt.Sprintf("Female artwork for species #%03d retains a previous-generation icon as flagged by the source.", id))
	}
	previousKeys := make([]string, 0, len(previousIcons))
	for key := range previousIcons {
		previousKeys = append(previousKeys, key)
	}
	sort.Strings(previousKeys)
	for _, key := range previousKeys {
		bundle.Coverage.SourceQualityFlags = append(bundle.Coverage.SourceQualityFlags, "Retained previous-generation source artwork: "+key)
	}
	if slices.Contains(mappings.ReviewedProviders, "bamq/pokemon-sprites") {
		bundle.Coverage.SourceQualityFlags = append(bundle.Coverage.SourceQualityFlags, "bamq/pokemon-sprites: community artwork adapted upstream to 68x56; exact imported bytes verified at the pinned provider revision. Generated candidates remain excluded.")
	}
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
			fmt.Fprintf(&output, "{ID:%q,Name:%q,Types:%#v,Tags:%#v,DefaultGender:%q,Genders:%#v,SourceAliases:%#v},", f.ID, f.Name, f.Types, f.Tags, f.DefaultGender, f.Genders, f.SourceAliases)
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
	coverage := coverageReport{DatasetID: datasetID, RulesVersion: mappings.RulesVersion, CatalogSpecies: len(species), ExactEligibleVariants: len(assets), Missing: []missingAsset{}, MissingVariants: []missingVariant{}, Exclusions: mappings.Exclusions, SourceQualityFlags: []string{"Only automatically derived or explicitly mapped artwork with reviewed provider and exact source provenance is accepted.", "Generated candidates, source-only aliases, shiny fallbacks, and shiny-only collectibles are rejected.", "Image copyrights are separate from repository code licenses; no underlying-rights clearance is claimed."}}
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
	coverage.AchievementInventory = deriveAchievementInventory(species, eligible, forms, coverage.ShinySprites)
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
	fmt.Fprintln(&output, "\n## Achievement inventory\n\nTargets include accepted regular artwork only; metadata-only forms do not qualify.")
	a := coverage.AchievementInventory
	fmt.Fprintf(&output, "\nSupported types: %s.\nRegional forms: %d. Transformation forms: %d. National species: %d.\n", strings.Join(a.Types, ", "), a.RegionalForms, a.TransformationForms, a.NationalSpecies)
	gens := make([]int, 0, len(a.GenerationSpecies))
	for gen := range a.GenerationSpecies {
		gens = append(gens, gen)
	}
	sort.Ints(gens)
	for _, gen := range gens {
		fmt.Fprintf(&output, "\n- Generation %d: %d eligible species.\n", gen, a.GenerationSpecies[gen])
	}
	fmt.Fprintln(&output, "\nFully supported branching families (National numbers):")
	for _, family := range a.BranchingFamilies {
		fmt.Fprintf(&output, "\n- %v\n", family)
	}
	for _, limitation := range a.Limitations {
		fmt.Fprintf(&output, "\n- Limitation: %s\n", limitation)
	}
	if automaticRuleLevel(coverage.RulesVersion) >= 12 {
		fmt.Fprintln(&output, "\n## Source inventory audit")
		fmt.Fprintf(&output, "\nFolded source aliases: %d.\n", coverage.InventoryAudit.FoldedSourceAliases)
		fmt.Fprintln(&output, "\nThese metadata varieties lack resolved catalog/source identities. This is separate from unavailable artwork for catalog appearances; no images or identities are guessed. This report is variety-level, not a claim to enumerate all upstream cosmetic form records.\n\n| Species | Variety ID | Metadata identifier | Reason |\n| --- | --- | --- | --- |")
		for _, gap := range coverage.InventoryAudit.MetadataVarietyGaps {
			fmt.Fprintf(&output, "| #%03d | %d | %s | %s |\n", gap.SpeciesID, gap.PokemonID, gap.Identifier, gap.Reason)
		}
	}
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
	for _, src := range lock.Sources {
		if src.ID == "bamq" {
			fmt.Fprintf(&output, "\n## Community provider artwork\n\nSource: https://github.com/%s/tree/%s\n\n%s\n\n%s\n", src.Repository, src.Revision, src.Terms, src.Attribution)
			for _, name := range []string{"README.md", "contributors.md"} {
				data, _, _, err := readInput(lock, cache, "bamq", name)
				if err != nil {
					return nil, err
				}
				output.Write(data)
				output.WriteByte('\n')
			}
		}
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

func verifyReviewedAliasPixels(lock sourceLock, cache string, m mappingConfig) error {
	assets, err := reviewedAliasAssets(lock, cache, m)
	if err != nil {
		return err
	}
	if len(assets) == 0 {
		return nil
	}
	inherited, err := loadInheritedInventory(lock, cache)
	if err != nil {
		return err
	}
	data, _, _, err := readInput(lock, cache, "pokesprite-v2", "data/pokemon.json")
	if err != nil {
		return err
	}
	var manifest sourceManifest
	if err = json.Unmarshal(data, &manifest); err != nil {
		return err
	}
	indexData, _, _, err := readInput(lock, cache, "pokesprite-v2", "sources/generated/asset-index.json")
	if err != nil {
		return err
	}
	var index map[string]json.RawMessage
	if err = json.Unmarshal(indexData, &index); err != nil {
		return err
	}
	for _, asset := range assets {
		var slots map[string]struct {
			Source      string `json:"source"`
			IsGenerated *bool  `json:"is_generated"`
		}
		if err = json.Unmarshal(index[asset.SourceSlug], &slots); err != nil {
			return err
		}
		provenance := slots[asset.Palette]
		if provenance.Source != "msikma/pokesprite" || provenance.IsGenerated == nil || *provenance.IsGenerated {
			return fmt.Errorf("invalid reviewed alias provenance %d/%s/%s", asset.SpeciesID, asset.SourceFormID, asset.Palette)
		}
		var sp sourceSpecies
		for _, s := range manifest.Pokemon {
			if s.ID == asset.SpeciesID {
				sp = s
			}
		}
		if _, err = inherited.verify(asset, sp.Slug); err != nil {
			return err
		}
		target, err := sourceFormByID(sp, sp.DefaultForm)
		if err != nil {
			return err
		}
		original, _, _, err := readInput(lock, cache, asset.SourceID, asset.Path)
		if err != nil {
			return err
		}
		targetAsset := asset
		targetAsset.SourceSlug = target.Slug
		targetAsset.SourceFormID = target.ID
		targetAsset.Path, _ = assetSourcePath(targetAsset)
		other, _, _, err := readInput(lock, cache, targetAsset.SourceID, targetAsset.Path)
		if err != nil {
			return err
		}
		first, _, _, err := normalizePNG(original)
		if err != nil {
			return err
		}
		second, _, _, err := normalizePNG(other)
		if err != nil {
			return err
		}
		var correction reviewedDefaultAlias
		for _, a := range m.ReviewedDefaultAliases {
			if a.SpeciesID == asset.SpeciesID && a.SourceFormID == asset.SourceFormID {
				correction = a
			}
		}
		if digest(first) != correction.SourceHashes[asset.Palette] || digest(second) != correction.DefaultHashes[asset.Palette] {
			return fmt.Errorf("reviewed alias hash evidence differs %d/%s/%s", asset.SpeciesID, asset.SourceFormID, asset.Palette)
		}
	}
	return nil
}

// Availability restricts targets; it never supplies missing family members.
func deriveAchievementInventory(species []normalizedSpecies, eligible map[int]bool, forms map[string]bool, shiny int) achievementInventory {
	a := achievementInventory{Types: []string{}, BranchingFamilies: [][]int{}, GenerationSpecies: map[int]int{}, Limitations: []string{}}
	types := map[string]bool{}
	children := map[int][]int{}
	for _, s := range species {
		if s.EvolvesFrom > 0 {
			children[s.EvolvesFrom] = append(children[s.EvolvesFrom], s.ID)
		}
		if eligible[s.ID] {
			a.NationalSpecies++
			a.GenerationSpecies[s.Generation]++
		}
		for _, f := range s.Forms {
			if !forms[fmt.Sprintf("%d/%s", s.ID, f.ID)] {
				continue
			}
			for _, typ := range f.Types {
				types[typ] = true
			}
			if slices.Contains(f.Tags, "regional") {
				a.RegionalForms++
			}
			if slices.Contains(f.Tags, "mega") || slices.Contains(f.Tags, "gigantamax") {
				a.TransformationForms++
			}
		}
	}
	for typ := range types {
		a.Types = append(a.Types, typ)
	}
	sort.Strings(a.Types)
	roots := []int{}
	for _, s := range species {
		if s.EvolvesFrom == 0 {
			roots = append(roots, s.ID)
		}
	}
	sort.Ints(roots)
	for _, root := range roots {
		family := []int{}
		queue := []int{root}
		complete, branching := true, false
		for len(queue) > 0 {
			id := queue[0]
			queue = queue[1:]
			family = append(family, id)
			if !eligible[id] {
				complete = false
			}
			if len(children[id]) > 1 {
				branching = true
			}
			queue = append(queue, children[id]...)
		}
		if complete && branching {
			sort.Ints(family)
			a.BranchingFamilies = append(a.BranchingFamilies, family)
		}
	}
	if shiny == 0 {
		a.Limitations = append(a.Limitations, "shiny.first: no accepted shiny slots")
	}
	if len(a.Types) == 0 {
		a.Limitations = append(a.Limitations, "types.complete: no eligible form types")
	}
	if a.RegionalForms == 0 {
		a.Limitations = append(a.Limitations, "regional.first: no eligible regional forms")
	}
	if a.TransformationForms == 0 {
		a.Limitations = append(a.Limitations, "transformation.first: no eligible Mega/Gigantamax forms")
	}
	if len(a.BranchingFamilies) == 0 {
		a.Limitations = append(a.Limitations, "evolution.branching: no fully supported branching families")
	}
	if a.NationalSpecies == 0 {
		a.Limitations = append(a.Limitations, "national.complete: no eligible species")
	}
	return a
}

// Report metadata/source scope gaps separately from unavailable catalog assets.
// A canonical image pointer cannot conceal conflicting ordinary form typing.
func auditMetadataInventory(lock sourceLock, cache string, mappings mappingConfig, species []normalizedSpecies) (inventoryAudit, error) {
	audit := inventoryAudit{MetadataVarietyGaps: []metadataVarietyGap{}}
	rows, err := readTable(lock, cache, "pokemon.csv", "id", "identifier", "species_id")
	if err != nil {
		return audit, err
	}
	type variety struct {
		id, owner  int
		identifier string
	}
	byName := map[string]variety{}
	varieties := []variety{}
	for _, row := range rows {
		id, err := integer(row, "id", false)
		if err != nil {
			return audit, err
		}
		owner, err := integer(row, "species_id", false)
		if err != nil {
			return audit, err
		}
		v := variety{id, owner, row["identifier"]}
		byName[v.identifier] = v
		varieties = append(varieties, v)
	}
	types, err := identifiers(lock, cache, "types.csv")
	if err != nil {
		return audit, err
	}
	typeRows, err := readTable(lock, cache, "pokemon_types.csv", "pokemon_id", "type_id", "slot")
	if err != nil {
		return audit, err
	}
	slots := map[int]map[int]string{}
	for _, row := range typeRows {
		id, e := integer(row, "pokemon_id", false)
		if e != nil {
			return audit, e
		}
		typ, e := integer(row, "type_id", false)
		if e != nil {
			return audit, e
		}
		slot, e := integer(row, "slot", false)
		if e != nil {
			return audit, e
		}
		if slots[id] == nil {
			slots[id] = map[int]string{}
		}
		slots[id][slot] = types[typ]
	}
	_, formSlots, err := metadataFormTyping(lock, cache, mappings.RulesVersion)
	if err != nil {
		return audit, err
	}
	formRows, err := readTable(lock, cache, "pokemon_forms.csv", "id", "identifier", "pokemon_id")
	if err != nil {
		return audit, err
	}
	formIDs := map[string]int{}
	formVarieties := map[string]int{}
	for _, row := range formRows {
		id, e := integer(row, "id", false)
		if e != nil {
			return audit, e
		}
		owner, e := integer(row, "pokemon_id", false)
		if e != nil {
			return audit, e
		}
		formIDs[row["identifier"]] = id
		formVarieties[row["identifier"]] = owner
	}
	selected := map[int]bool{}
	used := map[int]bool{}
	for _, sp := range species {
		selected[sp.ID] = true
	}
	for _, f := range mappings.Forms {
		used[f.PokemonID] = true
		for _, g := range f.SourceGenders {
			used[g.PokemonID] = true
		}
	}
	for _, sp := range species {
		for _, f := range sp.Forms {
			// Explicit accepted or metadata-only gender declarations can resolve a
			// separate metadata variety without another collectible form.
			for _, gender := range f.Genders {
				if gender == "default" {
					continue
				}
				if v, ok := byName[sp.Slug+"-"+gender]; ok && v.owner == sp.ID {
					used[v.id] = true
				}
			}
			for _, alias := range f.SourceAliases {
				audit.FoldedSourceAliases++
				key := sp.Slug + "-" + alias
				varietyID := formVarieties[key]
				if v, ok := byName[key]; ok {
					if v.owner != sp.ID {
						return audit, fmt.Errorf("source alias metadata owner mismatch #%03d/%s", sp.ID, alias)
					}
					if formVarieties[key] != 0 && formVarieties[key] != v.id {
						return audit, fmt.Errorf("source alias variety/form disagreement #%03d/%s", sp.ID, alias)
					}
					varietyID = v.id
				}
				if varietyID == 0 {
					continue
				} // Source-only synonyms have no metadata typing.
				foundOwner := false
				for _, v := range varieties {
					if v.id == varietyID && v.owner == sp.ID {
						foundOwner = true
						break
					}
				}
				if !foundOwner {
					return audit, fmt.Errorf("source alias form owner mismatch #%03d/%s", sp.ID, alias)
				}
				used[varietyID] = true
				exact := slots[varietyID]
				if values := formSlots[formIDs[key]]; len(values) > 0 {
					exact = values
				}
				aliasTypes := []string{exact[1]}
				if exact[2] != "" {
					aliasTypes = append(aliasTypes, exact[2])
				}
				canonicalTypes := slices.Clone(f.Types)
				sort.Strings(canonicalTypes)
				sort.Strings(aliasTypes)
				if !slices.Equal(aliasTypes, canonicalTypes) {
					return audit, fmt.Errorf("source alias metadata typing conflict #%03d/%s: %v versus %v; review identity without borrowing artwork", sp.ID, alias, aliasTypes, f.Types)
				}
			}
		}
	}
	for _, v := range varieties {
		if selected[v.owner] && !used[v.id] {
			audit.MetadataVarietyGaps = append(audit.MetadataVarietyGaps, metadataVarietyGap{SpeciesID: v.owner, PokemonID: v.id, Identifier: v.identifier, Reason: "Pinned metadata variety has no resolved catalog/source identity; no artwork inferred."})
		}
	}
	sort.Slice(audit.MetadataVarietyGaps, func(i, j int) bool {
		a, b := audit.MetadataVarietyGaps[i], audit.MetadataVarietyGaps[j]
		if a.SpeciesID != b.SpeciesID {
			return a.SpeciesID < b.SpeciesID
		}
		return a.PokemonID < b.PokemonID
	})
	return audit, nil
}
