# Implementation progress

Specification: v1, including approved sections 25–27 and implementation clarifications
Current release target: v0.2 — Complete public engine
Last published milestone: v0.1
Current increment: D06 generation-4 inventory and exact form typing
Current source commit: c4d8712bbd3183262060d043ff2c8d464ddb66d5
Commit note: verified baseline before this increment; owner verification/commit pending.
Published release: https://github.com/mayanklad/pokecrt/releases/tag/v0.1

## Runtime

- Root/help/version; named or uniformly random standard regular printing
- Compact/sprite output, natural-size truecolor half blocks, transparency
- NO_COLOR, piped colors, quiet broken pipes; no trainer state or runtime downloads
- No new public flags; explicit form/shiny/gender selectors remain D07

## Dataset

Development dataset ID:
`0d5db8c1c2ee02ebb7f2a86e7a9bd9a0e298b922a51796c7d3ff126787e24a8a`

- 512 catalog species: all generation-1 through generation-4 species plus full evolution-family closure and existing gen-6 families
- 700 metadata forms; 676 collectible forms; 1,356 exact regular/shiny assets
- 503 eligible species; 505 standard regular slots; 678 shiny slots
- 4 distinct visual gender slots across Pyroar/Meowstic, with male defaults
- 24 unavailable metadata form appearances; nine species lack audited standard artwork
- 1,372 pinned inputs; coverage.json/coverage.md record availability and quality
- Direct images remain pinned PokéSprite-v2; inherited provider records stay distinct
- Complete evolution references, aliases, typing, stages, flags, and name corrections

## Tooling and verification

- Pinned/hash-verified developer downloads; deterministic generation and ignored asset preparation
- Exact source/palette provenance; inherited edit/retained-generation flags audited
- Reject provisional candidates, unknown inherited flags, duplicate pixels, and unsupported layouts
- Explicit name corrections require exact pinned source/canonical spellings and reasons
- Assistant full tests/vet and generation checks passed; all 226 new sprites inspected
- Tests cover family completeness, baby stages, source quality gates, name aliases,
  uniform species boundaries, missing artwork, and installed-binary behavior
- Routine species/form/asset entries are derived automatically; only policy and exceptions are maintained
- Provenance modules/tests and dataset documentation consolidated
- Static selection moved into catalog with injected sprite availability
- Earlier inventory remains intact; 78 catalog species and 226 assets added
- Pinned form table resolves Unown/Pichu; four reasoned typing exceptions remain
- Source defaults checked against metadata defaults, including Dudunsparce
- Explicit --update-lock derives and validates new artwork pins; ordinary generation preserves locks
- Owner verification and commit remain pending

Storage schema version: none.

## Next

Continue D06 wider inventory, remaining visual genders, and provider/quality audits.
D06 is not complete; D07 selectors and D08 public listing remain unimplemented.
Trainers, encounters, achievements, and TUI remain later milestones.

## Decisions and workflow

- Section 25 governs ignored build-time PNGs, offline embedding, and distribution policy
- Source locks/mappings/generated metadata/reports are tracked; image/cache files are ignored
- Specification remains local and excluded from Git; changes delivered as ZIP bundles
- Owner applies files, verifies, commits, pushes, tags, and publishes
- Assistant GitHub access remains strictly read-only

Generation-3 verification covers Castform weather typing, Deoxys alias folding,
Spinda source-template exclusions, generation completeness and evolution closure.
Source exclusion evidence is tested; no additional package or source file added.


## Developer progress increment

The progress feature was implemented in the preceding increment. It adds
stderr progress for downloads and verified cache hits, a terminal bar/spinner,
plain redirected logs, and optional per-file `--verbose`. Concurrent artwork
lock updates use the same serialized reporter. Tests cover cache corruption,
stdout/stderr separation, concurrent reporting, and reporter shutdown; tests,
vet, and race checks pass. That preceding increment left generated files, pins, dataset identity and runtime
unchanged and is committed at the baseline above.


## Generation-4 increment

Exact form-type overrides come from pinned pokemon_form_types.csv and verified
metadata form/variety ownership. Arceus's 18 supported forms are typed correctly;
its unsupported unknown source appearance is excluded automatically. Tests also
cover Wormadam cloak typing, Rotom appliances, Shaymin Sky, source aliases,
malformed type rows and wrong owners. Existing artwork is unchanged. Generation
checks, tests, vet, race checks and visual review pass. Owner verification/commit
are pending. No additional source file, package, dependency or runtime flag added.
