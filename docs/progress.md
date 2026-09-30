# Implementation progress

Specification revision/date: 30 September 2026
Current release target: v0.1 — Basic printer
Current development step: v0.1 release preparation — D06 paused
Current source commit: 959e4f66b89d69d73a2ca116ade08c459983d9c7
Commit reference note: D05 baseline. Release licensing/documentation changes await user application. No release is published.

Implemented runtime commands:
- Root help
- --help / -h
- --version with generated dataset identity
- print --help / -h
- print with exact canonical species name or generated alias
- print with uniform random eligible species selection
- compact (default) and sprite output

Implemented developer tooling:
- Pinned downloads and SHA-256 verification
- Catalog and asset normalization with deterministic transparent-margin cropping
- Deterministic generation and offline generated-output drift checks
- Coverage reports and third-party notices
- Local sprite preview via tools/render-preview
- Local executable installation via scripts/install.sh
- Linux amd64 candidate packaging and checksums via scripts/package.sh
- Release checklist in docs/release-v0.1.md

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
- D03 committed; user real-terminal review is not recorded here
- D04 assistant mirror tests, vet, and build passed
- D04 exact output, selector boundaries, failures, NO_COLOR, and no-state tests passed
- D04 user formatting, tests, vet, build, printing, and error statuses passed
- D05 assistant tests and vet passed
- D05 installed-binary tests verify version, piped ANSI, NO_COLOR, closed pipes, and no trainer state
- D05 installer smoke check passed
- D05 packaging refuses a missing LICENSE; temporary fixture archive verification passed
- Real local v0.1 candidate archive, checksum, Linux amd64 executable, version, and extracted-binary checks passed in the assistant mirror
- User visual/network-disabled checks remain pending; this environment does not permit an isolated network namespace

Known limitations:
- Public filters, forms, genders, and shiny selectors await D06
- Six catalog species have no selected artwork
- Full coverage, forms, palettes, and visual genders remain scheduled for D06
- Source-image redistribution gate remains open under specification section 13
- Generated PNGs remain local pending that gate
- Owner selected MIT for original code; LICENSE and LICENSING.md are prepared
- Public v0.1 tag/release remains unpublished

Next development step: complete v0.1 release checks; D06 stays unapplied
Release work: resolve code/image licensing and complete v0.1 smoke checks before publication

Approved deviations:
- Specification remains local and excluded from Git history
- D02 split into input verification and normalization/bundling

Workflow:
- Complete files in chat or individual files; no ZIPs
- User applies changes, verifies, commits, and pushes
- Assistant GitHub operations remain read-only
