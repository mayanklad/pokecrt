# Implementation progress

Specification: v1, updated 2 October 2026; approved sections 25–27 apply.
Current release target: v0.2 — Complete public engine.
Last published milestone: v0.1; D01–D05 complete.
Current step: D09 coverage/public behavior verified; D10 benchmarks are next.
Verified baseline commit: fa578c03048c02961424514f782dcc042000ad4e.
Baseline status: D08 catalog owner-committed and pushed; clean tree.
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

D09 adds verification and documentation only; production behavior is unchanged.
Coverage is checked against embedded catalog/manifest identities and public
default queries, including asset ownership, palette/gender eligibility, missing
artwork, generation/national targets and complete branching-family membership.

The public command matrix checks generation OR, type AND/type-any OR, selected
form typing, species color/stage across shiny/forms/genders, false status flags,
empty intersections and unavailable combinations. Installed-binary tests verify
print/list status differences, command-specific invalid flags, detail ANSI and
NO_COLOR, real broken pipes and unchanged nonexistent trainer/data directories.
README examples state exact status and availability behavior.

No runtime code corrections were needed. No new files, packages, dependencies,
flags or commands. Source revisions, dataset, all generated outputs and accepted
artwork remain unchanged. Optional additions or optimizations require owner
confirmation before implementation.

All 2,669 PNGs and all 2,947 source pins remain unchanged. No ignored filenames
need removal in this increment. No trainer data exists, so no storage migration
applies. Previously supported collectible identities and default printing remain.

Checks: full tests, vet, race, build, deterministic generation, fresh asset
preparation and twice-repeated pinned generation pass; artwork is unchanged.
The complete public matrix passes without production-code changes.

## Remaining work and next step

D06–D08 are owner-committed for the reviewed supported inventory; its 53 metadata
scope gaps and unsupported artwork remain explicit limitations. D09 is
assistant-verified with owner application/verification/commit pending. D10
measures startup/memory/rendering and establishes performance budgets before
v0.2 release. Any suggested optimization will be presented for owner confirmation
before code changes. No benchmark result or release gate is claimed yet.
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
