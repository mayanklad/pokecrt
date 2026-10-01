# Dataset generation and audit

## Automatic inventory

`mappings.json` contains a development selection policy and reviewed exceptions.
It no longer contains routine `catalog_species`, `forms`, or `assets` lists.
`inventory.go` derives those entries in memory each time generation, checking,
or asset preparation runs. No intermediate maintained inventory file is needed.

The current policy selects generations 1 through 5 and the Litleo/Espurr family seeds.
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

Six reasoned form exceptions remain: Hisuian Noble Arcanine/Electrode/Lilligant,
Noble Kleavor, and Shadow Lugia share explicitly identified variety typing.
The artwork identity Darmanitan `galar` maps to exact owning metadata variety
`darmanitan-galar-standard`; `galar-zen` resolves automatically and stays distinct.
Shadow Lugia artwork remains excluded by the independent quality gate.
Two exact name corrections preserve the Farfetch’d/Sirfetch’d apostrophe aliases.
Unused, redundant, duplicate, and wrong-owner form exceptions fail validation.

## Artwork policy and current audit scope

Routine regular/shiny paths are derived from upstream availability flags.
Only nongenerated `msikma/pokesprite` inherited appearances with verified
identity and nonprovisional gen-8 flags pass this batch's quality gate.
Excluded appearances stay in metadata and are reported automatically.
Asset-index palette/provider provenance and pinned file hashes are also checked.
Decoded PNGs are cropped only at transparent margins; distinct regular/shiny
and gender pixels are required. There are no synthesized substitutes.

Visual-gender policy currently covers only the reviewed Pyroar and Meowstic
pairs. Their female paths come from the inherited gen-8 layout and require
explicit female provenance; male is the default. This scope choice prevents
unaudited female candidates from being added by the refactor. It is not a v1
exclusion. Other gender pairs and providers remain D06 work.

The current coverage is 671 catalog species, 892 metadata forms,
660 printable species, 859 collectible forms, and 1,722 assets. Thirty-three appearances
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
Generation-5 expansion changes that ID and adds 366 exact images. No trainer database exists, so no storage migration applies.

## Package structure

`tools/dataset` keeps source fetching, inventory derivation, normalization,
provenance, generation, and their tests together. Short provenance/name modules
are consolidated into `provenance.go` and `provenance_test.go`.
Static standard selection lives in `internal/catalog/query.go`. The caller
supplies exact sprite availability, preventing a catalog-to-sprite import cycle.
There is no separate `internal/query` package or duplicated selection engine.

## Sources, terms, and history

Direct artwork is downloaded only from `darknesspwnsu/pokesprite-v2`, initially
pinned at `32ab52ea6b61871da34d9a3c61c7760c65a37af7`. Inherited
`msikma/pokesprite` labels describe provenance, not a second download source.
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

Current rules version: `d06-auto-4`. It requires pinned metadata form and form-type tables.
Dudunsparce’s non-base default maps to standard with exact metadata verification;
its generated/unaudited artwork remains unavailable. No new source provider or
visual-gender scope is accepted by this batch.

Maximum cropped artwork dimensions in this batch are 58 columns by 54 source
pixel rows (up to 27 half-block terminal rows). Narrow terminals may wrap.


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
