package main

import (
	"bytes"
	"image/color"
	"image/png"
	"slices"
	"strings"
	"testing"
)

func TestInheritedQualityGate(t *testing.T) {
	for _, tc := range []struct {
		name, snapshot string
		valid          bool
	}{
		{"audited old generation", `{"001":{"idx":"001","slug":{"eng":"bulbasaur"},"gen-8":{"forms":{"$":{"is_prev_gen_icon":true}}}}}`, true},
		{"provisional", `{"001":{"idx":"001","slug":{"eng":"bulbasaur"},"gen-8":{"forms":{"$":{"is_unofficial_icon":true}}}}}`, false},
		{"wrong species", `{"001":{"idx":"002","slug":{"eng":"bulbasaur"},"gen-8":{"forms":{"$":{}}}}}`, false},
		{"wrong slug", `{"001":{"idx":"001","slug":{"eng":"other"},"gen-8":{"forms":{"$":{}}}}}`, false},
		{"missing appearance", `{"001":{"idx":"001","slug":{"eng":"bulbasaur"},"gen-8":{"forms":{}}}}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			lock, cache, m := variantFixture(t)
			m.RulesVersion = "d06b-gen1-1"
			fixtureInput(t, &lock, cache, "pokesprite-v2", "sources/upstreams/msikma-pokemon.json", []byte(tc.snapshot))
			bundle, err := buildBundle(lock, cache, m)
			if !tc.valid {
				if err == nil {
					t.Fatal("accepted unverified/provisional source")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(strings.Join(bundle.Coverage.SourceQualityFlags, " "), "Retained previous-generation source artwork: #001/standard") {
				t.Fatal("lost retained-generation provenance")
			}
		})
	}
}

func TestPinnedGen8FemaleLayout(t *testing.T) {
	for _, tc := range []struct {
		name, snapshot string
		valid          bool
	}{
		{"verified", `{"001":{"idx":"001","slug":{"eng":"bulbasaur"},"gen-8":{"forms":{"$":{"has_female":true,"is_prev_gen_icon":true}}}}}`, true},
		{"missing declaration", `{"001":{"idx":"001","slug":{"eng":"bulbasaur"},"gen-8":{"forms":{"$":{}}}}}`, false},
		{"unofficial", `{"001":{"idx":"001","slug":{"eng":"bulbasaur"},"gen-8":{"forms":{"$":{"has_female":true,"has_unofficial_female_icon":true}}}}}`, false},
		{"wrong owner", `{"001":{"idx":"002","slug":{"eng":"bulbasaur"},"gen-8":{"forms":{"$":{"has_female":true}}}}}`, false},
		{"wrong slug", `{"001":{"idx":"001","slug":{"eng":"other"},"gen-8":{"forms":{"$":{"has_female":true}}}}}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			lock, cache, m := variantFixture(t)
			m.RulesVersion = "d06b-gender-1"
			m.Forms[0].DefaultGender = "male"
			m.Forms[0].Genders = []string{"male", "female"}
			m.Assets[0].Gender = "male"
			female := m.Assets[0]
			female.Gender = "female"
			female.SourceLayout = "gen8-female"
			female.Path = "pokemon-gen8/regular/female/bulbasaur.png"
			m.Assets = append(m.Assets, female)
			pixels, err := png.Decode(bytes.NewReader(encodeFixturePNG(t, false)))
			if err != nil {
				t.Fatal(err)
			}
			pixels.(interface{ Set(int, int, color.Color) }).Set(1, 2, color.NRGBA{G: 255, A: 255})
			var output bytes.Buffer
			if err := png.Encode(&output, pixels); err != nil {
				t.Fatal(err)
			}
			fixtureInput(t, &lock, cache, "pokesprite-v2", female.Path, output.Bytes())
			fixtureInput(t, &lock, cache, "pokesprite-v2", "sources/upstreams/msikma-pokemon.json", []byte(tc.snapshot))
			bundle, err := buildBundle(lock, cache, m)
			if !tc.valid {
				if err == nil {
					t.Fatal("accepted unverified female provenance")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if bundle.Coverage.DistinctVisualGenderSlots != 2 || bundle.Coverage.CollectibleForms != 1 {
				t.Fatalf("gender counted as another form: %+v", bundle.Coverage)
			}
			if !strings.Contains(strings.Join(bundle.Coverage.SourceQualityFlags, " "), "previous-generation") {
				t.Fatal("lost source quality flag")
			}
		})
	}
}

func TestFemaleLayoutCannotBeReusedForOtherIdentities(t *testing.T) {
	for _, a := range []assetMapping{
		{SourceLayout: "unknown", Gender: "female", FormID: "standard", SourceFormID: "base"},
		{SourceLayout: "gen8-female", Gender: "male", FormID: "standard", SourceFormID: "base"},
		{SourceLayout: "gen8-female", Gender: "female", FormID: "mega", SourceFormID: "mega"},
	} {
		if _, err := assetSourcePath(a); err == nil {
			t.Fatalf("accepted wrong layout identity: %+v", a)
		}
	}
}

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

func TestReviewedProviderEvidence(t *testing.T) {
	for _, name := range []string{"verified", "unreviewed", "wrong revision", "missing credits", "different bytes", "unsupported gender"} {
		t.Run(name, func(t *testing.T) {
			lock, cache, m := variantFixture(t)
			m.RulesVersion = "d06-auto-7"
			m.ReviewedProviders = []string{"bamq/pokemon-sprites"}
			revision := strings.Repeat("a", 40)
			lock.Sources = append(lock.Sources, source{ID: "bamq", Repository: "bamq/pokemon-sprites", Revision: revision, Terms: "fixture", Attribution: "fixture"})
			fixtureInput(t, &lock, cache, "pokesprite-v2", "sources/upstreams/upstream-lock.json", []byte(`{"sources":{"bamq_repo":{"url":"https://github.com/bamq/pokemon-sprites","head":"`+revision+`"}}}`))
			fixtureInput(t, &lock, cache, "bamq", "README.md", []byte("fixture origin"))
			fixtureInput(t, &lock, cache, "bamq", "contributors.md", []byte("fixture credits"))
			a := m.Assets[0]
			a.Provider = "bamq/pokemon-sprites"
			pixels := encodeFixturePNG(t, false)
			fixtureInput(t, &lock, cache, "pokesprite-v2", a.Path, pixels)
			fixtureInput(t, &lock, cache, "bamq", a.Path, pixels)
			switch name {
			case "unreviewed":
				m.ReviewedProviders = nil
			case "wrong revision":
				lock.Sources[len(lock.Sources)-1].Revision = strings.Repeat("b", 40)
			case "missing credits":
				lock.Sources[len(lock.Sources)-1].Files = lock.Sources[len(lock.Sources)-1].Files[:1]
			case "different bytes":
				fixtureInput(t, &lock, cache, "bamq", a.Path, encodeFixturePNG(t, true))
			case "unsupported gender":
				a.Gender = "female"
			}
			_, err := verifyAssetProvider(lock, cache, m, a, "bulbasaur", true)
			if (err == nil) != (name == "verified") {
				t.Fatalf("evidence accepted=%t: %v", err == nil, err)
			}
		})
	}
}
func TestReviewedProviderPolicy(t *testing.T) {
	for _, providers := range [][]string{{"other/provider"}, {"bamq/pokemon-sprites", "bamq/pokemon-sprites"}} {
		if err := validateReviewedProviders(mappingConfig{RulesVersion: "d06-auto-7", ReviewedProviders: providers}); err == nil {
			t.Fatal("accepted invalid provider policy")
		}
	}
	if err := validateReviewedProviders(mappingConfig{RulesVersion: "d06-auto-6", ReviewedProviders: []string{"bamq/pokemon-sprites"}}); err == nil {
		t.Fatal("accepted new provider under old rules")
	}
}
