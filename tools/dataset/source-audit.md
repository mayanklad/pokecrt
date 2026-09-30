# D02a source audit

Application baseline: 7c7c67b18e265ca25315105ccf66c8348904ee33

## Inputs

| Source | Pinned revision | Role |
| --- | --- | --- |
| PokeAPI/pokeapi | bc92d3b6029ef1abe9e7ad424c400b338f3c11fe | CSV metadata and license |
| darknesspwnsu/pokesprite-v2 | 32ab52ea6b61871da34d9a3c61c7760c65a37af7 | Artwork candidates, inventory, provenance, terms and credits |

`sources.json` pins 16 files with full revisions, sizes, and SHA-256 values
measured from actual downloaded bytes. Inputs are cached under
`.cache/dataset/<source-id>/<source-path>` and are excluded from Git.

This increment creates no generated catalog, runtime bundle, final coverage
report, or dataset ID.

## Initial artwork inspection

| Regular candidate | Dimensions | Visible bounds, exclusive right/bottom | Alpha values |
| --- | --- | --- | --- |
| Bulbasaur | 68 × 56 | (24, 34) to (44, 53) | 0, 255 |
| Charizard | 68 × 56 | (12, 15) to (56, 54) | 0, 255 |
| Squirtle | 68 × 56 | (24, 36) to (45, 53) | 0, 255 |

All three decoded successfully. The pinned asset index attributes them to
`msikma/pokesprite` and marks their regular images `is_generated: false`.
That records upstream provenance; it does not establish image licensing or
prove every inherited sprite is unedited game artwork. Terminal review is D03.

## Terms and attribution

PokéAPI's LICENSE.md specifies BSD-3-Clause terms and credits Paul Hallett and
PokéAPI contributors. Preserve the complete notice when bundling derived
metadata. Its notice identifies Pokémon character names as Nintendo trademarks.

PokéSprite v2's README separates image copyright from MIT terms for code and
other material. It identifies image copyrights as Nintendo/Creatures Inc./GAME
FREAK Inc. Its contributors document invites project reuse and identifies
community contributions. That invitation is not a grant from the underlying
image rights holders. Do not label the PNG files MIT-licensed or owned by
PokéCRT. The source-image redistribution gate remains open; this audit claims
no release-rights clearance.

Keep source terms and credits as audit inputs. Prepare THIRD_PARTY_NOTICES.md
before a redistributed bundle.

## Provenance and mapping findings

- Canonical source images can be provisional images generated from PokéAPI
  artwork using Lanczos resizing. Classify and review these explicitly.
- The upstream builder can use regular artwork when shiny artwork is absent.
  A has_shiny flag alone does not establish a genuine shiny appearance.
- Source base must normalize to PokéCRT standard. Do not adopt source aliases
  or orientation duplicates as collectible identities.
- Visual gender inventory requires actual distinct assets; biological gender
  metadata is insufficient.

## Remaining D02 work

D02 is split for separate source and generation review. D02b adds normalization,
mapping records, generated catalog, embedded PNGs/manifest, dataset ID,
deterministic coverage, and generated drift checks. Full form/palette/gender
expansion remains D06.
