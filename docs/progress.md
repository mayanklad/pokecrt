# Implementation progress

Specification revision/date: 30 September 2026
Specification amendment: docs/release-policy.md, approved by owner
Current release target: v0.1 — Basic printer
Current development step: final v0.1 publication preparation; D06 paused
Current source commit: c3e184fe4a0bc9b867feee1cf81b6ad0b020a0b1
Commit reference note: baseline before final policy/notices/build-preparation commit.
Publication status: final tag and GitHub release are not yet confirmed.

Implemented runtime commands:
- Root help, --help / -h, --version with dataset identity
- print --help / -h
- Named or uniformly random standard regular printing
- Compact (default) and sprite-only output

Implemented developer tooling:
- Pinned downloads and SHA-256 verification
- Deterministic catalog/assets/notices/coverage generation and offline drift checks
- Local render preview, installation, and Linux amd64 archive/checksum packaging

Rendering:
- Truecolor source-pixel half blocks and terminal-default transparency
- Odd-height handling and color resets
- Nonempty NO_COLOR suppresses ANSI; pipes otherwise preserve colors
- Partial alpha is rejected; failed rendering returns no partial artwork

Dataset ID:
5bb40e33703ffd1b07855ba3552cff88edd4f8d2c0d03f2860872aee279803e4

Coverage:
- 9 catalog species
- 3 eligible species, standard regular sprites, and exact variants
- 0 shiny sprites and distinct visual gender slots
- Six catalog species lack artwork
- Coverage reports: tools/dataset/coverage.json and coverage.md

Storage schema version: none

Verification:
- User generation checks, tests, vet, and build passed
- Candidate checksums and matching v0.1/dataset metadata verified
- User screenshots show all three colored sprites on a light background
- Monochrome NO_COLOR output and network-isolated unshare printing verified
- Assistant installed-binary tests cover pipe handling and no trainer state
- Assistant tests, vet, and race suite passed after build-preparation changes
- Final policy/notices/build-preparation commit, clean tag rebuild, and publication remain pending

Distribution decision:
- Owner selected MIT for original code
- Owner approved attributed fan-project bundles on 30 September 2026
- Underlying image rights remain unresolved; no clearance or endorsement is claimed
- Generated PNGs remain outside Git; pinned preparation and Go embedding support offline binaries
- Package preserves licensing scope, provenance, notices, coverage, and policy

Known limitations:
- Public filters/selectors await D07; inventory expansion remains D06
- Trainers, encounters, achievements, and TUI are not implemented
- Initial verified platform is Linux amd64

Next: publish v0.1, then resume D06 from this baseline.
D06a files previously supplied have not been applied.

Approved deviations:
- Specification stays local and excluded from Git history
- D02 split into input verification and normalization/bundling
- Section 13 amended by the owner-approved fan-project release policy

Workflow:
- Complete files in chat or individual downloads; no ZIPs
- User applies changes, tests, commits, pushes, tags, and publishes
- Assistant GitHub operations remain read-only
