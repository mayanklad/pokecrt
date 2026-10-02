# Implementation progress

Specification: v1, updated 2 October 2026; approved sections 25–27 apply.
Current release target: v0.2 — Complete public engine.
Last published milestone: v0.1; D01–D05 complete.
Current step: D06, automatic inherited visual-gender audit; incomplete.
Verified baseline commit: 7a901ba813188b7d33a1d155b427bbdf12c1ab6f.
Baseline status: Generation 9 provider increment owner-committed; clean tree.
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

Dataset ID: `7b4c9f35addda4a07ba4463119dfbd981b382bfa9e6a291f264c9648bdfd45af`.
Rules: d06-auto-8. Coverage: tools/dataset/coverage.json and coverage.md.

- 1,025 catalog species across all nine generations; 1,447 metadata forms.
- 1,017 eligible encounter species; 1,013 standard-printable species.
- 1,327 collectible forms; 2,669 exact assets; 1,334 shiny slots.
- 1,021 standard regular slots; 16 distinct gender slots across eight species.
- 12 unavailable standards; 120 unavailable metadata appearances.
- 2,947 pinned inputs; maximum cropped dimensions 67×56 source pixels.
- Reviewed inherited and bamq providers; generated/provisional candidates excluded.

## Current increment

The inherited inventory now derives standard male/female pairs automatically,
replacing the maintained species list. Require exact identity, explicit female
artwork declarations, approved provider/layout and nonprovisional quality flags.
Both genders need distinct regular pixels and locked palette inputs.
Hippopotas, Hippowdon, Unfezant, Frillish, Jellicent and Indeedee add 12 female
assets alongside the earlier Pyroar/Meowstic pairs. No extra collectible forms.
Existing default identities for those six species become male; all preceding
2,657 image contents and default print artwork stay unchanged. No trainer data
exists, so no storage migration applies. Remove their 12 obsolete ignored default
filenames before generating, as documented in tools/dataset/README.md.

Checks: full tests, vet, race, build, deterministic generation, fresh asset
preparation and visual comparison pass. No source files, packages, dependencies
or public flags added. Original source revisions/provider scope unchanged.

## Remaining work and next step

D06 remains open: Oinkologne form/gender identity, other provider/layout and
provisional artwork review, source alias semantics, and achievement tags.
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
