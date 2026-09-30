# Implementation progress

Specification revision/date: 30 September 2026
Current release target: v0.1 — Basic printer
Last completed development step: D02a — Pinned source inputs and verification
Current source commit: 7c7c67b18e265ca25315105ccf66c8348904ee33
Commit reference note: D01 baseline; D02a is verified and awaiting commit.

Implemented runtime commands/flags:
- Root help
- --help / -h
- --version

Implemented D02a increment:
- Source lock for 16 inputs with full revisions, sizes, SHA-256, terms, and attribution
- Developer-only --fetch / --check with explicit --sources and --cache paths
- Offline cache verification and rejection of corrupt or mismatched downloads
- Source audit record; downloaded inputs remain excluded from Git

Dataset ID and coverage report: unbundled; normalization and generated coverage pending D02b
Storage schema version: none

Checks run and results:
- D01 committed source reviewed through read-only GitHub access
- D01 root/help/version behavior and duplicate-version exit status 2 verified
- D01 and D02a unit tests and vet passed locally using Go 1.27.1 linux/amd64
- All 16 pinned input checksums calculated from actual downloaded bytes
- Bulbasaur, Charizard, and Squirtle PNGs decoded and inspected for dimensions, transparent bounds, and binary alpha
- Local application build passed with -buildvcs=false because the verification mirror lacks repository metadata
- Real fetch into an empty cache and offline cached verification passed
- User's D02a formatting, tests, vet, and application build passed
- User's fetch/check and offline check each reported: Verified 16 pinned inputs.
- User's Git status showed only expected README, progress, and tools changes

Known issues/blockers:
- No runtime printer or trainer features implemented yet
- Source-image copyrights are separate from repository code licenses; the source-image redistribution verification gate remains open
- Generated artwork must be classified explicitly, never silently treated as original sprites
- Source alias, visual gender, palette, and form normalization remain incomplete

Next development step:
- Commit and push the verified D02a increment
- Review the committed source through read-only GitHub access
- Implement D02b: normalized catalog, mapping records, embedded assets and manifest, dataset ID, deterministic coverage, and generated-output drift checks

Explicit approved deviations:
- The specification is maintained locally and excluded from Git history
- D02 is split into D02a input pinning/verification and D02b normalization/bundling for separate provenance and generation review

Collaboration workflow:
- Provide complete changed files directly in chat or as individual files; no ZIP archives
- The user applies repository changes, runs checks, creates commits, and pushes
- Assistant GitHub operations remain read-only
- Advance to the next implementation increment after user verification
