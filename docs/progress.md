# Implementation progress

Specification: v1, updated 2 October 2026; approved sections 25–27 apply.
Current release target: v0.2 — Complete public engine.
Last published milestone: v0.1; D01–D05 complete.
Current step: D07 selectors verified; D08 public catalog is next.
Verified baseline commit: b90dba38b7475c4d1d409bb5ec1114c68cd2f2d6.
Baseline status: D06 closing audit owner-committed; clean tree.
Current increment: assistant verification complete; owner verification/commit pending.
Published release: https://github.com/mayanklad/pokecrt/releases/tag/v0.1

## Implemented behavior

- Root/help/version; named or uniformly random matching-species printing.
- Composable generation/type/color/stage/status and exact form/gender/shiny selectors.
- Compact/sprite output, natural-size truecolor half blocks, transparency.
- NO_COLOR, piped colors and quiet broken pipes.
- No trainer storage or runtime downloads; public catalog listing remains D08.
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

Print now uses flag.FlagSet with shared one-occurrence validation and selector
registration. Root flags use the same duplicate/alias checks. Lists trim and
deduplicate; bundled vocabulary validation rejects unknown static selectors.
Boolean false status filters impose no restriction. No CLI framework dependency.

The shared query evaluates all constraints against one exact selected form,
retains unavailable regular metadata entries for future listing, and requires
actual shiny assets for shiny selection. Print selects matching renderable
species uniformly and never borrows another appearance's artwork/typing.
Headings show form, explicitly requested gender and shiny in the specified order.
Defaults remain standard/regular/form-designated gender.

Tests cover selector composition, valid empty intersections versus invalid
invocations, gender and palette absence, exact output bytes, uniform species
weighting, boolean false, duplicate flags, isolated query results, offline
installed variants and state-free public commands.

All 2,669 PNGs and all 2,947 source pins remain unchanged. No ignored filenames
need removal in this increment. No trainer data exists, so no storage migration
applies. Previously supported collectible identities and default printing remain.

Checks: full tests, vet, race, build, deterministic generation, fresh asset
preparation pass; artwork is unchanged. No source files, packages, dependencies
added. Public selector flags implement D07; no new commands. Original source revisions/provider scope unchanged.

## Remaining work and next step

D06 is verified for the reviewed supported inventory; its 53 metadata scope gaps
and unsupported artwork remain explicit limitations. D07 is assistant-verified
with owner application/verification/commit pending. The handwritten print parser
has been replaced by FlagSet and shared validation. D08 adds compact listing and
detailed catalog entries using the same selected-appearance query.
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
