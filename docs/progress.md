# Implementation progress

Current development milestone: v0.2 — Complete public engine.
Published milestone: [v0.1](https://github.com/mayanklad/pokecrt/releases/tag/v0.1).
Implementation baseline: `d11627713611e097097b221dbbb10b260ba6e60d`.

## Implemented behavior

D01–D10 implement the public engine:

- Root/help/version; named or uniformly random matching-species printing.
- Composable generation/type/color/stage/status and exact form/gender/shiny selectors.
- Compact/sprite output, natural-size truecolor half blocks and transparency.
- NO_COLOR, piped colors and quiet broken pipes.
- Public compact catalog and detailed entries with shared selectors.
- No trainer storage or runtime downloads; listing has no discovery restriction.
- Pinned/hash-verified preparation, provenance and deterministic generation.
- Routine inventory derived from source data; only policies/corrections maintained.
- Exact sprite lookup indexed automatically from the generated manifest.

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

The catalog also reports 53 metadata varieties without resolved catalog/source
identities; these are separate from missing artwork counts. No fallback artwork
is inferred. Storage schema version: none.

## Validation and performance

The public suite covers selector composition, exact artwork identity, inventory,
coverage, status/stream behavior and installed execution outside the source tree.
Tests, vet, race checks and pinned dataset checks pass for the implementation
baseline. Benchmark methods, environments, startup/memory tradeoffs and numeric
budgets are recorded in [benchmarks.md](benchmarks.md).

Intel i7 warm color random/list medians are 3.274/3.791 ms versus
12.705/13.394 ms before indexing: about 3.88×/3.53× improvement. These figures
exclude process startup and are not universal latency guarantees.

## Roadmap

v0.2 provides the complete public print/catalog engine for the reviewed inventory.
[Release notes](release-v0.2.md) describe its behavior and known limitations.
v0.3 covers trainer profiles, SQLite storage, encounters, progression, achievements
and private Pokédex CLI. The TUI and final hardening remain later milestones.
Those features are not implemented in the current public engine.
