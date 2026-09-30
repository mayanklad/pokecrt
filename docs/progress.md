# Implementation progress

Specification revision/date: 30 September 2026
Current release target: v0.1 — Basic printer
Current development step: D03 — Transparent half-block rendering
Current source commit: 4064fb215dff2bd3199d6e9092cc575005678f88
Commit reference note: D02b baseline; D03 awaits user verification and commit.

Implemented runtime commands:
- Root help
- --help / -h
- --version with generated dataset identity

Implemented developer tooling:
- Pinned downloads and SHA-256 verification
- Catalog and asset normalization with deterministic transparent-margin cropping
- Deterministic generation and offline generated-output drift checks
- Coverage reports and third-party notices
- Local sprite preview via tools/render-preview

Implemented rendering:
- Truecolor half blocks at source-pixel scale
- Transparent halves use the terminal's default background
- Odd heights, nonzero image origins, and color resets
- Monochrome block output when color is disabled
- Partial-alpha rejection without returning partial output
- Preview honors nonempty NO_COLOR and preserves ANSI through pipes otherwise

Dataset ID:
5bb40e33703ffd1b07855ba3552cff88edd4f8d2c0d03f2860872aee279803e4

Coverage:
- 9 catalog species
- 3 eligible species and standard regular sprites
- 0 shiny sprites
- 0 distinct visual gender slots
- 3 exact eligible variants
- Reports: tools/dataset/coverage.json and coverage.md
- Explicit Git ignore exceptions allow these reports to be committed

Storage schema version: none

Verification:
- D02b user formatting, tests, vet, build, generation, and drift checks passed
- D03 assistant mirror tests and vet passed
- D03 preview build and piped ANSI/NO_COLOR checks passed
- D03 user tests, build, and real-terminal visual review pending

Known limitations:
- Public print commands are not implemented yet
- Six catalog species have no selected artwork
- Full coverage, forms, palettes, and visual genders remain scheduled for D06
- Source-image redistribution gate remains open under specification section 13
- Generated PNGs remain local pending that gate

Next development step: D04 — Named and random printing, after D03 verification

Approved deviations:
- Specification remains local and excluded from Git history
- D02 split into input verification and normalization/bundling

Workflow:
- Complete files in chat or individual files; no ZIPs
- User applies changes, verifies, commits, and pushes
- Assistant GitHub operations remain read-only
