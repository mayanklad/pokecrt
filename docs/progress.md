# Implementation progress

Specification: v1, updated 2 October 2026; approved sections 25–27 apply.
Current release target: v0.2 — Complete public engine.
Last published milestone: v0.1; D01–D05 complete.
Current step: D06, automatic themed-inventory audit; incomplete.
Verified baseline commit: f5fe6d5c31b79ab3a94d706d0aaa3dec9656a22a.
Baseline status: Source-encoded gender increment owner-committed; clean tree.
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

Dataset ID: `90d38f3647a85e1a53f13e066c564e3758ff7a7d6ce0d0c14debc6f897c80fef`.
Rules: d06-auto-11. Coverage: tools/dataset/coverage.json and coverage.md.

- 1,025 catalog species across all nine generations; 1,448 metadata forms.
- 1,017 eligible encounter species; 1,013 standard-printable species.
- 1,327 collectible forms; 2,669 exact assets; 1,334 shiny slots.
- 1,021 standard regular slots; 16 distinct gender slots across eight species.
- 12 unavailable standards; 122 unavailable metadata appearances.
- 2,947 pinned inputs; maximum cropped dimensions 67×56 source pixels.
- Reviewed inherited and bamq providers; generated/provisional candidates excluded.

## Current increment

Exact owner-checked metadata now derives regional, Mega and Gigantamax tags.
Reviewed source-only overrides inherit their mapped variety's default metadata
classification. No species tag lists or source-name guesses are maintained.
Coverage derives future themed goals from accepted regular artwork: 18 types,
38 regional forms, 80 Mega/Gigantamax forms and 17 fully supported branching
families. It also records generation/national species denominators and explicit
limitations for unsupported goals. Entire families must qualify; palettes and
genders do not duplicate form/species targets. Achievement gameplay remains D14.

All 2,669 PNGs and all 2,947 source pins remain unchanged. No ignored filenames
need removal in this increment. No trainer data exists, so no storage migration
applies. Previously supported collectible identities and default printing remain.

Checks: full tests, vet, race, build, deterministic generation, fresh asset
preparation pass; artwork is unchanged. No source files, packages, dependencies
or public flags added. Original source revisions/provider scope unchanged.

## Remaining work and next step

D06 remains open: other provider/layout and
provisional artwork review, source alias semantics and inventory coverage limits.
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
