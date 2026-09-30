package main

import (
	"fmt"
	"slices"
	"strings"
)

func validateSourceNames(m mappingConfig) error {
	seen := map[int]bool{}
	for _, name := range m.SourceNameOverrides {
		if m.RulesVersion != "d06b-gen1-1" || !slices.Contains(m.CatalogSpecies, name.SpeciesID) || seen[name.SpeciesID] || strings.TrimSpace(name.SourceName) == "" || strings.TrimSpace(name.CanonicalName) == "" || name.SourceName == name.CanonicalName || strings.TrimSpace(name.Reason) == "" {
			return fmt.Errorf("invalid or duplicate source-name correction for #%03d", name.SpeciesID)
		}
		seen[name.SpeciesID] = true
	}
	return nil
}

func sourceNameMatches(m mappingConfig, id int, source, canonical string) bool {
	for _, name := range m.SourceNameOverrides {
		if name.SpeciesID == id {
			return name.SourceName == source && name.CanonicalName == canonical
		}
	}
	return source == canonical
}
