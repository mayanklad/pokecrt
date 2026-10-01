package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
)

type sourceNameOverride struct {
	SpeciesID     int    `json:"species_id"`
	SourceName    string `json:"source_name"`
	CanonicalName string `json:"canonical_name"`
	Reason        string `json:"reason"`
}

type sourceFormExclusion struct {
	UnsupportedType string `json:"unsupported_type,omitempty"`
	SpeciesID       int    `json:"species_id"`
	SourceFormID    string `json:"source_form_id"`
	Reason          string `json:"reason"`
}

type mappingConfig struct {
	SourceFormExclusions []sourceFormExclusion `json:"source_form_exclusions,omitempty"`
	Selection            *inventorySelection   `json:"selection,omitempty"`
	FormOverrides        []formMapping         `json:"form_overrides,omitempty"`
	SourceNameOverrides  []sourceNameOverride  `json:"source_name_overrides,omitempty"`
	RulesVersion         string                `json:"rules_version"`
	CatalogSpecies       []int                 `json:"catalog_species"`
	SourceStandardForm   string                `json:"source_standard_form"`
	StandardFormReason   string                `json:"standard_form_reason"`
	Assets               []assetMapping        `json:"assets"`
	Forms                []formMapping         `json:"forms,omitempty"`
	Exclusions           []string              `json:"exclusions"`
}

type assetMapping struct {
	SourceLayout string `json:"source_layout,omitempty"`
	SpeciesID    int    `json:"species_id"`
	SourceID     string `json:"source_id"`
	Path         string `json:"path"`
	SourceSlug   string `json:"source_slug"`
	Reason       string `json:"reason"`
	FormID       string `json:"form_id,omitempty"`
	SourceFormID string `json:"source_form_id,omitempty"`
	Gender       string `json:"gender,omitempty"`
	Palette      string `json:"palette,omitempty"`
}

type formMapping struct {
	MetadataFormID int      `json:"metadata_form_id,omitempty"`
	SpeciesID      int      `json:"species_id"`
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	SourceFormID   string   `json:"source_form_id"`
	PokemonID      int      `json:"pokemon_id"`
	DefaultGender  string   `json:"default_gender"`
	Genders        []string `json:"genders"`
	SourceAliases  []string `json:"source_aliases,omitempty"`
	Reason         string   `json:"reason"`
}

type normalizedForm struct {
	ID            string
	Name          string
	Types         []string
	DefaultGender string
	Genders       []string
	SourceFormID  string
	SourceAliases []string
}

type normalizedSpecies struct {
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
	Types       []string
	Forms       []normalizedForm
}

type sourceManifest struct {
	Pokemon []sourceSpecies `json:"pokemon"`
}

type sourceSpecies struct {
	ID          int          `json:"dex"`
	Name        string       `json:"species_name"`
	Slug        string       `json:"species_slug"`
	Aliases     []string     `json:"species_aliases"`
	DefaultForm string       `json:"default_form"`
	Forms       []sourceForm `json:"forms"`
}

type sourceForm struct {
	ID            string  `json:"id"`
	Label         string  `json:"label"`
	Slug          string  `json:"file_slug"`
	CanonicalForm *string `json:"canonical_form"`
	HasRegular    *bool   `json:"has_regular"`
	HasShiny      *bool   `json:"has_shiny"`
	IsGenerated   *bool   `json:"is_generated"`
	Source        string  `json:"source"`
}

func readMappings(filename string) (mappingConfig, error) {
	var mappings mappingConfig
	data, err := os.ReadFile(filename)
	if err != nil {
		return mappings, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&mappings); err != nil {
		return mappings, fmt.Errorf("decode mappings: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return mappings, fmt.Errorf("mappings must contain exactly one JSON document")
	}
	if mappings.Selection != nil {
		if err := validateSelection(mappings); err != nil {
			return mappings, err
		}
		return mappings, nil
	}
	if (mappings.RulesVersion != "d02b-1" && mappings.RulesVersion != "d06a-1" && mappings.RulesVersion != "d06b-gender-1" && mappings.RulesVersion != "d06b-gen1-1" && (mappings.RulesVersion != "d06-auto-1" && mappings.RulesVersion != "d06-auto-2" && mappings.RulesVersion != "d06-auto-3" && mappings.RulesVersion != "d06-auto-4")) || mappings.SourceStandardForm != "base" || strings.TrimSpace(mappings.StandardFormReason) == "" || len(mappings.CatalogSpecies) == 0 {
		return mappings, fmt.Errorf("mappings require supported rules, base-to-standard reason, and species IDs")
	}
	sort.Ints(mappings.CatalogSpecies)
	for i, id := range mappings.CatalogSpecies {
		if id <= 0 || (i > 0 && id == mappings.CatalogSpecies[i-1]) {
			return mappings, fmt.Errorf("invalid or duplicate mapped species ID %d", id)
		}
	}
	if err := validateMappings(&mappings); err != nil {
		return mappings, err
	}
	return mappings, nil
}

func readInput(lock sourceLock, cache, sourceID, path string) ([]byte, source, sourceFile, error) {
	for _, source := range lock.Sources {
		if source.ID != sourceID {
			continue
		}
		for _, file := range source.Files {
			if file.Path != path {
				continue
			}
			data, err := os.ReadFile(filepath.Join(cache, source.ID, filepath.FromSlash(path)))
			if err != nil {
				return nil, source, file, err
			}
			if err := verifyBytes(file, data); err != nil {
				return nil, source, file, fmt.Errorf("%s/%s: %w", source.ID, file.Path, err)
			}
			return data, source, file, nil
		}
	}
	return nil, source{}, sourceFile{}, fmt.Errorf("input %s/%s is not pinned in the source lock", sourceID, path)
}

func readTable(lock sourceLock, cache, filename string, required ...string) ([]map[string]string, error) {
	data, _, _, err := readInput(lock, cache, "pokeapi", "data/v2/csv/"+filename)
	if err != nil {
		return nil, err
	}
	return parseTable(data, required...)
}

func parseTable(data []byte, required ...string) ([]map[string]string, error) {
	reader := csv.NewReader(bytes.NewReader(data))
	header, err := reader.Read()
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool)
	for _, name := range header {
		if name == "" || seen[name] {
			return nil, fmt.Errorf("empty or duplicate CSV column %q", name)
		}
		seen[name] = true
	}
	for _, name := range required {
		if !seen[name] {
			return nil, fmt.Errorf("missing CSV column %s", name)
		}
	}
	var rows []map[string]string
	for {
		values, err := reader.Read()
		if err == io.EOF {
			return rows, nil
		}
		if err != nil {
			return nil, err
		}
		row := make(map[string]string, len(header))
		for i, name := range header {
			row[name] = values[i]
		}
		rows = append(rows, row)
	}
}

func integer(row map[string]string, field string, optional bool) (int, error) {
	if optional && row[field] == "" {
		return 0, nil
	}
	number, err := strconv.Atoi(row[field])
	if err != nil || number <= 0 {
		return 0, fmt.Errorf("invalid positive integer %s=%q", field, row[field])
	}
	return number, nil
}

func sourceBool(row map[string]string, field string) (bool, error) {
	switch row[field] {
	case "0":
		return false, nil
	case "1":
		return true, nil
	default:
		return false, fmt.Errorf("invalid source boolean %s=%q", field, row[field])
	}
}

func identifiers(lock sourceLock, cache, filename string) (map[int]string, error) {
	rows, err := readTable(lock, cache, filename, "id", "identifier")
	if err != nil {
		return nil, err
	}
	result := make(map[int]string)
	for _, row := range rows {
		id, err := integer(row, "id", false)
		if err != nil {
			return nil, err
		}
		if result[id] != "" || row["identifier"] == "" {
			return nil, fmt.Errorf("invalid or duplicate identity in %s", filename)
		}
		result[id] = row["identifier"]
	}
	return result, nil
}

func evolutionStages(parents map[int]int) (map[int]int, error) {
	stages := make(map[int]int)
	visiting := make(map[int]bool)
	var visit func(int) (int, error)
	visit = func(id int) (int, error) {
		if stage := stages[id]; stage != 0 {
			return stage, nil
		}
		parent, ok := parents[id]
		if !ok {
			return 0, fmt.Errorf("evolution references missing species #%03d", id)
		}
		if visiting[id] {
			return 0, fmt.Errorf("evolution cycle at species #%03d", id)
		}
		visiting[id] = true
		stage := 1
		if parent != 0 {
			previous, err := visit(parent)
			if err != nil {
				return 0, err
			}
			stage = previous + 1
		}
		visiting[id] = false
		stages[id] = stage
		return stage, nil
	}
	ids := make([]int, 0, len(parents))
	for id := range parents {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		if _, err := visit(id); err != nil {
			return nil, err
		}
	}
	return stages, nil
}

func normalizeCatalog(lock sourceLock, cache string, mappings mappingConfig) ([]normalizedSpecies, map[int]sourceSpecies, error) {
	colors, err := identifiers(lock, cache, "pokemon_colors.csv")
	if err != nil {
		return nil, nil, err
	}
	generations, err := identifiers(lock, cache, "generations.csv")
	if err != nil {
		return nil, nil, err
	}
	types, err := identifiers(lock, cache, "types.csv")
	if err != nil {
		return nil, nil, err
	}
	rows, err := readTable(lock, cache, "pokemon_species.csv", "id", "identifier", "generation_id", "evolves_from_species_id", "color_id", "is_baby", "is_legendary", "is_mythical", "has_gender_differences")
	if err != nil {
		return nil, nil, err
	}
	speciesRows := make(map[int]map[string]string)
	parents := make(map[int]int)
	children := make(map[int][]int)
	for _, row := range rows {
		id, err := integer(row, "id", false)
		if err != nil {
			return nil, nil, err
		}
		parent, err := integer(row, "evolves_from_species_id", true)
		if err != nil {
			return nil, nil, err
		}
		if speciesRows[id] != nil || row["identifier"] == "" {
			return nil, nil, fmt.Errorf("duplicate or invalid species #%03d", id)
		}
		speciesRows[id], parents[id] = row, parent
		if parent != 0 {
			children[parent] = append(children[parent], id)
		}
	}
	stages, err := evolutionStages(parents)
	if err != nil {
		return nil, nil, err
	}
	names := make(map[int]string)
	nameRows, err := readTable(lock, cache, "pokemon_species_names.csv", "pokemon_species_id", "local_language_id", "name")
	if err != nil {
		return nil, nil, err
	}
	for _, row := range nameRows {
		if row["local_language_id"] != "9" {
			continue
		}
		id, err := integer(row, "pokemon_species_id", false)
		if err != nil || names[id] != "" || strings.TrimSpace(row["name"]) == "" {
			return nil, nil, fmt.Errorf("invalid or duplicate English species name")
		}
		names[id] = row["name"]
	}
	defaults := make(map[int]int)
	varietyOwners := make(map[int]int)
	pokemonRows, err := readTable(lock, cache, "pokemon.csv", "id", "species_id", "is_default")
	if err != nil {
		return nil, nil, err
	}
	for _, row := range pokemonRows {
		isDefault, err := sourceBool(row, "is_default")
		if err != nil {
			return nil, nil, err
		}

		id, err := integer(row, "id", false)
		if err != nil {
			return nil, nil, err
		}
		speciesID, err := integer(row, "species_id", false)
		if err != nil || speciesRows[speciesID] == nil || varietyOwners[id] != 0 || (isDefault && defaults[speciesID] != 0) {
			return nil, nil, fmt.Errorf("invalid or duplicate default Pokemon variety")
		}
		varietyOwners[id] = speciesID
		if isDefault {
			defaults[speciesID] = id
		}
	}
	typeSlots := make(map[int]map[int]string)
	typeRows, err := readTable(lock, cache, "pokemon_types.csv", "pokemon_id", "type_id", "slot")
	if err != nil {
		return nil, nil, err
	}
	for _, row := range typeRows {
		id, err := integer(row, "pokemon_id", false)
		if err != nil {
			return nil, nil, err
		}
		typeID, err := integer(row, "type_id", false)
		if err != nil || types[typeID] == "" {
			return nil, nil, fmt.Errorf("unknown Pokemon type")
		}
		slot, err := integer(row, "slot", false)
		if err != nil || slot > 2 {
			return nil, nil, fmt.Errorf("invalid Pokemon type slot")
		}
		if typeSlots[id] == nil {
			typeSlots[id] = make(map[int]string)
		}
		if typeSlots[id][slot] != "" {
			return nil, nil, fmt.Errorf("duplicate Pokemon type slot")
		}
		typeSlots[id][slot] = types[typeID]
	}
	data, _, _, err := readInput(lock, cache, "pokesprite-v2", "data/pokemon.json")
	if err != nil {
		return nil, nil, err
	}
	var manifest sourceManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, nil, err
	}
	sourceSpeciesByID := make(map[int]sourceSpecies)
	for _, species := range manifest.Pokemon {
		if species.ID <= 0 || sourceSpeciesByID[species.ID].ID != 0 {
			return nil, nil, fmt.Errorf("invalid or duplicate source manifest species")
		}
		sourceSpeciesByID[species.ID] = species
	}
	formOwners, formTypes, err := metadataFormTyping(lock, cache, mappings.RulesVersion)
	if err != nil {
		return nil, nil, err
	}
	var result []normalizedSpecies
	aliasOwners := make(map[string]int)
	for _, id := range mappings.CatalogSpecies {
		row, ok := speciesRows[id]
		if !ok || names[id] == "" {
			return nil, nil, fmt.Errorf("mapped species #%03d lacks metadata", id)
		}
		generation, err := integer(row, "generation_id", false)
		if err != nil || generations[generation] == "" {
			return nil, nil, fmt.Errorf("invalid generation for #%03d", id)
		}
		color, err := integer(row, "color_id", false)
		if err != nil || colors[color] == "" {
			return nil, nil, fmt.Errorf("invalid color for #%03d", id)
		}
		baby, err := sourceBool(row, "is_baby")
		if err != nil {
			return nil, nil, err
		}
		legendary, err := sourceBool(row, "is_legendary")
		if err != nil {
			return nil, nil, err
		}
		mythical, err := sourceBool(row, "is_mythical")
		if err != nil {
			return nil, nil, err
		}
		if _, err := sourceBool(row, "has_gender_differences"); err != nil {
			return nil, nil, err
		}
		sourceSpecies := sourceSpeciesByID[id]
		if sourceSpecies.Slug != row["identifier"] || !sourceNameMatches(mappings, id, sourceSpecies.Name, names[id]) || ((mappings.RulesVersion != "d06-auto-1" && mappings.RulesVersion != "d06-auto-2" && mappings.RulesVersion != "d06-auto-3" && mappings.RulesVersion != "d06-auto-4") && sourceSpecies.DefaultForm != mappings.SourceStandardForm) {
			return nil, nil, fmt.Errorf("source identity/default mapping mismatch for #%03d", id)
		}
		if defaults[id] == 0 || typeSlots[defaults[id]][1] == "" {
			return nil, nil, fmt.Errorf("missing standard typing for #%03d", id)
		}
		selectedTypes := []string{typeSlots[defaults[id]][1]}
		if second := typeSlots[defaults[id]][2]; second != "" {
			if second == selectedTypes[0] {
				return nil, nil, fmt.Errorf("duplicate typing for #%03d", id)
			}
			selectedTypes = append(selectedTypes, second)
		}
		relations := append([]int(nil), children[id]...)
		sort.Ints(relations)
		for _, related := range append(append([]int(nil), relations...), parents[id]) {
			if related != 0 && !slices.Contains(mappings.CatalogSpecies, related) {
				return nil, nil, fmt.Errorf("mapped catalog omits related species #%03d; include complete evolution families", related)
			}
		}
		aliases := []string{row["identifier"], strings.ToLower(names[id]), strings.ToLower(sourceSpecies.Name)}
		for _, alias := range sourceSpecies.Aliases {
			alias = strings.ToLower(strings.TrimSpace(alias))
			if alias == "" {
				return nil, nil, fmt.Errorf("empty alias for #%03d", id)
			}
			aliases = append(aliases, alias)
		}
		sort.Strings(aliases)
		aliases = slices.Compact(aliases)
		for _, alias := range aliases {
			if owner := aliasOwners[alias]; owner != 0 && owner != id {
				return nil, nil, fmt.Errorf("ambiguous species alias %q", alias)
			}
			aliasOwners[alias] = id
		}
		forms, err := normalizeForms(id, sourceSpecies, mappings, defaults[id], varietyOwners, typeSlots, metadataTyping{Owners: formOwners, Slots: formTypes})
		if err != nil {
			return nil, nil, err
		}
		result = append(result, normalizedSpecies{ID: id, Name: names[id], Slug: row["identifier"], Aliases: aliases, Generation: generation, Color: colors[color], Stage: stages[id], Baby: baby, Legendary: legendary, Mythical: mythical, EvolvesFrom: parents[id], EvolvesTo: relations, Types: selectedTypes, Forms: forms})
	}
	return result, sourceSpeciesByID, nil
}

type metadataTyping struct {
	Owners map[int]int
	Slots  map[int]map[int]string
}

func supportedPokemonType(value string) bool {
	return slices.Contains([]string{"normal", "fire", "water", "electric", "grass", "ice", "fighting", "poison", "ground", "flying", "psychic", "bug", "rock", "ghost", "dragon", "dark", "steel", "fairy"}, value)
}

// Form type rows override variety types only for their exact metadata form owner.
func metadataFormTyping(lock sourceLock, cache, rules string) (map[int]int, map[int]map[int]string, error) {
	owners := map[int]int{}
	slots := map[int]map[int]string{}
	pinned := false
	for _, src := range lock.Sources {
		if src.ID == "pokeapi" {
			for _, file := range src.Files {
				if file.Path == "data/v2/csv/pokemon_form_types.csv" {
					pinned = true
				}
			}
		}
	}
	if !pinned {
		if rules == "d06-auto-4" {
			return nil, nil, fmt.Errorf("automatic rules v4 require pinned pokemon_form_types.csv")
		}
		return owners, slots, nil
	}
	forms, e := readTable(lock, cache, "pokemon_forms.csv", "id", "pokemon_id")
	if e != nil {
		return nil, nil, e
	}
	for _, r := range forms {
		id, e := integer(r, "id", false)
		if e != nil {
			return nil, nil, e
		}
		owner, e := integer(r, "pokemon_id", false)
		if e != nil || owners[id] != 0 {
			return nil, nil, fmt.Errorf("invalid or duplicate metadata form owner")
		}
		owners[id] = owner
	}
	types, e := identifiers(lock, cache, "types.csv")
	if e != nil {
		return nil, nil, e
	}
	rows, e := readTable(lock, cache, "pokemon_form_types.csv", "pokemon_form_id", "type_id", "slot")
	if e != nil {
		return nil, nil, e
	}
	for _, r := range rows {
		id, e := integer(r, "pokemon_form_id", false)
		if e != nil || owners[id] == 0 {
			return nil, nil, fmt.Errorf("unknown metadata form typing owner")
		}
		typ, e := integer(r, "type_id", false)
		if e != nil || types[typ] == "" {
			return nil, nil, fmt.Errorf("unknown metadata form type")
		}
		slot, e := integer(r, "slot", false)
		if e != nil || slot > 2 {
			return nil, nil, fmt.Errorf("invalid metadata form type slot")
		}
		if slots[id] == nil {
			slots[id] = map[int]string{}
		}
		if slots[id][slot] != "" {
			return nil, nil, fmt.Errorf("duplicate metadata form type slot")
		}
		slots[id][slot] = types[typ]
	}
	for _, values := range slots {
		if values[1] == "" || values[1] == values[2] {
			return nil, nil, fmt.Errorf("invalid metadata form type cardinality")
		}
	}
	return owners, slots, nil
}
