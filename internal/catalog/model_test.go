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

func TestNormalizedFormsKeepTheirOwnTyping(t *testing.T) {
	species, ok := ByNumber(6)
	if !ok || len(species.Forms) != 4 {
		t.Fatalf("Charizard forms: %+v", species.Forms)
	}
	var standard, megaX Form
	for _, form := range species.Forms {
		switch form.ID {
		case "standard":
			standard = form
		case "mega-x":
			megaX = form
		}
	}
	if len(standard.Types) != 2 || standard.Types[1] != "flying" || len(megaX.Types) != 2 || megaX.Types[1] != "dragon" {
		t.Fatalf("standard=%+v mega-x=%+v", standard, megaX)
	}
	if standard.DefaultGender != "default" || len(standard.Genders) != 1 {
		t.Fatalf("fabricated gender identity: %+v", standard)
	}
	count := 0
	for _, species := range All() {
		count += len(species.Forms)
	}
	if count != 585 {
		t.Fatalf("form count=%d; want 585 in the current mapping", count)
	}
}

func TestSourceAliasesDoNotCreateCollectibleForms(t *testing.T) {
	species, ok := ByNumber(20)
	if !ok || len(species.Forms) != 2 {
		t.Fatalf("Raticate forms: %+v", species.Forms)
	}
	var alola Form
	for _, form := range species.Forms {
		if form.ID == "alola" {
			alola = form
		}
	}
	if len(alola.SourceAliases) != 2 || alola.SourceAliases[0] != "totem" || alola.SourceAliases[1] != "totem-alola" {
		t.Fatalf("source aliases: %+v", alola)
	}
	alola.SourceAliases[0] = "changed"
	again, _ := ByNumber(20)
	for _, form := range again.Forms {
		if form.ID == "alola" && form.SourceAliases[0] == "changed" {
			t.Fatal("alias mutation changed catalog")
		}
	}
}

func TestVisualGenderSlotsBelongToStandardForm(t *testing.T) {
	for _, id := range []int{668, 678} {
		species, ok := ByNumber(id)
		if !ok || len(species.Forms) != 1 {
			t.Fatalf("species %d: %+v", id, species)
		}
		f := species.Forms[0]
		if f.ID != "standard" || f.DefaultGender != "male" || len(f.Genders) != 2 || f.Genders[0] != "female" || f.Genders[1] != "male" {
			t.Fatalf("gender slots: %+v", f)
		}
	}
}

func TestGenerationsOneThroughThreeFamiliesAreComplete(t *testing.T) {
	for id := 1; id <= 386; id++ {
		if _, ok := ByNumber(id); !ok {
			t.Fatalf("missing Gen1–Gen3 species %d", id)
		}
	}
	for _, species := range All() {
		if species.EvolvesFrom != 0 {
			if _, ok := ByNumber(species.EvolvesFrom); !ok {
				t.Fatalf("missing parent for %d", species.ID)
			}
		}
		for _, child := range species.EvolvesTo {
			if _, ok := ByNumber(child); !ok {
				t.Fatalf("missing child for %d", species.ID)
			}
		}
	}
	pikachu, _ := ByNumber(25)
	if pikachu.Stage != 2 || pikachu.EvolvesFrom != 172 {
		t.Fatalf("baby predecessor ignored: %+v", pikachu)
	}
}

func TestUnownFormsAndDudunsparceDefault(t *testing.T) {
	unown, ok := ByNumber(201)
	if !ok || len(unown.Forms) != 28 {
		t.Fatalf("Unown forms: %+v", unown.Forms)
	}
	for _, f := range unown.Forms {
		if len(f.Types) != 1 || f.Types[0] != "psychic" {
			t.Fatalf("Unown typing: %+v", f)
		}
	}
	d, ok := ByNumber(982)
	if !ok || len(d.Forms) != 2 || d.Forms[0].ID != "standard" {
		t.Fatalf("Dudunsparce default: %+v", d.Forms)
	}
}

func TestCastformTypesAndDeoxysAlias(t *testing.T) {
	castform, _ := ByNumber(351)
	expected := map[string]string{"standard": "normal", "sunny": "fire", "rainy": "water", "snowy": "ice"}
	if len(castform.Forms) != 4 {
		t.Fatalf("Castform forms: %+v", castform.Forms)
	}
	for _, f := range castform.Forms {
		if len(f.Types) != 1 || f.Types[0] != expected[f.ID] {
			t.Fatalf("Castform typing: %+v", f)
		}
	}
	deoxys, _ := ByNumber(386)
	if len(deoxys.Forms) != 4 || len(deoxys.Forms[0].SourceAliases) != 1 || deoxys.Forms[0].SourceAliases[0] != "normal" {
		t.Fatalf("Deoxys forms/alias: %+v", deoxys.Forms)
	}
	spinda, _ := ByNumber(327)
	if len(spinda.Forms) != 1 {
		t.Fatal("Spinda templates became Pokémon forms")
	}
}
