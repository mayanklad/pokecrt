# Implementation progress

Specification: v1, updated 2 October 2026; approved sections 25–27 apply.
Current release target: v0.2 — Complete public engine.
Last published milestone: v0.1; D01–D05 complete.
Current step: D10a baseline measured; lookup optimization awaits owner confirmation.
Verified baseline commit: 8490227efd6e7c5fd3782995acf7bfe739d4fda5.
Baseline status: D09 validation owner-committed and GitHub commit verified; clean tree.
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

D10a adds repeatable benchmarks to four existing test files and introduces the
specified docs/benchmarks.md report. It measures fresh-process random/named/
filtered/variant print and compact/detailed list, warm query/CLI/decode/render/
lookup, binary size and peak child RSS. The report records machine, toolchain,
dataset, commands, samples, initial regression budgets and limitations.

All tested scenarios meet the initial measured budgets. True cold filesystem
startup is not claimed; shared page caches are not evicted. Encounter/history
latency remains deferred until the trainer implementation exists.

The random-print CPU profile attributes 73.54% cumulative sampled CPU to repeated
sprite.Lookup scans and key equality. An automatically derived exact-key index
in the existing sprite package is proposed but awaits owner confirmation.
Runtime code is unchanged; no new packages, dependencies, flags or commands.
Only docs/benchmarks.md is a new tracked file. Source revisions, dataset,
generated outputs and accepted artwork remain unchanged.

D10 is split into baseline and any owner-approved optimization so the owner can
review measured evidence before runtime changes, as requested. The initial
baseline remains available for the follow-up comparison. No v0.2 release gate
or publication is claimed by this measurement-only increment.

All 2,669 PNGs and all 2,947 source pins remain unchanged. No ignored filenames
need removal in this increment. No trainer data exists, so no storage migration
applies. Previously supported collectible identities and default printing remain.

Checks: full tests, vet, race, release-style build and pinned dataset check pass.
Five 500-ms benchmark runs per case, 100 process samples per case/mode and ten
independent RSS measurements per case/mode are recorded. No timing assertion is
added to correctness tests.

## Remaining work and next step

D06–D09 are owner-committed for the reviewed supported inventory; its 53 metadata
scope gaps and unsupported artwork remain explicit limitations. D10a is
assistant-verified with owner application/verification/commit pending.
Next: obtain owner confirmation for the measured exact-key lookup optimization,
then implement/compare/validate it if approved. Complete final terminal and
release archive verification before owner publication of v0.2.
Trainer CLI, storage, progression and TUI remain their specified later milestones.

Storage schema version: none.

## Approved workflow

- Specification stays local at docs/specification-v1.md and excluded from Git.
- Source locks, mappings, generated Go/reports/notices are tracked; PNG/cache ignored.
- ZIPs contain only changed source/config/tests/docs; owner regenerates outputs.
- Owner applies, verifies, commits, pushes, tags and publishes.
- Assistant GitHub access remains strictly read-only.
- Fan-project distribution and separate artwork rights follow section 25 and
  docs/release-policy.md; metadata and original provider credits remain preserved.
