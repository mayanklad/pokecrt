package main

import (
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
