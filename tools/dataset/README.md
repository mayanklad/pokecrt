# Dataset generation and audit

## Automatic inventory

`mappings.json` contains a development selection policy and reviewed exceptions.
It no longer contains routine `catalog_species`, `forms`, or `assets` lists.
`inventory.go` derives those entries in memory each time generation, checking,
or asset preparation runs. No intermediate maintained inventory file is needed.

The current policy selects generation 1 and the Litleo/Espurr family seeds.
The tool traverses pinned PokéAPI parent/child edges until complete connected
families are included, including later-generation ancestors, descendants, and
branches. The sorted species list therefore comes from data, not a National total.

For every selected source species, canonical forms are read from the pinned
PokéSprite-v2 manifest. Base becomes standard; display labels come from upstream.
Standard typing uses the owning default PokéAPI variety. Other forms require an
exact species-slug/form-ID variety match and checked species ownership.
Source canonical pointers are folded into aliases, never extra collectibles.
Unknown matches fail; the tool does not guess typing from a similar filename.

Four reasoned form exceptions remain: Hisuian Noble Arcanine/Electrode,
Spiky-eared Pichu, and Noble Kleavor share explicitly identified variety typing.
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

The current coverage remains 192 catalog species, 275 metadata forms,
190 printable species, 265 collectible forms, and 534 assets. Ten appearances
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
values authorize exact bytes. It is not an editable species inventory and is not
silently rewritten during normal generation. Scope expansion or source refresh
requires an explicit reviewed lock update for newly required inputs; changing
selection alone does not authorize an unpinned image. Hash discovery/refresh is
a separate development action, not a runtime feature or trust bypass.

The dataset ID includes the policy, resolved inventory, exceptions, and assets.
This refactor changes that ID even though metadata fields and PNG bytes match
the prior inventory. No trainer database exists, so no storage migration applies.

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
Earlier batches reviewed starter forms, species #010-#024, Raticate Totem alias
folding, Pyroar/Meowstic genders, then generation-1 evolution-family closure.
The previous batch inspected all 450 added sprites. This refactor adds no artwork.
The published v0.1 tag and release remain intact; D06 is still incomplete.
