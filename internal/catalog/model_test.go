package catalog

import "testing"

func TestExactSpeciesLookupAndIndependentCopies(t *testing.T) {
	species, ok := ByName("  CHARIZARD  ")
	if !ok || species.ID != 6 || species.Stage != 3 || species.Forms[0].Types[0] != "fire" || species.Forms[0].Types[1] != "flying" {
		t.Fatalf("unexpected standard Charizard metadata: %+v", species)
	}
	species.Forms[0].Types[0] = "changed"
	species.Aliases[0] = "changed"
	again, ok := ByNumber(6)
	if !ok || again.Forms[0].Types[0] != "fire" || again.Aliases[0] == "changed" {
		t.Fatal("caller mutation changed bundled catalog")
	}
	if _, ok := ByName("char"); ok {
		t.Fatal("partial name accepted as an exact alias")
	}
	if _, ok := ByNumber(0); ok {
		t.Fatal("unknown National number resolved")
	}
	all := All()
	for i := 1; i < len(all); i++ {
		if all[i-1].ID >= all[i].ID {
			t.Fatal("catalog not sorted uniquely by National number")
		}
	}
}
