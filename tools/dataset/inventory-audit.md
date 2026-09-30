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

## Second batch — species #010–#024 and alias audit

Baseline: `d5a51817a79ad894e59ccf98ed075177a2f91a2a`.
Adds Caterpie through Arbok, plus Butterfree Gigantamax, Beedrill/Pidgeot Mega,
and Alolan Rattata/Raticate. Every non-aliased form matches an exact same-species
PokéAPI variety and retains its own typing. Raticate `totem` and `totem-alola`
point to the same `alola` identity/file slug in the pinned manifest; they are
source-only aliases, not two additional collectible forms.

Current totals: 24 species, 36 collectible forms, 36 regular and 36 shiny assets,
85 pinned inputs, and no missing regular appearances within this batch.

### Visual-gender audit finding

The pinned tree has female artwork in other upstream layouts such as
`pokemon-gen8/regular/female/` and `pokemon-gen8/shiny/female/`. For example,
female Pikachu, Pyroar, and Meowstic occur there, while their consolidated
`data/pokemon.json` records have only a base identity. A biological gender flag
or the consolidated base record cannot establish those visual slots.

The current flat consolidated-path mapping contract cannot describe these
alternative-layout images with their own palette-specific provenance. Therefore
no male/female distinction is fabricated in this batch. Explicit source layout,
regular/shiny provenance, ownership, and pixel-difference validation must precede
adding them. This finding is an implementation task, not an exclusion of visual
genders from v1 scope. Wider inventory and this provenance extension remain D06.
