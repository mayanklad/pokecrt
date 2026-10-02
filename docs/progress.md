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

D12 implements profile commands. D13 adds internal encounter operations; the
public encounter command remains pending. Public print/list/help/
version do not resolve storage paths and remain usable with corrupt trainer state.
No legacy JSON storage was present, so no import routine is introduced.

## Trainer profiles

D12 adds explicit local profiles:

- `trainer create`, `trainer list`, `trainer use`, active profile viewing and
  `trainer --name` viewing without switching.
- Names trimmed to 1–32 Unicode input code points, stored in NFC with a
  case-folded NFC lookup key; duplicate canonical names rejected.
- Atomic trainer/progress creation, first-profile activation and active selection.
  Later creation preserves the active choice, including an empty choice.
- Sorted profile lists, explicit first-run guidance and no default trainer.
- Read-only viewing/listing create no missing state; command help and syntax
  validation precede path resolution and database construction.

Profile views currently show name, UTC creation time and active status. Complete
statistics and achievement views are scheduled for D17. Profiles never create
encounters, discoveries or XP awards. Encounter recording is an internal D13
foundation until progression and output are complete.
The SQLite schema and dataset are unchanged. Unicode handling pins
`golang.org/x/text v0.42.0`.

## Encounter foundation

D13 adds internal encounter selection and atomic history/discovery recording:

- Pool derived from current metadata and exact regular-artwork inventory:
  1,017 eligible species, 1,327 forms and 2,669 regular/shiny variants.
- Separate uniform species, form and supported visual-gender draws, with an
  exact-combination 1/4096 shiny roll only when shiny artwork exists.
- Lazy pool construction and crypto/rand-backed production selection; injectable
  random source, clock and artwork preparation for deterministic verification.
- Active trainer captured before selection; decode/render preparation completes
  in memory before recording. Errors, empty preparation and cancellation record
  nothing. Successful results own the prepared artwork buffer.
- One immediate transaction verifies the captured profile, reads first-discovery
  state, inserts the metadata snapshot, and upserts species/variant counts.
  Times use MIN/MAX semantics; prior history timestamps are never rewritten.
- Exact current artwork and form metadata validated before new records; no
  palette/form/gender fallback or implicit regular discovery after shiny-first.
- Returned first flags and counts come from transactional state, after commit.
  Failed writes/commits return no successful record and are never retried.

The production CLI does not expose encounters yet. D13's internal recording
uses zero XP; progression and achievement evaluation are added in D14 before
public encounter history can be created. All five output modes follow in D15.
No database migration, dependency or generated dataset change is introduced.

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
Profile tests cover canonical collisions, creation rollback, concurrent first/
duplicate creation, selection isolation, missing-state guidance, syntax/help
boundaries, installed execution and broken pipes after committed creation.
D13 verifies history/discovery count equality, one first flag per identity,
foreign keys, and XP/history equality for its zero-XP foundation. It also covers
shiny-first/repeats, clock reversal, profile isolation, concurrent first records,
profile switching during real artwork preparation, and rollback at insertion,
upsert and deferred commit failures. XP bonus/unlock tests land in D14;
committed-output mode/stream tests land in D15.
Tests, vet, race checks and pinned dataset checks pass for the implementation
baseline. Benchmark methods, environments, startup/memory tradeoffs and numeric
budgets are recorded in [benchmarks.md](benchmarks.md).

Intel i7 warm color random/list medians are 3.274/3.791 ms versus
12.705/13.394 ms before indexing: about 3.88×/3.53× improvement. These figures
exclude process startup and are not universal latency guarantees.

D12 public startup, warm runtime, rendering and RSS checks stay within existing
thresholds. The stripped binary is 11.043 MiB against the v0.3 trainer-inclusive
16 MiB size threshold. The historical public-only 8 MiB baseline is retained;
public latency and memory limits are unchanged.

D13 public fresh-process checks pass the existing thresholds; its stripped
binary size remains unchanged at 11.043 MiB. No new warm/RSS result is claimed.

## Roadmap

Published v0.2 provides the complete public print/catalog engine for the reviewed inventory.
[Release notes](release-v0.2.md) describe its behavior and known limitations.
v0.3 covers trainer profiles, SQLite storage, encounters, progression, achievements
and private Pokédex CLI. The TUI and final hardening remain later milestones.
Storage and profiles are implemented in development source; the remaining
trainer features are pending. The published v0.2 feature set is unchanged.
