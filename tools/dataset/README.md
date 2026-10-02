# Dataset generation and audit

## Automatic inventory

`mappings.json` contains a development selection policy and reviewed exceptions.
It no longer contains routine `catalog_species`, `forms`, or `assets` lists.
`inventory.go` derives those entries in memory each time generation, checking,
or asset preparation runs. No intermediate maintained inventory file is needed.

The current policy selects generations 1 through 9. The earlier Litleo/Espurr
family seeds are now redundant and removed; generation selection includes them.
The tool traverses pinned PokéAPI parent/child edges until complete connected
families are included, including later-generation ancestors, descendants, and
branches. The sorted species list therefore comes from data, not a National total.

For every selected source species, canonical forms are read from the pinned
PokéSprite-v2 manifest. The source-designated default becomes standard; display labels come from upstream.
A non-base source default must match the exact owning PokéAPI default variety.
Standard typing uses the owning default PokéAPI variety. Other forms require an
exact species-slug/form-ID match against pinned varieties or `pokemon_forms.csv`,
with checked species ownership. Unown’s 28 appearances resolve from that table;
its standard artwork is A. Pichu’s Spiky-eared identity also resolves there.
Source canonical pointers are folded into aliases, never extra collectibles.
Unknown matches fail; the tool does not guess typing from a similar filename.

Eleven reasoned form exceptions remain: Hisuian Noble Arcanine/Electrode/Lilligant/Avalugg,
Noble Kleavor, and Shadow Lugia share explicitly identified variety typing.
The artwork identity Darmanitan `galar` maps to exact owning metadata variety
`darmanitan-galar-standard`; `galar-zen` resolves automatically and stays distinct.
Toxtricity/Urshifu Gigantamax and Calyrex rider spellings have four checked
style-specific variety corrections.
Shadow Lugia artwork remains excluded by the independent quality gate.
Two exact name corrections preserve the Farfetch’d/Sirfetch’d apostrophe aliases.
Unused, redundant, duplicate, and wrong-owner form exceptions fail validation.

## Artwork policy and current audit scope

Routine regular/shiny paths are derived from upstream availability flags.
Nongenerated `msikma/pokesprite` inherited appearances require verified identity
and nonprovisional gen-8 flags. Explicitly reviewed `bamq/pokemon-sprites`
appearances require the pinned import commit, preserved credits, and byte-identical
PNGs from the original provider. Unknown providers and generated candidates fail
the acceptance policy.
Excluded appearances stay in metadata and are reported automatically.
Asset-index palette/provider provenance and pinned file hashes are also checked.
Decoded PNGs are cropped only at transparent margins; distinct regular/shiny
and gender pixels are required. There are no synthesized substitutes.

Visual-gender policy derives standard pairs automatically from the pinned
inherited inventory. Require explicit `has_female`, exact species identity,
nongenerated inherited base provenance, nonprovisional base/female flags, and
separate locked gen-8 female paths. Both regular genders must have distinct pixels.
Biological gender flags alone never create artwork slots. Male is the default.
The eight accepted pairs are Hippopotas, Hippowdon, Unfezant, Frillish, Jellicent,
Pyroar, Meowstic, and Indeedee. The acceptance logic maintains no species list.
Provisional female candidates stay excluded; other providers/layouts and
source-encoded layouts beyond the reviewed male-default case remain D06 review work.

The current coverage is 1,025 catalog species, 1,448 metadata forms,
1,013 standard-printable species, 1,017 eligible encounter species, 1,327
collectible forms, and 2,669 assets. 122 appearances
remain unavailable. `coverage.json` and `coverage.md` enumerate the current
availability, exclusion decisions, and retained-generation quality flags.

## Reproduce and verify

From the project root:

```bash
go run ./tools/dataset \
  --sources tools/dataset/sources.json \
  --cache .cache/dataset \
  --mappings tools/dataset/mappings.json \
  --out . --fetch --generate --check

go test ./...
go vet ./...
```

`--generate --check` is offline with a verified cache. `--check` reports drift
without overwriting committed outputs. `--prepare-assets --check` validates
committed metadata and materializes ignored PNGs for a fresh checkout.
The installed executable embeds those PNGs and never downloads at runtime.

`sources.json` remains an explicit integrity lock: revisions, sizes, and SHA-256
values authorize exact bytes. Normal generation never rewrites it or accepts
unpinned images. For future reviewed artwork scope expansion at the existing
revision, the developer can explicitly discover missing image pins through code:

```bash
go run ./tools/dataset \
  --sources tools/dataset/sources.json \
  --cache .cache/dataset \
  --mappings tools/dataset/mappings.json \
  --out . --update-lock --generate --check
```

This action requires verified metadata already in the cache. It derives paths
from the quality-gated inventory, preserves all existing pins, downloads missing
images from the unchanged revision, decodes them, and measures raw sizes/hashes.
It validates the entire proposed inventory before atomically saving the lock.
Eight bounded download workers collect results; sorted records keep output
stable. Review lock/coverage diffs and images before committing. Failed downloads
or inventory validation do not save the proposed lock. Metadata source additions
and revision changes still require separate reviewed pin updates.
`--update-lock` cannot be used with release asset preparation and requires an
explicit `--generate`; it is not a runtime or ordinary build action.

The dataset ID includes the policy, resolved inventory, exceptions, and assets.
The reviewed provider expansion changes that ID and adds exact artwork pins. No trainer database exists, so no storage migration applies.

## Package structure

`tools/dataset` keeps source fetching, inventory derivation, normalization,
provenance, generation, and their tests together. Short provenance/name modules
are consolidated into `provenance.go` and `provenance_test.go`.
Static standard selection lives in `internal/catalog/query.go`. The caller
supplies exact sprite availability, preventing a catalog-to-sprite import cycle.
There is no separate `internal/query` package or duplicated selection engine.

## Sources, terms, and history

Runtime artwork is prepared from `darknesspwnsu/pokesprite-v2`, initially
pinned at `32ab52ea6b61871da34d9a3c61c7760c65a37af7`. Inherited
`msikma/pokesprite` labels describe provenance. Reviewed community imports are additionally checked against the
original pinned `bamq/pokemon-sprites` repository.
PokéAPI metadata is pinned at `bc92d3b6029ef1abe9e7ad424c400b338f3c11fe`.
The cache label is `pokesprite-v2`; raw inputs and normalized PNGs stay outside Git.

PokéAPI metadata retains BSD-3-Clause terms and attribution. PokéSprite-v2
separates artwork copyright from its code/non-image MIT terms. Community reuse
statements do not establish permission from underlying rights holders.
The owner-approved distribution policy is in `docs/release-policy.md`;
`LICENSING.md` and generated `THIRD_PARTY_NOTICES.md` preserve licensing scope
and notices. No rights-holder endorsement or clearance is claimed.

The pinned manifest contains 1,025 species and 1,594 source form records;
126 records have canonical pointers and 13 are source-generated candidates.
Those source totals are audit observations, never completion denominators.
Earlier batches reviewed starter forms, species #010–#024, Raticate Totem alias
folding, Pyroar/Meowstic genders, then generation-1 evolution-family closure.
The previous generation-1 batch inspected all 450 added sprites. Generation 2 added 252 sprites; the generation-3 batch inspects 344 new sprites
in 172 regular/shiny pairs.
The published v0.1 tag and release remain intact; D06 is still incomplete.

Current rules version: `d06-auto-10`. It requires pinned metadata form and form-type tables.
Dudunsparce’s non-base default maps to standard with exact metadata verification;
its generated standard artwork remains unavailable. The community provider is
reviewed below; the automatic inherited-gender audit is documented below.

Maximum cropped artwork dimensions in this batch are 67 columns by 56 source
pixel rows (up to 28 half-block terminal rows). Narrow terminals may wrap.


## Excluded source templates

`source_form_exclusions` records Spinda's `blank` and `filled` pattern templates.
These identities have no exact PokéAPI variety/form match and are marked
unofficial in the pinned inherited inventory. They are retained in coverage
exclusion reasons, not promoted to Pokémon forms or folded into false aliases.
A manual template exclusion requires a reason and matching species/slug/appearance evidence.
A manual template exclusion cannot hide a standard identity, an exact metadata form, or official artwork.
Duplicate, unused, aliased, or already claimed exclusions fail validation.

Castform standard/sunny/rainy/snowy types are Normal/Fire/Water/Ice from exact
metadata varieties. Deoxys `normal` is a source alias of standard; attack,
defense, and speed remain separate forms. Orientation files are not added.
No new source provider or visual-gender audit scope is introduced by this batch.


## Transfer progress

The dataset tool reports input download/cache-verification progress to stderr by
default. Interactive character-device stderr gets a single-line bar and a spinner
refreshed every 250 ms. Redirected stderr (or `TERM=dumb`) gets plain-text start,
periodic, and completion/failure lines without carriage returns or ANSI escapes.
Periodic log updates occur every five seconds or 250 completed inputs. Counts
separate downloaded files from successfully verified cache hits; corrupt cache
entries and failed downloads never count as completed.

`--verbose` adds one line per successfully downloaded or verified cached input.
Failures retain the affected filename in the error. Progress for the eight
artwork-pin download workers is serialized, and the heartbeat stops before the
phase returns. Counters are per phase, including post-update lock verification.
The final dataset summary remains on stdout. Progress is presentation only:
source locks, generation, coverage, dataset identity, and offline runtime are
unchanged. No dependency, package, or source file was added for this feature.


## Exact form-specific types

`pokemon_form_types.csv` supplies type overrides attached to exact metadata form
IDs, not filenames or inferred type words. `pokemon_forms.csv` establishes the
owning variety, and `pokemon.csv` establishes species ownership. When a form has
explicit type rows, use them instead of its variety's types; otherwise retain
its verified variety typing. Validate positive identities, ownership, known type
IDs, slot cardinality, and unique slots. Supported v1 types are the 18 standard
types. This derives all Arceus type forms automatically, and protects the
Wormadam, Rotom, and Shaymin form-specific types.

Arceus `unknown` has an exact metadata form but an unsupported type. The tool
automatically reports that source identity as excluded without assigning Normal
typing or creating an eligible asset. Such exclusions carry exact metadata form
and unsupported-type evidence; users cannot invent that evidence in mappings.
These differ from the manually reviewed Spinda source-template exclusions.

The generation-4 batch adds 78 catalog species through full family closure and
226 sprites in 113 regular/shiny pairs. Dialga/Palkia Origin are metadata-only
pending inherited source-quality review. All earlier PNGs are unchanged. Source
providers and reviewed visual-gender scope remain unchanged; D06 stays open.


## Generation 5 increment

The generation-5 policy adds 159 catalog species through connected family closure,
192 metadata forms, and 366 sprites in 183 regular/shiny pairs. All National
numbers 1–649 have audited standard artwork. Later relatives Basculegion and
Kingambit remain metadata-only under the unchanged source-quality gate. New
Hisuian appearances remain unavailable where inherited quality is provisional.
Darmanitan retains Fire, Fire/Psychic, Ice, and Ice/Fire typing by exact variety.
Meloetta Pirouette is Normal/Fighting. Genesect drives remain Bug/Steel; drive
names do not determine Pokémon types. Deerling/Sawsbuck seasons, Kyurem forms,
and source canonical aliases derive from pinned records without maintained lists.
All 366 new normalized sprites were visually inspected; all earlier PNG bytes
are unchanged. Source revisions, providers, generator code, and visual-gender
policy are unchanged. Rules remain `d06-auto-4` because no derivation logic changed.
D06 remains open for later generations and provider/gender/tag audits.


## Generation 6 increment

Selecting generation 6 adds 67 catalog species, 122 metadata forms, and 230
sprites in 115 regular/shiny pairs. The earlier four Pyroar/Meowstic family
species and Sylveon were already included, so they are not added twice. All
National numbers 1–721 have audited standard artwork. The existing reviewed
visual-gender pairs retain their identities and bytes.

Vivillon's 20 patterns, flower colors, Furfrou trims, Zygarde's three forms,
and Hoopa Unbound derive from exact metadata records. Hoopa standard is
Psychic/Ghost; Unbound is Psychic/Dark. Greninja battle-bond is a source alias
of Ash artwork. Scatterbug/Spewpa pattern names and Gourgeist size names are
source aliases of the same appearance, not additional collectible sprites.
Pumpkaboo's three alternate sizes remain metadata-only because the inherited
quality flags mark them as unofficial. Hisuian Sliggoo/Goodra/Avalugg and Noble Avalugg
also remain unavailable under the existing quality gate. Noble Avalugg shares
verified Hisuian variety typing through one reasoned exception.

All 230 new normalized sprites were visually inspected and all earlier PNGs
are unchanged. Source revisions, providers, generator code, and visual-gender
scope are unchanged. Rules remain `d06-auto-4`. No source files, packages,
dependencies, or public flags were added. D06 remains open for generations
7–9 and remaining provider, gender, and achievement-tag audits.


## Generation 7 increment

Generation 7 adds 88 catalog species, 124 metadata forms, and 211 eligible
images. All National numbers 1–809 have audited standard regular artwork.
Silvally's 18 types derive from exact metadata varieties; 17 alternate-type
artworks remain unavailable under the inherited quality gate. Hisuian Decidueye
also remains metadata-only. Oricorio styles, Necrozma fusions, Lycanroc forms,
Wishiwashi School, Magearna Original, and Melmetal Gigantamax resolve from pinned
identities. Totem, Rockruff own-tempo, and Mimikyu source aliases do not create
extra collectible appearances.

Eight reviewed `source_form_exclusions` cover seven Minior `*-gen7` records
and Marshadow `gen7`. These are historical artwork versions marked unofficial,
with no separate metadata identity. They are retained as coverage exclusions;
Minior's seven core colors and standard Marshadow remain supported. This is
exception evidence, not a manually maintained catalog or sprite list.

`duplicate_palette_exclusions` records the single reviewed Minior meteor case.
Both raw palettes remain pinned, provenance-validated, decoded and normalized.
The exception requires equal normalized SHA-256 values for the exact same
species/form/gender, removes only shiny artwork from the manifest/files, and
reports hash evidence in coverage. Duplicate, empty, unused, missing-palette,
and unequal-pixel exceptions fail. Without an exception, equal palettes still
fail the existing validator. Core Minior palettes remain verified independently.
There are 2,180 locked inputs: 2,163 eligible PNGs, one excluded duplicate PNG,
and 16 metadata/terms inputs. No synthetic palette or runtime fallback is added.

All 211 new eligible images were visually inspected; all 1,952 earlier PNGs
are unchanged. Generator changes stay in existing files. Source revisions,
provider scope and visual genders are unchanged. Rules version `d06-auto-5`
records the narrow reviewed duplicate-palette policy. Tests cover generation
completeness, aliases, exact typing, selection boundaries, and exclusion failure
cases. D06 remains open for generations 8–9 and provider/gender/tag audits.


## Generation 8 increment

Generation 8 adds 87 catalog species through family closure, 179 metadata forms,
and 240 sprites in 120 regular/shiny pairs. Earlier family members were already
included, so selection does not add them twice. All National numbers 1–898 have
standard regular artwork. Hisuian species 899–905 and new later relatives
Dipplin/Archaludon/Hydrapple remain metadata-only under the unchanged quality gate.
There are 117 unavailable metadata appearances and 15 unavailable standards.

Alcremie normalization uses one reviewed `form_suffix_rules` entry: append
`-sweet` only when an exact same-species metadata form record exists. Require
metadata form IDs and ownership; reject duplicate, unused or wrong-owner rules.
Unmatched source identities remain errors or independently reviewed exclusions.
This derives the supported combinations without maintaining dozens of mappings.
Nine decoration-free `*-plain` templates are unofficial source-only exclusions.
The 64 normalized Alcremie identities comprise standard, 62 other sweet/cream
combinations and Gigantamax. Most sweet artwork remains quality-gated unavailable.

`reviewed_default_aliases` folds vanilla-cream-strawberry into standard only
after its exact metadata form ID matches the owning default. The regular
source files differ in outline treatment; this correction is an explicit artwork
version decision, not an equality claim. Both source palettes remain locked;
verify manifest availability, nongenerated inherited provenance and source asset
index, then match all four reviewed normalized hashes (source/default regular
and shiny). Preserve source-designated standard artwork and report the correction
and hash evidence in coverage. Missing or changed pins, incomplete/wrong hashes,
invalid ownership, redundant/unused aliases and unaudited artwork fail. Normal
checks never fetch or refresh this evidence silently. Explicit lock expansion
adds the two evidence inputs through the same bounded, validated download path.

Calyrex standard/Ice Rider/Shadow Rider retain Psychic/Grass, Psychic/Ice and
Psychic/Ghost types. Crowned Zacian and Zamazenta retain their Steel second type.
Urshifu Rapid Strike Gigantamax remains Fighting/Water; its standard and default
Gigantamax are Fighting/Dark. These types use owning metadata varieties.
Four reasoned spelling exceptions resolve the source's shorter/longer names.

All 240 new assets were visually inspected and earlier PNG bytes are unchanged.
Rules `d06-auto-6` record suffix/default-alias normalization. 2,422 pinned inputs
include 2,403 eligible images, two default-alias evidence images, the existing
excluded Minior shiny image, and 16 metadata/terms files. Tests, vet, build and
reproducibility checks pass. No source files, packages, dependencies or public
flags added. D06 remains open for generation 9 and provider/gender/tag audits.


## Generation 9 metadata increment

Selecting generation 9 adds 112 catalog species and 130 metadata forms through
pinned-source derivation, without new exceptions or generator changes. The
catalog now spans all nine source generations: 1,025 species and 1,447 metadata
forms. This is species metadata coverage, not completion of D06 or full artwork
coverage. Generation 9 source appearances use generated candidates or
`bamq/pokemon-sprites`, outside the currently reviewed inherited-provider scope.
They remain metadata-only. No artwork source pins or PNG bytes change.

Eligible artwork remains 898 species, 1,200 collectible forms and 2,403 assets.
There are 127 unavailable standard appearances and 247 unavailable metadata
form appearances. Named generation-9 printing fails with artwork unavailable;
there is no substitute, generated fallback, or implicit runtime download.
Coverage distinguishes each excluded provider/generated source record.

Ogerpon masks use exact Grass/Rock, Grass/Fire, and Grass/Water types; standard
is Grass. Terapagos forms remain Normal according to their verified metadata;
`stellar` is not inferred as a type from a form name. Maushold/Squawkabilly/Palafin
non-base defaults match their owning default varieties. Koraidon/Miraidon source
mode aliases do not produce extra artwork collectibles. Tests verify these
metadata cases, complete species/family references, and missing-artwork failures.

Rules remain `d06-auto-6`; no new file, package, dependency, correction policy,
or runtime command is introduced. D06 next audits the additional provider,
remaining provisional artwork, visual-gender identities (including Oinkologne),
source-generated form/alias claims, and achievement tags. Generated appearance
claims are not evidence of usable artwork or completed form/gender semantics.


## Reviewed Generation 9 provider increment

Generation 9 metadata was owner-committed at
39abfc30e855ac8d0aeddf3a71667b7eb3e0e63d. Rules `d06-auto-7` explicitly accept
nongenerated `bamq/pokemon-sprites` appearances without maintaining a species or
image list. Paths and identities continue to derive from pinned source records.
The provider revision c1958e7260a4bce93bd104e79966a27fb333c7aa must match the
pinned v2 upstream lock. Reviewed importer/build scripts are pinned as evidence:
they copy PNGs without reencoding. Every accepted community PNG is locked in both
repositories and must match byte-for-byte. Missing or mismatched provider evidence
fails generation. The original provider README and contributor file are pinned
and included in generated notices, retaining project and artist-credit links.
These are community-adapted icons, resized upstream to 68×56; PokéCRT does not
resize them or claim they are untouched game assets. The existing owner-approved
fan-project release policy applies; the audit does not claim image rights clearance.

This adds 254 assets in 127 regular/shiny pairs across 119 species. Standard
printing gains 115 species, bringing its pool to 1,013 species. Four species have
accepted alternate artwork but no accepted standard, so encounter eligibility
across all forms is 1,017. Twelve standards remain unavailable: Hisuian species
899–905, Oinkologne, Palafin, Tatsugiri, Dudunsparce, and Ogerpon. Their exact
missing appearances stay reported; alternate artwork never replaces standard.
Generated candidates remain excluded. Oinkologne gender semantics and suspicious
source-only alias claims remain explicit D06 review work before public selectors.

All 254 new normalized images were visually inspected; all earlier 2,403 PNGs
remain byte-identical. Tests cover provider policy, exact import revision,
missing credits, byte mismatch, unsupported gender/layout, standard selection,
missing artwork and installed Generation 9 printing. Source locks total 2,935
inputs: 2,657 accepted v2 images, three existing excluded/alias evidence images,
254 original-provider images, and 21 metadata/terms/import-evidence files.
No additional source file, package, dependency or public flag is introduced.
The maximum cropped dimensions remain 67×56. D06 remains open.


## Automatic inherited visual genders increment

Provider increment owner-committed at 7a901ba813188b7d33a1d155b427bbdf12c1ab6f.
Rules `d06-auto-8` replace maintained `visual_gender_species` with the boolean
`audited_inherited_genders` policy. Source declarations derive the accepted pairs;
unsupported/provisional candidates are reported automatically. Tests reject wrong
owners/slugs and mixed automatic/manual policies; no biological-only inference.

Six newly reviewed pairs add 12 female PNGs. Their existing standard `default`
identities become `male`, preserving all image bytes and default print output.
They remain one collectible standard form per species. There is no trainer schema
or persisted discovery data yet, so no storage migration is applicable. The
published v0.1 starter identities are unchanged. This refinement changes dataset
identity; it is an explicit reviewed dataset change, not a byte-only refactor.

Before regeneration in an existing checkout, remove only these obsolete ignored
asset names (generation rejects unexpected images instead of deleting them):

```bash
rm -f internal/sprite/assets/{0449,0450,0521,0592,0593,0876}-standard-default-{regular,shiny}.png
```

All 12 added images and the complete 24-image new-pair comparison were visually
reviewed. All 2,657 preceding image contents are unchanged, with 12 male filenames
refined. Totals: 2,669 exact assets, 1,327 collectible forms, 16 distinct gender
slots, 1,021 standard regular slots across 1,013 printable species, and 2,947
locked inputs. Missing standards/appearances remain 12/120. Source revisions,
providers, maximum dimensions, dependencies and public commands are unchanged.
No additional source files or packages. D06 remains open for remaining gender/form
identity, provisional-source and achievement-tag review. D07 will use FlagSet
and shared validation for the public selectors.


## Transformation alias semantics increment

Automatic gender increment owner-committed at
17558ae1a53b8288b870b4103d7b198ff735f114. Rules `d06-auto-9` require the pinned
metadata form table's explicit `is_mega` flags when auditing source aliases.
A Mega-marked exact metadata form owned by a different same-species variety
cannot be folded into ordinary artwork merely because the source uses a canonical
pointer and the same filename. Retain that form with its own checked metadata
form/variety IDs and typing, report the source discrepancy, and accept no asset
for that metadata-only identity. This is derived automatically; there is no
maintained Tatsugiri list or manually assigned type. Missing targets, wrong
owners, mismatched source slugs, invalid metadata flags and borrowed asset
claims fail validation. Ordinary aliases retain their existing folding behavior.

The current pinned inputs contain two such cases: Tatsugiri Curly Mega and
Droopy Mega. Stretchy Mega was already separate metadata with excluded generated
artwork. All six source Tatsugiri forms now have distinct metadata representation;
the two ordinary aliases are removed. This adds two metadata forms and two
reported unavailable appearances, with no new source pins, sprites, collectible
forms, or printing candidates. Totals: 1,448 metadata forms, 122 unavailable
appearances; other availability counts remain unchanged. All 2,669 PNGs and all
2,947 source pins are unchanged. No trainer data exists and no stored identity
migration applies; previously supported artwork identities are preserved.

Tests cover exact metadata transformation ownership/typing, ordinary alias
preservation, malformed evidence, forbidden borrowed artwork and absence of
regular/shiny fallback in runtime lookup. Full tests/vet/race, build, deterministic
generation and fresh preparation pass. No new source files, packages, dependency,
or public flags added. Oinkologne source-encoded gender identity, other alias and
provisional-source audits, and achievement tags remain D06 work. Parser conversion
and public selectors remain D07.


## Source-encoded gender identity increment

Transformation-alias increment owner-committed at
795183b41aac94d5fbe0e1cd083ca9d24713427c. Rules `d06-auto-10` derive the
source’s explicit male-default, male/female layout as one standard form. Require
pinned gender-difference metadata, exact male/female `form_identifier` values,
positive form IDs, same-species variety ownership, matching default variety,
non-aliased source records, and identical exact metadata typing. The resolved
source-gender declarations are computed; no species exception list is maintained.

Oinkologne is the current pinned case. Its two Normal records become one standard
form with male/female slots and a male default. Both candidates are generated and
remain unavailable under the existing artwork policy. Metadata-only gender pairs
may have neither regular asset; partial pairs, identical regular pixels and
shiny-only slots still fail validation. Available pairs still require distinct
regular pixels. Asset claims must match their declared source gender.

Totals: 1,448 metadata forms, 122 unavailable appearances, 2,669 accepted assets,
16 available gender slots across eight species, and 2,947 unchanged source pins.
All accepted artwork bytes are unchanged. No files/packages/public flags are
added; no ignored filenames need removal. No trainer data exists to migrate.
Tests cover ownership, explicit gender identifiers, missing/aliased records,
typing, unavailable-pair validation, swapped asset claims and runtime absence.
Earlier increment descriptions above record their historical review scope;
Oinkologne identity is now resolved, while its artwork remains unavailable.
D06 remains open for other provider/layout, provisional, alias and tag audits.
