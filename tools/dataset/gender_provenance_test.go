package main

import (
	"bytes"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

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
