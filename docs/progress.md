# Implementation progress

Published milestone: [v0.2 — Complete public engine](https://github.com/mayanklad/pokecrt/releases/tag/v0.2), 2 October 2026.
Release source: `59fea7c10a1de45d5732f4bd84617757e6e57490`.
Next planned milestone: v0.3 — Trainer CLI.

## Implemented behavior

D01–D10 implement the public engine:

- Root/help/version; named or uniformly random matching-species printing.
- Composable generation/type/color/stage/status and exact form/gender/shiny selectors.
- Compact/sprite output, natural-size truecolor half blocks and transparency.
- NO_COLOR, piped colors and quiet broken pipes.
- Public compact catalog and detailed entries with shared selectors.
- Public commands never open trainer storage or download at runtime; listing has no discovery restriction.
- Pinned/hash-verified preparation, provenance and deterministic generation.
- Routine inventory derived from source data; only policies/corrections maintained.
- Exact sprite lookup indexed automatically from the generated manifest.

## Storage foundation

D11 adds the storage foundation for the trainer CLI:

- State path precedence: absolute `POKECRT_DATA_DIR`, absolute `XDG_DATA_HOME`,
  then the Linux user home fallback; missing-state reads create nothing.
- Explicit initialization with directory mode 0700 and database mode 0600.
- CGO-free SQLite (`modernc.org/sqlite v1.60.1`), pinned modules and checksums.
- Embedded schema 001, foreign keys, a 5000 ms busy timeout, DELETE journaling
  and FULL synchronization on a single-connection repository.
- Dedicated-connection immediate transactions with rollback on failure,
  cancellation or panic, and no automatic retry of an uncertain commit.
- Atomic versioned migrations with protected, nonoverwriting consistent backups
  before nonempty upgrades; corrupt, newer and unknown unversioned state preserved.

Profile commands and encounters remain unimplemented. Public print/list/help/
version do not resolve storage paths and remain usable with corrupt trainer state.
No legacy JSON storage was present, so no import routine is introduced.

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
is inferred. Storage schema version: 1 (explicit initialization only).

## Validation and performance

The public suite covers selector composition, exact artwork identity, inventory,
coverage, status/stream behavior and installed execution outside the source tree.
Storage tests cover initialization, path resolution, connection replacement,
constraints, rollback, concurrent writes, migration backups and preservation.
Upgrade tests use synthetic schema 002 fixtures; no production schema 002 exists.
Encounter aggregates and profile workflows are validated when those features land.
Tests, vet, race checks and pinned dataset checks pass for the implementation
baseline. Benchmark methods, environments, startup/memory tradeoffs and numeric
budgets are recorded in [benchmarks.md](benchmarks.md).

Intel i7 warm color random/list medians are 3.274/3.791 ms versus
12.705/13.394 ms before indexing: about 3.88×/3.53× improvement. These figures
exclude process startup and are not universal latency guarantees.

## Roadmap

Published v0.2 provides the complete public print/catalog engine for the reviewed inventory.
[Release notes](release-v0.2.md) describe its behavior and known limitations.
v0.3 covers trainer profiles, SQLite storage, encounters, progression, achievements
and private Pokédex CLI. The TUI and final hardening remain later milestones.
Those features are not implemented in the current public engine.
