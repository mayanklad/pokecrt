# D06a form and variant normalization

Baseline: b978f71e355a58e69dfcb47cbc3122c5a995d4f0.

This increment normalizes every source form of the current nine species:
16 metadata forms, with independent typing from exact pinned PokéAPI variety
IDs. It adds no artwork or species. Wider inventory work follows in D06b;
public selectors follow in D07.

## Mapping contract

`d06a-1` requires an explicit form record with species owner, canonical ID,
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

The current report has 9 species, 16 metadata forms, 3 regular assets and
collectible forms, 0 shiny assets, 0 distinct visual gender slots, and 13 missing
regular appearances. The source lock still has 16 verified inputs. No new
rights clearance is claimed, and PNGs remain local.

Fixtures test variant normalization without distributing additional source art.
Runtime continues to print the same three standard regular Pokémon.
