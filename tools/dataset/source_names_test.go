package main

import (
	"slices"
	"testing"
)

func TestExactDocumentedSourceNameCorrection(t *testing.T) {
	lock, cache, m := variantFixture(t)
	editManifest(t, &lock, cache, func(source *sourceManifest) { source.Pokemon[0].Name = "Fixture Bulbasaur" })
	m.RulesVersion = "d06b-gen1-1"
	if _, _, err := normalizeCatalog(lock, cache, m); err == nil {
		t.Fatal("silently accepted different source name")
	}
	m.SourceNameOverrides = []sourceNameOverride{{SpeciesID: 1, SourceName: "Fixture Bulbasaur", CanonicalName: "Bulbasaur", Reason: "Fixture pinned-source spelling difference"}}
	if err := validateMappings(&m); err != nil {
		t.Fatal(err)
	}
	species, _, err := normalizeCatalog(lock, cache, m)
	if err != nil {
		t.Fatal(err)
	}
	if species[0].Name != "Bulbasaur" || !slices.Contains(species[0].Aliases, "fixture bulbasaur") {
		t.Fatalf("lost canonical name or source alias: %+v", species[0])
	}
	m.SourceNameOverrides[0].SourceName = "Stale correction"
	if _, _, err := normalizeCatalog(lock, cache, m); err == nil {
		t.Fatal("accepted stale correction")
	}
	m.SourceNameOverrides = append(m.SourceNameOverrides, m.SourceNameOverrides[0])
	if err := validateMappings(&m); err == nil {
		t.Fatal("accepted duplicate correction")
	}
}
