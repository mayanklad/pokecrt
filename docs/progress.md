# Implementation progress

Specification revision/date: 30 September 2026, including approved section 25
Current release target: v0.2 — Complete public engine
Last published milestone: v0.1
Current increment: D06 automatic generation-2 inventory and evolution-family closure
Current source commit: 3b1cd4c8fa69ff4df7c24421c8bd773cf3c10a22
Commit note: verified baseline before this increment; owner verification/commit pending.
Published release: https://github.com/mayanklad/pokecrt/releases/tag/v0.1

## Runtime

- Root/help/version; named or uniformly random standard regular printing
- Compact/sprite output, natural-size truecolor half blocks, transparency
- NO_COLOR, piped colors, quiet broken pipes; no trainer state or runtime downloads
- No new public flags; explicit form/shiny/gender selectors remain D07

## Dataset

Development dataset ID:
`a742a00b3f123e70455d5a0009a55a325eb4f2ce73418f23b8f717d23a1e7a02`

- 293 catalog species: all generation-1 and generation-2 species plus full evolution-family closure and existing gen-6 families
- 413 metadata forms; 391 collectible forms; 786 exact regular/shiny assets
- 284 eligible species; 286 standard regular slots; 393 shiny slots
- 4 distinct visual gender slots across Pyroar/Meowstic, with male defaults
- 22 excluded form appearances; nine species lack audited standard artwork
- 801 pinned inputs; coverage.json/coverage.md record availability and quality
- Direct images remain pinned PokéSprite-v2; inherited provider records stay distinct
- Complete evolution references, aliases, typing, stages, flags, and name corrections

## Tooling and verification

- Pinned/hash-verified developer downloads; deterministic generation and ignored asset preparation
- Exact source/palette provenance; inherited edit/retained-generation flags audited
- Reject provisional candidates, unknown inherited flags, duplicate pixels, and unsupported layouts
- Explicit name corrections require exact pinned source/canonical spellings and reasons
- Assistant full tests/vet and generation checks passed; all 252 new sprites inspected
- Tests cover family completeness, baby stages, source quality gates, name aliases,
  uniform species boundaries, missing artwork, and installed-binary behavior
- Routine species/form/asset entries are derived automatically; only policy and exceptions are maintained
- Provenance modules/tests and dataset documentation consolidated
- Static selection moved into catalog with injected sprite availability
- Gen1 inventory remains intact; 101 catalog species and 252 assets added
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
