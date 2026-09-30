# D06b — first artwork inventory batch

Baseline: `25c6a0ab35e20bb2116fa1814ccfb7ff54611304`.
Direct source: `darknesspwnsu/pokesprite-v2`, revision
`32ab52ea6b61871da34d9a3c61c7760c65a37af7`.

## Full source inspection

The pinned manifest contains 1,025 species and 1,594 source form records.
126 records have a canonical-form pointer; these are aliases, not additional
collectibles. 13 records are marked generated. Form-level source labels count
1,445 `msikma/pokesprite`, 136 `bamq/pokemon-sprites`, and 13 `generated/pokeapi`.
These are overlapping audit dimensions, not an eligible variant count. They do
not establish image decoding, quality, species ownership, or palette eligibility.

## Included batch

All 16 previously mapped forms of the nine starter-family species now have exact
regular and shiny assets. Source availability, source slug, nongenerated status,
and asset-index provider agree for each palette. The 32 actual PNGs were downloaded
from the pinned PokéSprite-v2 repository, hashed, decoded, cropped without resizing,
and checked for different regular/shiny pixels. All inherited provider records
are `msikma/pokesprite`; this is provenance, not a separate download source.

The source lock now contains 45 inputs (13 metadata/terms inputs plus 32 PNGs).
Repository/revisions and the original three raw image hashes are unchanged.
Normalized dimensions are at most 49 columns by 50 source-pixel rows.
A contact-sheet inspection found coherent regular/shiny pairs, transparent
backgrounds, and distinct mapped Mega/Gigantamax appearances. Real terminal
review remains an owner verification step.

## Remaining work

This is an initial verified batch, not completion of D06 or all 1,025 species.
Wider species/forms, distinct visual gender pairs, additional source providers,
source-generated candidates, and aliases require explicit audits and mappings.
No source-generated or provisional candidate is accepted merely because it has
a filename. Full eligibility and themed tags remain part of the D06/D09 work.

No database or identity migration is needed: trainer storage is not implemented.
The development dataset changes; the published v0.1 bundle and tag remain intact.
