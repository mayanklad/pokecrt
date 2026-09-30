# D06a form and variant normalization

Baseline: b978f71e355a58e69dfcb47cbc3122c5a995d4f0.

This increment normalizes every source form of the current nine species:
16 metadata forms, with independent typing from exact pinned PokéAPI variety
IDs. It adds no artwork or species. Wider inventory work follows in D06b;
public selectors follow in D07.

## Mapping contract

`d06a-1`, `d06b-gender-1`, and `d06b-gen1-1` require an explicit form record with species owner, canonical ID,
display name, upstream form ID, PokéAPI variety ID, default visual gender,
declared visual genders, and a reason. Standard maps to the source default and
PokéAPI's default variety. Alternate typing must come from the same species.
Every source form must be mapped or explicitly folded into a source-only alias
or visual gender slot. Source-only aliases must point to the selected canonical
source form and its identical file slug; they create no collectible form.

Asset records identify the exact species/form/gender/palette and pinned source
path. Paths must match the palette and source slug. Manifest availability and
asset-index provenance must agree on the inherited source and nongenerated
status. No implicit regular-to-shiny or alternate-to-standard fallback exists.

A single visual slot is called `default`, regardless of biological gender.
Distinct slots require both `male` and `female` with different actual regular
pixels. The manifest designates the default slot explicitly. Biological gender
flags alone create no variants.

Every shiny slot requires its exact regular counterpart and different pixels.
A source has_shiny flag alone is insufficient. Duplicate regular pixels across
mapped collectibles fail for review; source aliases and orientation duplicates
must be folded rather than counted. Shiny pixels may be shared across distinct
forms when each has its own distinct regular identity; the regular/shiny pair
for each identity must still differ.

## Coverage accounting

- Catalog forms count metadata entries, including unavailable artwork.
- Eligible species count distinct species with at least one regular asset.
- Collectible forms count distinct species/form pairs with regular artwork.
- Standard regular sprites count actual standard regular asset slots.
- Distinct visual gender slots count explicit male/female regular slots.
- Exact eligible variants count validated regular and shiny identities.
- Missing regular appearances report each unavailable form/gender identity.

The previous gender batch established the explicit female layout described below.
The current expanded inventory totals are recorded in the Gen1 section.
`d06b-gender-1` adds an explicit `source_layout: gen8-female` mapping option.
Older mapping rule versions remain readable for their original consolidated paths.

The female layout is restricted to the standard/base female identity and exact
`pokemon-gen8/<palette>/female/<source-slug>.png` path. Require the separately
pinned `sources/upstreams/msikma-pokemon.json` species ID, English slug, gen-8
base form (`$`), and true `has_female`. Reject unofficial female or base icons.
The consolidated base palette/provider audit remains required, in addition to
actual pinned female PNG decoding and distinct gender and palette pixels.
Retained previous-generation flags are preserved in coverage quality notes.
No source identity is inferred from biological gender or arbitrary filenames.

Fixtures test variant normalization without distributing additional source art.
Runtime standard printing now covers 190 eligible species; public alternate selectors
remain D07 work.

## Gen1 family-closure rules

`d06b-gen1-1` requires the pinned inherited snapshot for every included asset,
with matching species ID/slug and exact gen-8 form flags. Reject provisional
`is_unofficial_icon` appearances and unknown inherited forms. Retained
previous-generation flags are recorded in coverage. The inherited provider
palette check and female-layout checks still apply.

The catalog closes complete evolution families; it does not stop at #151 and
lose baby predecessors or later branches. All original 151 species are present.
Current totals are 192 species, 275 metadata forms, 265 collectible forms,
534 exact assets, 190 eligible species, 192 standard regular slots, 267 shinies,
4 distinct visual gender slots, and 548 pinned inputs. Ten excluded appearances
remain catalog metadata with unavailable artwork, including two standard forms.

Name corrections require explicit source/canonical spellings, species owner,
and a reason. The source spelling is retained as a generated alias. Stale,
duplicate, or mismatched corrections fail; no runtime fuzzy-name rule is added.
