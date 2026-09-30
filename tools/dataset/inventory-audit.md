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

## Visual-gender batch — explicit inherited layout

Baseline: `5953b43f040dbca86933e168db7d07efa4d26a35`.
Adds Litleo/Pyroar and Espurr/Meowstic with complete evolution-family metadata.
Pyroar and Meowstic each have one standard form with two visual slots, male and
female, and exact regular/shiny palettes. Male is the designated default.

The pinned `sources/upstreams/msikma-pokemon.json` snapshot declares their gen-8
base `has_female` flag. IDs and English slugs agree with normalized ownership.
The corresponding PNGs are fetched from PokéSprite-v2's explicit
`pokemon-gen8/<palette>/female/` paths, not from another repository. Base provider
records and inherited snapshot provenance remain `msikma/pokesprite`.
Pyroar's snapshot marks its sprite as a retained previous-generation icon;
Meowstic's does not. Coverage records this quality detail. Unofficial female
candidates and unofficial base icons are rejected by this layout contract.

Both regular genders differ, and each shiny differs from its exact regular
counterpart after lossless transparent cropping. Visual inspection of all 12 new
sprites showed coherent male/female silhouettes and regular/shiny palettes.
There are 28 species, 40 forms, 84 assets, 4 distinct visual gender slots, and
98 pinned inputs. Standard regular artwork has 30 slots but only 28 species:
random printing still samples species, not asset slots. No storage migration.

This completes the initial explicit-layout gender support, not the entire D06
inventory. Remaining species, alternate layouts, edited candidates, and gender
combinations still require provenance, ownership, and pixel audits.

## Gen1 and complete connected evolution families

Baseline: `562f475bbb05c6934c86811fbbfbb9c50ff38e5f`.
The full original 151-species set is closed over pinned PokéAPI parent/child
edges. Including later relatives and the existing two gen-6 families yields
192 catalog species. This preserves stages, branches, and reference integrity:
Pikachu has Pichu as its predecessor; later evolutions are not silently dropped.

There are 275 metadata forms, 265 collectible forms, and 534 exact assets.
190 species have standard artwork; its 192 slots include the existing two
extra female slots. Random selection remains uniform over 190 species.
The lock has 548 verified inputs. No additional image provider was accepted.
All newly accepted images retain inherited `msikma/pokesprite` provenance but
are downloaded directly from the same pinned PokéSprite-v2 repository.

Excluded appearance identities are recorded individually in mappings/coverage:
Hisuian Growlithe, Arcanine (including Noble), Voltorb, Electrode (including Noble),
Spiky-eared Pichu, Kleavor (standard/Noble), and Annihilape. The inherited
snapshot marks the former appearances provisional; Annihilape has a different
provider requiring its own audit. They remain metadata with unavailable
artwork. Their exclusion does not silently change requested identity or typing.

Noble Hisuian states use their exact owning Hisuian PokéAPI variety for typing;
Spiky-eared Pichu and Noble Kleavor use their documented owning standard variety
where no separate variety exists. These corrections are explicit, and their
provisional/pending artwork is not included.

Farfetch'd/Sirfetch'd source names use straight apostrophes while the pinned
PokéAPI names use curly apostrophes. Exact source-name corrections retain the
canonical PokéAPI display names and generate both spelling aliases. No guessed
punctuation normalization is introduced at runtime.

Every accepted regular/shiny pair has different actual normalized pixels;
regular duplicate checks and source-only alias folding remain enforced. The
450 new sprites were inspected in contact sheets for coherent appearances,
regular/shiny palettes, transparent backgrounds, and cropping. Maximum cropped
dimensions are 52 by 54 source pixels. Real terminal review remains an owner check.

Wider species, remaining visual-gender slots, provisional candidates, and
other inherited providers remain D06 audit work. No trainer state exists and
no schema/history migration is needed. The published v0.1 bundle is unchanged.
