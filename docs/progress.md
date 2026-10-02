# Implementation progress

Specification: v1, updated 2 October 2026; approved sections 25–27 apply.
Current release target: v0.2 — Complete public engine.
Last published milestone: v0.1; D01–D05 complete.
Current step: D08 public catalog verified; D09 coverage/public behavior is next.
Verified baseline commit: 0e94a317fde4166de6adab1c3ca201a069121ecb.
Baseline status: D07 selectors owner-committed and pushed; clean tree.
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

Public list now uses the D07 shared FlagSet selector registration/validation and
exact-appearance query. Compact rows are aligned, one per species, in National
order; metadata-only regular entries remain visible and marked unavailable.
Valid empty lists succeed, while invalid invocations return status 2.

Details show exactly one selected sprite or an unavailable message, verified
metadata, the complete connected evolution family and all catalog forms with
actual regular/shiny/gender availability. Public headings are shared with print.
No alternate artwork is borrowed, results are uncapped and no pager is launched.
Two files are added within internal/cli for the list command and its tests.
No new packages or dependencies; this follows the existing print command layout.

Tests cover full sorted catalog coverage, exact selected sprite bytes/headings,
NO_COLOR, missing standard/variant assets, shiny/gender inventory, connected
branches, consistent entry separators, empty lists, invalid/repeated flags,
broken pipes, state-free commands and an offline installed binary.

All 2,669 PNGs and all 2,947 source pins remain unchanged. No ignored filenames
need removal in this increment. No trainer data exists, so no storage migration
applies. Previously supported collectible identities and default printing remain.

Checks: full tests, vet, race, build, deterministic generation, fresh asset
preparation pass; artwork is unchanged. Source revisions/provider scope and
dataset identity are unchanged. D08 adds list and its --details flag.

## Remaining work and next step

D06 is verified for the reviewed supported inventory; its 53 metadata scope gaps
and unsupported artwork remain explicit limitations. D07 is owner-committed.
D08 is assistant-verified with owner application/verification/commit pending.
D09 validates the full public behavior matrix and coverage/help/examples. D10
then measures startup/memory/rendering and establishes real performance budgets.
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
