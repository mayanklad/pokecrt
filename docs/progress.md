# Implementation progress

Specification: v1, updated 2 October 2026; approved sections 25–27 apply.
Current release target: v0.2 — Complete public engine.
Last published milestone: v0.1; D01–D05 complete.
Current step: D10b approved lookup optimization verified; owner final review pending.
Verified baseline commit: ac2deea8fdc7cab85591542cd0264103054075cb.
Baseline status: D10a baseline owner-committed/pushed and GitHub commit verified; clean tree.
Current increment: assistant verification complete; owner verification/commit pending.
Published release: https://github.com/mayanklad/pokecrt/releases/tag/v0.1

## Implemented behavior

- Root/help/version; named or uniformly random matching-species printing.
- Composable generation/type/color/stage/status and exact form/gender/shiny selectors.
- Compact/sprite output, natural-size truecolor half blocks, transparency.
- NO_COLOR, piped colors and quiet broken pipes.
- Public compact catalog and detailed entries with shared selectors.
- No trainer storage or runtime downloads; listing has no discovery restriction.
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

D10b implements the owner-approved exact-key lookup index in the existing sprite
package. Entry positions are derived automatically from the generated manifest;
the private map is read-only after initialization. Manifest order, returned
value ownership, exact lookup/absence behavior and source decoding are retained.
Tests cover every accepted asset, absent identity parts and copy isolation.
No new files, packages, dependencies, commands, flags or handwritten entries.

The benchmark report retains D10a, adds matched fresh-process comparisons,
indexed warm measurements, independent RSS and the owner's i7 pre-index reference.
Color random/list fresh medians improve about 3.4×. Several short commands cost
about 0.1–0.3 ms more to start; peak RSS remains below 7 MiB, with short-command
increases around 0.5–0.8 MiB. The stripped binary remains 6,373,536 bytes.
All indexed scenarios meet the unchanged initial EPYC-environment budgets.
Owner laptop timings must be compared against its own pre-index baseline.

README now documents the v0.2 candidate workflow and final owner release checks.
True cold filesystem measurements are not claimed; fresh processes include all
application initialization on a warm page cache. Encounter/history latency waits
for its specified later trainer milestone. No product scope is moved.

All 2,669 PNGs and all 2,947 source pins remain unchanged. No ignored filenames
need removal in this increment. No trainer data exists, so no storage migration
applies. Previously supported collectible identities and default printing remain.

Checks: full tests, vet, race, pinned preparation/check and release packaging
pass. The extracted candidate passes offline public command checks outside the
checkout. Generated outputs and all PNG bytes match the baseline. Repeated
five-run benchmarks, 100 process samples per case/mode/build and ten RSS samples
per case/mode/build document improvements and costs.

## Remaining work and next step

D06–D10a are owner-committed for the reviewed supported inventory; its 53 metadata
scope gaps and unsupported artwork remain explicit limitations. D10b is
assistant-verified with owner application/verification/commit pending.
Next: final owner terminal/benchmark review, final commit/tag, clean archive
verification and v0.2 publication. Do not begin D11 storage before closing the
public-engine release milestone. Trainer CLI, progression and TUI remain later.
The owner must confirm any further suggested optimization before implementation.

Storage schema version: none.

## Approved workflow

- Specification stays local at docs/specification-v1.md and excluded from Git.
- Source locks, mappings, generated Go/reports/notices are tracked; PNG/cache ignored.
- ZIPs contain only changed source/config/tests/docs; owner regenerates outputs.
- Owner applies, verifies, commits, pushes, tags and publishes.
- Assistant GitHub access remains strictly read-only.
- Fan-project distribution and separate artwork rights follow section 25 and
  docs/release-policy.md; metadata and original provider credits remain preserved.
