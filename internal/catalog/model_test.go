package catalog

import (
	"slices"
	"strings"
	"testing"
)

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
	if count != 1449 {
		t.Fatalf("form count=%d; want 1449 in the current mapping", count)
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
	for _, id := range []int{449, 450, 521, 592, 593, 668, 678, 876} {
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

func TestGenerationsOneThroughNineFamiliesAreComplete(t *testing.T) {
	for id := 1; id <= 1025; id++ {
		if _, ok := ByNumber(id); !ok {
			t.Fatalf("missing Gen1–Gen9 species %d", id)
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

func TestArceusTypesAndGenerationFourAliases(t *testing.T) {
	arceus, ok := ByNumber(493)
	if !ok || len(arceus.Forms) != 18 {
		t.Fatalf("Arceus forms: %+v", arceus.Forms)
	}
	for _, f := range arceus.Forms {
		want := f.ID
		if f.ID == "standard" {
			want = "normal"
		}
		if len(f.Types) != 1 || f.Types[0] != want {
			t.Fatalf("Arceus typing: %+v", f)
		}
	}
	expected := map[int]map[string][]string{
		413: {"standard": {"bug", "grass"}, "sandy": {"bug", "ground"}, "trash": {"bug", "steel"}},
		479: {"standard": {"electric", "ghost"}, "heat": {"electric", "fire"}, "wash": {"electric", "water"}, "frost": {"electric", "ice"}, "fan": {"electric", "flying"}, "mow": {"electric", "grass"}},
		492: {"standard": {"grass"}, "sky": {"grass", "flying"}},
	}
	for id, forms := range expected {
		s, ok := ByNumber(id)
		if !ok || len(s.Forms) != len(forms) {
			t.Fatalf("forms #%d: %+v", id, s.Forms)
		}
		for _, f := range s.Forms {
			if !slices.Equal(f.Types, forms[f.ID]) {
				t.Fatalf("typing #%d: %+v", id, f)
			}
		}
	}
	for _, tc := range []struct {
		id    int
		alias string
	}{{412, "plant"}, {413, "plant"}, {421, "overcast"}, {422, "west"}, {423, "west"}, {487, "altered"}, {493, "normal"}} {
		s, _ := ByNumber(tc.id)
		if !slices.Contains(s.Forms[0].SourceAliases, tc.alias) {
			t.Fatalf("lost source alias #%d/%s", tc.id, tc.alias)
		}
	}
}

func TestGenerationFiveFormTypingAndAliases(t *testing.T) {
	expected := map[int]map[string][]string{
		555: {"standard": {"fire"}, "zen": {"fire", "psychic"}, "galar": {"ice"}, "galar-zen": {"ice", "fire"}},
		648: {"standard": {"normal", "psychic"}, "pirouette": {"normal", "fighting"}},
		649: {"standard": {"bug", "steel"}, "burn": {"bug", "steel"}, "chill": {"bug", "steel"}, "douse": {"bug", "steel"}, "shock": {"bug", "steel"}},
	}
	for id, forms := range expected {
		species, ok := ByNumber(id)
		if !ok || len(species.Forms) != len(forms) {
			t.Fatalf("forms #%d: %+v", id, species.Forms)
		}
		for _, f := range species.Forms {
			if !slices.Equal(f.Types, forms[f.ID]) {
				t.Fatalf("typing #%d: %+v", id, f)
			}
		}
	}
	for _, tc := range []struct {
		id    int
		alias string
	}{{555, "standard"}, {647, "ordinary"}, {648, "aria"}, {649, "standard"}} {
		species, _ := ByNumber(tc.id)
		if !slices.Contains(species.Forms[0].SourceAliases, tc.alias) {
			t.Fatalf("lost alias #%d/%s", tc.id, tc.alias)
		}
	}
	for _, id := range []int{585, 586} {
		species, _ := ByNumber(id)
		if len(species.Forms) != 4 {
			t.Fatalf("seasonal forms #%d: %+v", id, species.Forms)
		}
	}
}

func TestGenerationSixPatternsTypingAndAliases(t *testing.T) {
	for _, tc := range []struct {
		id, count int
		types     []string
	}{
		{666, 20, []string{"bug", "flying"}}, {669, 5, []string{"fairy"}},
		{670, 6, []string{"fairy"}}, {671, 5, []string{"fairy"}},
		{676, 10, []string{"normal"}}, {718, 3, []string{"dragon", "ground"}},
	} {
		species, ok := ByNumber(tc.id)
		if !ok || len(species.Forms) != tc.count {
			t.Fatalf("forms #%d: %+v", tc.id, species.Forms)
		}
		for _, f := range species.Forms {
			if !slices.Equal(f.Types, tc.types) {
				t.Fatalf("typing #%d: %+v", tc.id, f)
			}
		}
	}
	hoopa, _ := ByNumber(720)
	expected := map[string][]string{"standard": {"psychic", "ghost"}, "unbound": {"psychic", "dark"}}
	for _, f := range hoopa.Forms {
		if !slices.Equal(f.Types, expected[f.ID]) {
			t.Fatalf("Hoopa typing: %+v", f)
		}
	}
	for _, tc := range []struct {
		id    int
		count int
		alias string
	}{
		{664, 1, "meadow"}, {665, 1, "meadow"}, {666, 20, "meadow"},
		{681, 2, "shield"}, {711, 1, "super"}, {716, 2, "neutral"}, {718, 3, "50"},
	} {
		species, _ := ByNumber(tc.id)
		if len(species.Forms) != tc.count || !slices.Contains(species.Forms[0].SourceAliases, tc.alias) {
			t.Fatalf("aliases #%d: %+v", tc.id, species.Forms)
		}
	}
	greninja, _ := ByNumber(658)
	if len(greninja.Forms) != 2 || greninja.Forms[1].ID != "ash" || !slices.Contains(greninja.Forms[1].SourceAliases, "battle-bond") {
		t.Fatalf("Greninja forms: %+v", greninja.Forms)
	}
}

func TestGenerationSevenTypesAndSourceAliases(t *testing.T) {
	silvally, _ := ByNumber(773)
	if len(silvally.Forms) != 18 {
		t.Fatalf("Silvally forms: %+v", silvally.Forms)
	}
	for _, f := range silvally.Forms {
		typ := f.ID
		if typ == "standard" {
			typ = "normal"
		}
		if !slices.Equal(f.Types, []string{typ}) {
			t.Fatalf("Silvally typing: %+v", f)
		}
	}
	expected := map[int]map[string][]string{
		741: {"standard": {"fire", "flying"}, "pau": {"psychic", "flying"}, "pom-pom": {"electric", "flying"}, "sensu": {"ghost", "flying"}},
		800: {"standard": {"psychic"}, "dawn": {"psychic", "ghost"}, "dusk": {"psychic", "steel"}, "ultra": {"psychic", "dragon"}},
	}
	for id, forms := range expected {
		species, _ := ByNumber(id)
		if len(species.Forms) != len(forms) {
			t.Fatalf("forms #%d: %+v", id, species.Forms)
		}
		for _, f := range species.Forms {
			if !slices.Equal(f.Types, forms[f.ID]) {
				t.Fatalf("typing #%d: %+v", id, f)
			}
		}
	}
	minior, _ := ByNumber(774)
	if len(minior.Forms) != 8 {
		t.Fatalf("historical Minior artwork became forms: %+v", minior.Forms)
	}
	for _, f := range minior.Forms {
		if !slices.Equal(f.Types, []string{"rock", "flying"}) {
			t.Fatalf("Minior typing: %+v", f)
		}
	}
	marshadow, _ := ByNumber(802)
	if len(marshadow.Forms) != 1 {
		t.Fatal("historical Marshadow artwork became a form")
	}
	for _, tc := range []struct {
		id    int
		alias string
	}{{735, "totem"}, {744, "own-tempo"}, {745, "midday"}, {746, "solo"}, {778, "busted"}} {
		species, _ := ByNumber(tc.id)
		if !slices.Contains(species.Forms[0].SourceAliases, tc.alias) {
			t.Fatalf("lost alias #%d/%s", tc.id, tc.alias)
		}
	}
}

func TestGenerationEightTypingAndAlcremieIdentities(t *testing.T) {
	expected := map[int]map[string][]string{
		888: {"standard": {"fairy"}, "crowned": {"fairy", "steel"}},
		889: {"standard": {"fighting"}, "crowned": {"fighting", "steel"}},
		892: {"standard": {"fighting", "dark"}, "gmax": {"fighting", "dark"}, "rapid-strike-gmax": {"fighting", "water"}},
		898: {"standard": {"psychic", "grass"}, "ice-rider": {"psychic", "ice"}, "shadow-rider": {"psychic", "ghost"}},
	}
	for id, forms := range expected {
		species, _ := ByNumber(id)
		if len(species.Forms) != len(forms) {
			t.Fatalf("forms #%d: %+v", id, species.Forms)
		}
		for _, f := range species.Forms {
			if !slices.Equal(f.Types, forms[f.ID]) {
				t.Fatalf("typing #%d: %+v", id, f)
			}
		}
	}
	alcremie, _ := ByNumber(869)
	if len(alcremie.Forms) != 64 || !slices.Contains(alcremie.Forms[0].SourceAliases, "vanilla-cream-strawberry") {
		t.Fatalf("Alcremie identities: %+v", alcremie.Forms)
	}
	for _, f := range alcremie.Forms {
		if !slices.Equal(f.Types, []string{"fairy"}) || strings.HasSuffix(f.ID, "-plain") {
			t.Fatalf("Alcremie typing/template: %+v", f)
		}
	}
}

func TestGenerationNineDefaultsAndTypes(t *testing.T) {
	expected := map[int]map[string][]string{
		925:  {"standard": {"normal"}, "family-of-three": {"normal"}},
		931:  {"standard": {"normal", "flying"}, "blue-plumage": {"normal", "flying"}, "white-plumage": {"normal", "flying"}, "yellow-plumage": {"normal", "flying"}},
		964:  {"standard": {"water"}, "hero": {"water"}},
		1017: {"standard": {"grass"}, "cornerstone-mask": {"grass", "rock"}, "hearthflame-mask": {"grass", "fire"}, "wellspring-mask": {"grass", "water"}},
		1024: {"standard": {"normal"}, "terastal": {"normal"}, "stellar": {"normal"}},
	}
	for id, forms := range expected {
		species, ok := ByNumber(id)
		if !ok || species.Generation != 9 || len(species.Forms) != len(forms) {
			t.Fatalf("forms #%d: %+v", id, species.Forms)
		}
		for _, f := range species.Forms {
			if !slices.Equal(f.Types, forms[f.ID]) {
				t.Fatalf("typing #%d: %+v", id, f)
			}
		}
	}
	for _, tc := range []struct {
		id    int
		types []string
		alias string
	}{{1007, []string{"fighting", "dragon"}, "limited-build"}, {1008, []string{"electric", "dragon"}, "low-power-mode"}} {
		species, _ := ByNumber(tc.id)
		if len(species.Forms) != 1 || !slices.Equal(species.Forms[0].Types, tc.types) || !slices.Contains(species.Forms[0].SourceAliases, tc.alias) {
			t.Fatalf("source mode #%d: %+v", tc.id, species.Forms)
		}
	}
}

func TestTatsugiriTransformationsAreDistinctMetadata(t *testing.T) {
	species, ok := ByNumber(978)
	if !ok || len(species.Forms) != 6 {
		t.Fatalf("Tatsugiri forms: %+v", species.Forms)
	}
	seen := map[string]bool{}
	for _, f := range species.Forms {
		seen[f.ID] = true
		for _, alias := range f.SourceAliases {
			if alias == "curly-mega" || alias == "droopy-mega" {
				t.Fatal("transformation folded into ordinary artwork alias")
			}
		}
		if !slices.Equal(f.Types, []string{"dragon", "water"}) && !slices.Equal(f.Types, []string{"water", "dragon"}) {
			t.Fatalf("exact form typing: %+v", f)
		}
	}
	for _, id := range []string{"standard", "curly-mega", "droopy", "droopy-mega", "stretchy", "stretchy-mega"} {
		if !seen[id] {
			t.Fatalf("missing %s", id)
		}
	}
}
