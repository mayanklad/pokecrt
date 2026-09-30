# Implementation progress

Specification revision/date: 30 September 2026, including approved section 25
Current release target: v0.2 — Complete public engine
Last completed milestone: v0.1 published
Current increment: separate source-ID/cache naming cleanup, prepared for owner verification
Current source commit: d00c8a389f6a20360188eeca8499bd9ad6bd9538
Commit note: verified D06a baseline before this naming cleanup; update on the next increment.
Release: https://github.com/mayanklad/pokecrt/releases/tag/v0.1
Publication verified: normal release with Linux amd64 archive and SHA256SUMS attached.

## Runtime

- Root/help/version and named or uniformly random standard regular printing
- Compact and sprite output, truecolor half blocks, transparency, and natural size
- Nonempty NO_COLOR, preserved piped colors, quiet broken pipes
- No trainer storage, runtime network, or source-tree dependency
- Public selectors remain unchanged; alternate selectors arrive in D07

## Dataset

Development dataset ID:
`3d5af3be715de18bf313b0b00c87c14fabcee8c510d96f0a6fa4ac526303e47e`

Published v0.1 dataset ID:
`5bb40e33703ffd1b07855ba3552cff88edd4f8d2c0d03f2860872aee279803e4`

- 9 catalog species and 16 metadata forms with exact PokéAPI variety typing
- 3 eligible species, collectible forms, standard regular sprites, and exact variants
- 0 shiny assets and distinct visual gender slots
- 13 missing regular form/gender appearances; no artwork added in D06a
- Same 16 pinned source inputs; reports in tools/dataset/coverage.json and coverage.md
- Direct artwork source: pinned PokéSprite-v2; inherited provenance preserved separately

## Developer tooling

- Explicit form ownership, source identities/aliases, genders, and palette mappings
- Reject wrong typing ownership, undeclared forms, duplicate variant identities,
  generated/aliased candidates, shiny fallback, and unsupported gender pairs
- Deterministic catalog, manifest, coverage, and notices generation
- Ignored PNG preparation checks committed generated metadata before writing assets
- Source locks/mappings/reports are tracked; downloaded/generated PNGs stay outside Git
- Installed binaries embed assets and remain offline

Storage schema version: none.

## Verification

- Published v0.1 source and release metadata reconciled using read-only GitHub access
- Prior owner tests, vet, race tests, build, checksums, and extracted offline smoke passed
- Assistant D06a generation/check, full tests, and vet passed
- Fixtures cover form typing, aliases, real shiny differences, visual gender pairs,
  invalid mappings, and folding source gender identities into one collectible form
- D06a committed by owner; naming cleanup generation/check, tests, and vet passed
- Source ID/cache folder now pokesprite-v2; all pinned inputs and sprite bytes unchanged
- Owner verification and commit of the naming cleanup remain pending

## Next

Complete D06b wider inventory/source audit, then D07 filters and appearance selectors.
D06 as a whole is not complete; no v0.2 release is ready yet.
Trainers, encounters, achievements, and TUI remain future milestones.

## Approved decisions and workflow

- Specification remains local and excluded from Git history
- D06 split into schema/validation (D06a) and wider inventory (D06b) to review independently
- Owner-approved attributed fan distribution; MIT applies to original code, not artwork
- Licensing scope, source terms, attribution, coverage, and release policy are preserved
- User applies files, tests, commits, pushes, tags, and publishes
- Assistant GitHub access remains strictly read-only
- Complete files or individual links; no ZIP bundles
