# Implementation progress

Specification revision/date: 30 September 2026
Current release target: v0.1 — Basic printer
Last verified development step: D02b — Normalized catalog and asset generation
Current source commit: 08595f1591efffad8f77a56e2e71ae231f11552c
Commit reference note: D02a baseline; D02b is verified locally and awaiting commit.

Implemented runtime commands:
- Root help
- --help / -h
- --version with generated dataset identity

Implemented developer tooling:
- Pinned downloads and SHA-256 verification
- Catalog and asset normalization
- Deterministic generation
- Offline generated-output drift checks
- Coverage reports and third-party notices

Dataset ID:
5bb40e33703ffd1b07855ba3552cff88edd4f8d2c0d03f2860872aee279803e4

Coverage:
- 9 catalog species
- 3 eligible species and standard regular sprites
- 0 shiny sprites
- 0 distinct visual gender slots
- 3 exact eligible variants
- Reports: tools/dataset/coverage.json and coverage.md

Storage schema version: none

Verification:
- User formatting, tests, vet, and build passed
- Generation and offline drift checks passed
- Version output matches the generated dataset ID
- Local reproducibility and malformed-input tests passed

Known limitations:
- Renderer and print command are not implemented yet
- Six catalog species have no selected artwork
- Full coverage, forms, palettes, and visual genders remain scheduled for D06
- Source-image redistribution gate remains open under specification section 13
- Generated PNGs remain local pending that gate

Next development step: D03 — Transparent half-block rendering

Approved deviations:
- Specification remains local and excluded from Git history
- D02 split into input verification and normalization/bundling

Workflow:
- Complete files in chat or individual files; no ZIPs
- User applies changes, verifies, commits, and pushes
- Assistant GitHub operations remain read-only
