# Implementation progress

Specification revision/date: 30 September 2026, including approved section 25
Current release target: v0.2 — Complete public engine
Last completed milestone: v0.1 published
Current increment: D06b second artwork batch and alias/gender audit, prepared for owner verification
Current source commit: d5a51817a79ad894e59ccf98ed075177a2f91a2a
Commit note: verified first artwork-batch baseline before this increment; update on the next increment.
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
`814889022ff532a6a2b5df724f7cca5d785c533441109e4ebfb8c454b9e050ff`

Published v0.1 dataset ID:
`5bb40e33703ffd1b07855ba3552cff88edd4f8d2c0d03f2860872aee279803e4`

- 24 catalog species and 36 metadata forms with exact PokéAPI variety typing
- 24 eligible species and standard regular sprites; 36 collectible forms
- 36 shiny assets, 72 exact variants, and 0 distinct visual gender slots
- No missing regular appearances within the current 24-species batch
- 85 pinned source inputs; reports in tools/dataset/coverage.json and coverage.md
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
- Naming cleanup committed by owner
- Assistant batch generation/check, tests, and vet passed; all 20 newly added sprite pairs visually inspected
- Raticate Totem aliases folded into Alolan identity; no duplicate collectibles
- Separate-layout female artwork found upstream; explicit provenance support remains pending
- Owner terminal review and commit of this batch remain pending

## Next

Continue D06b wider inventory and visual-gender audits, then D07 filters and selectors.
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
