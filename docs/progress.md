# Implementation progress

Specification: v1, updated 2 October 2026; approved sections 25–27 apply.
Current release target: v0.2 — Complete public engine.
Last published milestone: v0.1; D01–D05 complete.
Current step: D06 closing audit verified for the reviewed inventory; D07 next.
Verified baseline commit: c2a26e87cb726a6fee3a7242a24fa00358b70956.
Baseline status: Themed-inventory increment owner-committed; clean tree.
Current increment: assistant verification complete; owner verification/commit pending.
Published release: https://github.com/mayanklad/pokecrt/releases/tag/v0.1

## Implemented behavior

- Root/help/version; named or uniformly random standard regular printing.
- Compact/sprite output, natural-size truecolor half blocks, transparency.
- NO_COLOR, piped colors and quiet broken pipes.
- No trainer storage or runtime downloads; public selectors remain D07.
- Pinned/hash-verified preparation, exact provenance and deterministic generation.
- Routine inventory derived from source data; only policies/corrections maintained.

## Dataset

Dataset ID: `8e5073aeeff3f7761e26ccdf9f068189516ae7eded9c0998a9d970b7d37e6b60`.
Rules: d06-auto-12. Coverage: tools/dataset/coverage.json and coverage.md.

- 1,025 catalog species across all nine generations; 1,448 metadata forms.
- 1,017 eligible encounter species; 1,013 standard-printable species.
- 1,327 collectible forms; 2,669 exact assets; 1,334 shiny slots.
- 1,021 standard regular slots; 16 distinct gender slots across eight species.
- 12 unavailable standards; 122 unavailable metadata appearances.
- 2,947 pinned inputs; maximum cropped dimensions 67×56 source pixels.
- Reviewed inherited and bamq providers; generated/provisional candidates excluded.

## Current increment

Coverage now automatically reports 53 pinned metadata varieties without
resolved catalog/source identities, separately from the 122 unavailable catalog
appearances. Primary mappings, aliases and declared genders resolve identities;
metadata outside the source scope never creates guessed sprites or collectibles.
These are variety-level gaps, not necessarily distinct missing images.

All 125 folded ordinary source aliases pass exact metadata owner/type checks.
Conflicting variety/form ownership or type sets now fail generation. The two
Tatsugiri Mega identities remain separate and unavailable. Provider declarations
and their generated/provisional exclusions were audited; no acceptance gate is
relaxed. Current supported forms, genders, palettes, tags and provenance are
verified. Achievement gameplay remains D14.

All 2,669 PNGs and all 2,947 source pins remain unchanged. No ignored filenames
need removal in this increment. No trainer data exists, so no storage migration
applies. Previously supported collectible identities and default printing remain.

Checks: full tests, vet, race, build, deterministic generation, fresh asset
preparation pass; artwork is unchanged. No source files, packages, dependencies
or public flags added. Original source revisions/provider scope unchanged.

## Remaining work and next step

D06 is verified for the reviewed supported inventory; owner checks are pending.
Unsupported candidates and the 53 metadata scope gaps remain explicit limitations.
Further providers/layouts/artwork require separate reviewed dataset changes;
D06 does not require accepting every upstream candidate.
D07 implements composable public selectors using FlagSet and shared validation;
the existing handwritten print parser will be replaced there. D08 adds listing.
Trainer CLI, storage, progression and TUI remain their specified later milestones.
Performance measurements and budgets remain required before v0.2 release.

Storage schema version: none.

## Approved workflow

- Specification stays local at docs/specification-v1.md and excluded from Git.
- Source locks, mappings, generated Go/reports/notices are tracked; PNG/cache ignored.
- ZIPs contain only changed source/config/tests/docs; owner regenerates outputs.
- Owner applies, verifies, commits, pushes, tags and publishes.
- Assistant GitHub access remains strictly read-only.
- Fan-project distribution and separate artwork rights follow section 25 and
  docs/release-policy.md; metadata and original provider credits remain preserved.
