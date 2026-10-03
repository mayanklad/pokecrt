# Implementation progress

Published milestone: [v0.2 - Complete public engine](https://github.com/mayanklad/pokecrt/releases/tag/v0.2), 2 October 2026.
Release source: `59fea7c10a1de45d5732f4bd84617757e6e57490`.
Next planned milestone: v0.3 - Trainer CLI.

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

D12 implements profile commands. D13/D14 add encounter operations and progression;
D15 exposes the encounter command and its five output modes. Public print/list/
help/version do not resolve storage paths and remain usable with corrupt trainer state.
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
encounters, discoveries or XP awards. Encounter recording is available through
the D15 encounter command.
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

D14 awards XP and
checks numerical/themed achievements within that same transaction. Base XP is
10, with 40 for first species, 20 for first exact variant and 100 for shiny.
Levels advance every 1,000 XP without a cap or probability benefit. Completion
goals derive from currently eligible inventory; type evidence uses encountered
forms and branching goals require every catalog family member to be supported.
Unlocks retain their timestamp, dataset and applicable target, with unique
trainer/achievement identities. Only newly inserted unlocks are returned, in
stable policy order. Overflow and any failed write roll back the whole action.
D15 implements all five encounter output modes.
No database migration, dependency or generated dataset change is introduced.


## Encounter command and output

D15 adds `encounter [--output full|compact|no-title|achievements|sprite]`.
Full is the default. Compact shows artwork and the exact appearance heading;
no-title suppresses the identity/type block; achievements shows only artwork
and newly earned notices; sprite shows artwork alone. All modes persist the
same state and use a formatter that reads only the committed result. Progress
contains current eligible completion intersections, the committed XP award and
actual before/after level changes. Historical counts remain retained separately.
Regular-counterpart collection status is read inside the same transaction for
shiny-first guidance. No selector flags or sprite-only alias are accepted.
First-run guidance requires explicit profile creation/selection; missing state
is not initialized. Syntax and help precede path resolution. Public command
independence and existing broken-pipe behavior remain intact.
No schema, dependency, generated dataset or encounter probability changes.


## Expanded achievements

The development registry now supports 50 feasible achievements for the current
inventory. It preserves the original 34 IDs/names and adds the 16 themed goals
listed in [the achievement catalog](../README.md#achievement-catalog-in-development-source).
One ordered domain registry supplies definitions and current progress; no
presentation code evaluates awards. Unavailable policy definitions remain
nameable for retained unlocks, while impossible current goals are suppressed.

New goals cover generation/type/color/stage breadth, shared/dual-type encounters,
regional/transformation/alternate-form collecting, ordinary evolution-family
completion, source legendary/mythical/baby flags and distinct shiny collection.
Target capacities derive automatically from eligible metadata/artwork. Actual
committed form snapshots supply type and regional/transformation evidence;
repeat encounters, gender and palette differences do not inflate form counts.
New species/form exploration intersects current eligibility; numerical and
shiny-variant historical collection counts remain retained. Earned unlocks are
never deleted or re-earned after an inventory change.

All evaluation stays in the same encounter transaction. A previously satisfied
new goal unlocks on the next committed encounter, with no retroactive XP award.
No schema migration, dependency, generated dataset, new source file or CLI flag
is introduced. Private Pokédex remains D16; full trainer/achievement views remain
D17. The published v0.2 feature set is unchanged.

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
foreign keys, and XP/history equality. It also covers
shiny-first/repeats, clock reversal, profile isolation, concurrent first records,
profile switching during real artwork preparation, and rollback at insertion,
upsert and deferred commit failures. D14 verifies all six XP cases, level/overflow
boundaries, every numerical goal threshold, completion intersections, exact-form
type evidence, fully supported branching families, inventory-derived targets
against generated coverage, one-time unlocks, concurrent bonuses/unlocks and
XP/unlock rollback. Already satisfied new goals unlock on the next committed
encounter;
D15 tests exact output boundaries, identical state across modes, no repeated
unlock announcements and no successful output before commit. Write errors and
short writes preserve the committed state; broken pipes exit quietly. Installed
binary tests cover every mode, NO_COLOR/color, first-run/help/syntax paths and a
real closed pipe with exactly one committed encounter.
Expanded achievement tests cover 50 unique feasible definitions, below/at/above
thresholds, distinct form/type/shiny evidence, missing family members, impossible
goal suppression, retained metadata, per-trainer isolation, simultaneous notices,
concurrent threshold unlocks and complete rollback at a new themed insertion.
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
D14 fresh-process short-command P95 is at most 3.012 ms (8 ms limit), and
random/list P95 at most 4.979 ms (25 ms limit). Size remains 11.043 MiB.

D15 stripped size is 11.125 MiB against the approved 16 MiB limit. Public
fresh-process P95 is at most 3.093 ms for short commands (8 ms limit) and
4.589 ms for random/list (25 ms limit). Native peak child RSS is at most
8.950 MiB (10 MiB limit). Warm public/render checks pass unchanged thresholds.

The approved achievement expansion passes public startup, warm-command, native
RSS and stripped-size checks on a Xeon host: short P95 <=3.459 ms, random/list
P95 <=6.254 ms, RSS <=9.020 MiB and size 11.152 MiB. An initial largest-render
measurement exceeded budget; isolated same-host baseline/expanded checks resolve
it at 0.509/0.510 ms against 0.60 ms, with identical renderer source/allocations.
The full method and initial result are retained in [benchmarks.md](benchmarks.md).

## Roadmap

Published v0.2 provides the complete public print/catalog engine for the reviewed inventory.
[Release notes](release-v0.2.md) describe its behavior and known limitations.
v0.3 covers trainer profiles, SQLite storage, encounters, progression, achievements
and private Pokédex CLI. The TUI and final hardening remain later milestones.
Storage, profiles, encounters, progression and achievement evaluation are
implemented in development source. D16 implements private Pokédex browsing; D17 adds
full trainer statistics and achievement views. The published v0.2 feature set
is unchanged.

## D16: private Pokédex

Development baseline: `c4d171d9c5877147cea74ec96061c9690180299e`.

Summary, National list and exact entry commands now use safe trainer view models.
Hidden identities, form names, gender identities and artwork are absent from
those models until discovered. Encountered-form type conditions apply to one
form; evolution nodes obscure undiscovered names. Read-only queries retrieve
one trainer's persisted discoveries and form snapshots in a single transaction.
No schema, dependency, dataset, encounter probability or reward policy changes.

Disclosure, shiny-first/alternate-first locks, removed-artwork history, gender
locks, trainer isolation, cancellation, command parsing and unchanged database
bytes are covered by tests. Full tests, vet, race tests, CGO-free domain/storage
checks and 2947 pinned inputs pass. Public regression measurements and their
initial contaminated timing runs are recorded in benchmarks.md. D17 remains
full trainer statistics and achievement browsing; interactive views remain later.
