# Implementation progress

Current milestone: [v0.4 - Interactive Adventure Menu](https://github.com/mayanklad/pokecrt/releases/tag/v0.4).
Source: [v0.4 tag](https://github.com/mayanklad/pokecrt/tree/v0.4).
[Release notes](release-v0.4.md).

Released source: `f4464c7d60109935ccc734ff02de739c2a1ef948`.
D29–D31 navigation, Trainer scrolling and documentation updates are committed.
All pinned inputs verify. Local packaging, checksum verification and extracted
executable checks were completed before publication. The annotated v0.4 tag
points to that commit; the Linux amd64 archive and SHA256SUMS are uploaded.
Publication and tag/asset metadata were verified through read-only GitHub access.
Published archive bytes have not been independently downloaded and re-tested
in this documentation follow-up.

The final product target remains v1.0. Compact refinements are required before
that release and are being reviewed page by page;
publication does not certify every compact flow or all terminal/multiplexer
configurations. Existing timing/RSS measurements remain dated historical results.

Earlier entries below record status at each increment; their pending release
items are superseded where completed by the publication record above.

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

D17 profile views include name, UTC creation time, active status, full statistics
and achievement progress. Profiles never create
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

The achievement registry supports 50 feasible achievements for the current
inventory. It preserves the original 34 IDs/names and adds the 16 themed goals
listed in [the achievement catalog](../README.md#achievement-catalog).
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
is introduced. Private Pokédex and trainer/achievement views are implemented in D16/D17.
The published v0.2 feature set is unchanged.

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

v0.2 provides the public print/catalog engine for the reviewed inventory.
v0.3 adds profiles, SQLite storage, encounters, progression, 50 achievements,
private Pokédex and trainer statistics. [Trainer release notes](release-v0.3.md)
describe the complete CLI milestone. The TUI remains v0.4; final acceptance,
upgrade and resource audits remain v1.0 work.

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
initial contaminated timing runs are recorded in benchmarks.md. D17 implements
full trainer statistics and achievement browsing; interactive views remain v0.4 work.

## D17: trainer statistics and achievement browsing

Development baseline: `9678c29edfa80cbcd0bd3bbbdb14c726f49d17f8`.

Trainer summaries now include XP/level, next-level progress, encounter and
collection counts, repeated versus distinct shiny counts, UTC first/last times,
current eligible completion and generation discoveries. Named viewing preserves
the active trainer. Historical identities remain counted if artwork or metadata
is removed; unknown generation metadata is not fabricated.

`trainer achievements` groups earned and locked entries in the existing registry
order. Earned dates/targets persist; newly satisfied definitions remain locked
until the next committed encounter. Unsupported locked goals are suppressed,
while previously earned definitions remain readable. Safe models contain counts
and generic descriptions, without unseen species or form identities.

Reads share existing discovery/evidence queries and use one consistent SQLite
transaction. No new migration, dependency, dataset or achievement policy is
introduced.

Full tests, vet, race, CGO-free domain/storage and pinned-input checks pass.
Fixtures cover repeated shiny encounters, empty and epoch-zero timestamps,
named/active isolation, removed inventory, retained unlocks, pending goals,
concurrent encounter snapshots and unchanged database bytes. Installed binaries
exercise summaries, achievements, first-run/help and quiet broken pipes.
Fresh public startup and size checks pass existing thresholds; benchmarks.md
records the method. D18 integration, offline and history-performance results
are recorded below.

## D18: trainer CLI integration and release documentation

Source baseline: `43e0a34b35a792ad268e40407aa68e863f0aa763`.

Achievement checks reuse persisted form flags and distinct shiny discoveries;
old-predicate equivalence tests retain historical snapshot semantics. Schema 1,
indexes, dependencies, dataset, encounter probabilities and reward rules are
unchanged. History-growth benchmarks record remaining scan costs and workload
limits. Dex no-active guidance now lists existing profiles before selection.

Migration fixtures now preserve actual gameplay state through upgrade/backup and
rollback. Installed-process concurrency, full tests/vet/race, CGO-free checks and
all pinned inputs pass. An extracted release executable passes socket-denied
public/trainer CLI, all five modes, discovery locks, profile isolation, read-only
browsing, installation, Bash piping and corrupt-state independence.

Public startup, warm-command, rendering, extracted native RSS and stripped size
meet the existing budgets on this host. Initial RSS measurement sensitivity and
all methods are retained in benchmarks.md. Fastfetch raw-input piping is verified on a separate Linux amd64 desktop.
Fastfetch is unavailable on the benchmark host; JSONC command-raw examples are
checked against official documentation, without claiming a runtime check.

The v0.3 archive includes the executable, product documentation, release notes,
coverage and licensing notices. It contains no trainer state. The interactive
TUI remains the v0.4 milestone.

### Player-facing text review

Achievement requirements, root/subcommand help, profile statistics, Dex notices,
name validation and artwork errors use player-facing language. Source
classification and storage/evaluation details remain in technical documentation
where they explain the rules. Achievement IDs, names, requirements and retained
unlock behavior are unchanged. All-time counts remain distinct from current
artwork collection progress. Single-command section spacing and singular
encounter labels are consistent; no trailing spacing is added to separate
independent commands. Full CLI/domain tests and installed execution cover the
updated wording and presentation.

## Previous releases

[v0.2 - Complete public engine](https://github.com/mayanklad/pokecrt/releases/tag/v0.2)
was published 2 October 2026. Its source is
[`59fea7c10a1de45d5732f4bd84617757e6e57490`](https://github.com/mayanklad/pokecrt/commit/59fea7c10a1de45d5732f4bd84617757e6e57490).
It contains the public print/catalog engine; trainer gameplay is included in v0.3.

## D19: Adventure Menu interface shell

Implemented against `0fc3efd8feefd56be46bdbcd981773e42c097d4e`. The chosen
visual direction is Adventure Menu: a Pokédex device, dialogue and four-section
menu. `pokecrt tui` starts the interface explicitly; the root command still
prints help. D19 is the interface foundation, not the completed v0.4 product.

The shell implements arrows, Tab/Shift+Tab, Enter/Space, j/k, clickable controls
and mouse-wheel focus navigation. Wide, compact, short and too-small layouts
preserve section/focus on resize; mouse hit targets derive from the displayed
frame. Small-screen Quit remains clickable. Settings apply Dark, Light, Follow
Terminal or Terminal Native immediately for the session. NO_COLOR keeps focus
and selected mode visible without styling. Terminal Native leaves default
colors untouched; transparency belongs to terminal configuration.

Follow Terminal uses asynchronous background-color queries with a 750 ms
response timeout. Successful replies enable a two-second focused refresh chain.
Unsupported queries fall back to native colors and stop periodic queries;
focus regain/reselection retry detection. Stale timeout/poll/load messages are
ignored. No global terminal palette is overwritten or desktop theme inferred.

Profile status is read asynchronously through storage.ReadOnly and closed in
the worker. Missing storage stays absent; existing/corrupt databases are not
modified. Errors leave shell controls usable. The initial design used the
Pokédex device alternative and a clean `?` screen. Landscape and
pixel Poké Ball decoration were removed. No hidden identities
or uncollected sprites are displayed, and no gameplay action is connected yet.

Next: D20 interactive welcome/create/select and settings; D21 safe Pokédex; D22
encounters/history/statistics/achievements; D23 complete scenario, resource,
terminal compatibility and release checks. Bubble Tea v2.0.10 and compatible
exact dependency versions are pinned. Dependency notices for the final v0.4
binary must be reviewed before release packaging.

Fresh full tests, vet, race, CGO-free tui/trainer/storage and 2,947 pinned-input
checks pass. PTY exercises verify arrow/mouse/resize/Ctrl+C and restoration;
model tests verify all shell controls and disclosure-safe neutral framing.
Public latency/RSS, warm rendering and binary size remain within existing
budgets in the initial D19 measurements. The revised device binary is 12.633 MiB;
its 120×36 frame assembly median is 0.817 ms in three shared-host samples.
Detailed measurements and limits are in benchmarks.md. Real terminal theme
changes/transparency still require real-terminal verification; network isolation was
unavailable on this host and is not newly claimed.


## D20: interactive trainer setup, selection and saved appearance

Baseline: committed D19 `742ac3248883ccacf396c1533deff0ebb65ad560`.
The approved Pokédex device and Terminal Native foreground/accent behavior are
preserved. Fresh setup opens a name form; profiles without an active selection
open the chooser; active trainers enter the Adventure Menu. Trainer opens
chooser/create controls. Existing domain parsing and storage create/use methods
are reused. Only the first profile activates on creation. Additional creation
selects the new row but offers a separate Use action.

Typing/paste supports Unicode names, cursor editing and literal shortcut letters.
A three-page on-screen keyboard and clickable field cursor support mouse-only
creation/editing. Lists scroll around selected rows, including long lists; short
lists use a compact panel. 40×12 remains usable; smaller windows retain Quit.
Resize preserves form text, cursor, focus and selection.

Profile actions run asynchronously with one in-flight request. Validation happens
before initialization. Navigation/cancellation before submission creates no
state. Cancel during a request cancels its context and refreshes from storage;
a committed result is retained, never retried automatically. Busy controls cannot
queue duplicate writes, and stale reads/results cannot replace the new selection.
Corrupt/newer trainer state is not recreated. Setup has no gameplay write path.

Appearance remains live; explicit Save remembers it in a tiny versioned
appearance.conf with protected atomic replacement. Startup honors explicit flag,
then saved choice, then Follow Terminal. Startup reads cannot replace a newer
manual choice/save. Invalid settings remain unchanged and errors are recoverable.
Help/public commands/redirection do not read or write either store. No new
module dependency, trainer migration or generated-data change is introduced.

Full tests, vet and race checks pass, with CGO-free UI/domain/storage checks.
Terminal runs cover keyboard/mouse creation and selection, Unicode paste,
empty/invalid names, first-run cancellation, live/saved appearance and flag
precedence, resizing, minimum size, Native/NO_COLOR and terminal restoration.
Database checks verify no encounters, discoveries or achievement unlocks.
Resource measurements and the noisy-host startup limitation are in benchmarks.md.

Next: D21 disclosure-safe interactive Pokédex; D22 encounters/history/trainer
statistics/achievements; D23 complete integration, resources and release review.

D20 navigation correction: arrow keys enter and navigate every on-screen letter row; Enter preserves letter focus. Tab reaches all letter keys. Trainer list arrows leave at boundaries and reach the paired action buttons. Regression coverage includes wide, compact, and minimum supported terminal dimensions.

D20 full TUI review correction: focus-specific setup guidance replaces the fixed keyboard hint; mouse actions retain their visible focus, and returning from Appearance restores setup focus. Appearance displays navigation guidance. Trainer wheel scrolling stops at boundaries; refresh clamps stale selection. Busy name fields allow vertical focus movement. Unsupported small windows accept only visible Quit, preventing hidden mutations. Reviewed keyboard/mouse setup, two-column controls, all appearance palettes, NO_COLOR, three keyboard pages, resizing, asynchronous operations and terminal restoration.


## D21: interactive disclosure-safe Pokédex

Baseline: committed D20 `5ad62fd5d1e896eec24cb360e685038052f7e176`.
Adds National list, All/Seen/Unseen and generation filters, safe name/number
search with mouse-accessible on-screen keyboard, entry facts and counts/times,
known form/gender/palette selector, original-size artwork scrolling, and clickable/
keyboard-accessible evolution nodes. Wide list/detail panes and compact switching
preserve selection, active pane and focus on resize down to 40×12. Existing
appearance modes, setup behavior and Adventure Menu styling are preserved.

Domain Dex produces selector options for observed forms/genders only. Missing
palettes remain locked; removed collected assets retain history with a notice.
Unsupported exact selections retain safe species facts for UI recovery while CLI
error behavior remains unchanged. Hidden names cannot match search. No catalog
identity join or sprite fallback is introduced in presentation.

Storage reads use existing ReadOnly/current-schema APIs. Data loads and exact
sprite decoding/rendering are commands with stale-result guards. Reopening clears
trainer-scoped data. Public command paths never construct the Dex. No schema,
dataset, dependency, encounter/XP/achievement rule or asset pin changes.

Full tests/vet/race pass; frame/mouse/disclosure tests cover all modes, NO_COLOR,
all keyboard pages, list/picker/detail/search, minimum and undersized windows,
async results, trainer isolation and unchanged database bytes. Terminal review
covers keyboard and mouse-only search/artwork, exact locks, evolution, creation,
resize, Native/NO_COLOR, live Follow Terminal and terminal restoration. Resource
measurements and unresolved shared-host timing gates are in benchmarks.md.

Next: D22 explicit encounters/history, trainer statistics and achievement views;
D23 full integration, terminal compatibility, resource and release review.


D21 approved device redesign: replaces the initial long-report presentation with
a framed Overview plus separate Variants, Evolution and Records tabs. Status
filters, selected tabs and footer actions have consistent visual grouping. Wide
index/detail panes and compact two-row tabs share one frame geometry; short
windows retain a View cycle control. Compact filters can be opened and closed
with a mouse. Arrow navigation follows visible rows/columns; Tab reaches all
controls and safe evolution nodes without duplicate focus targets. Native sprite
proportions and exact collection locks are preserved. Tab switching does not
perform a data read or sprite render, and an explicitly selected appearance
survives filtering when the same entry remains selected. Times remain precise
UTC records. Supersedes the earlier D21 layout ZIP; the complete replacement is
against committed D20 `5ad62fd5d1e896eec24cb360e685038052f7e176`.


## D22: trainer activity and progression views

Baseline: committed D21 `a33df3f7e1c7284ff7c44f8cf603ad470b7df5b1`.
Implementation baseline: D21.

Connects Encounters, Trainer and Achievements to the existing domain services.
Explicit encounters use the shared atomic engine, exact sprite preparation,
captured trainer identity, an in-flight guard and recoverable errors. Browsing,
scrolling, resizing and appearance changes never record an encounter. History
reads at most 50 indexed rows using persisted identities and stable time/ID
ordering; selected rows open the exact safe Dex appearance. Current-schema
read-only workers supply statistics and disclosure-safe achievement views.
Opening a screen clears previous trainer state; refreshed results from another
trainer clear old artwork/results. The Adventure Menu includes trainer level.

Wide cards separate trainer facts/generation progress and earned/locked goals.
Compact screens scroll and offer artwork/Info switching without scaling pixels.
Keyboard and mouse controls share geometry. Screen transitions request one clear
redraw after terminal review found retained frame fragments. Existing live
appearance modes and setup controls remain unchanged. No new dependency,
dataset change or schema migration is introduced.

Validation: full tests, vet and full race suite passed; affected TUI/storage race
checks passed after final view changes. New tests cover explicit-only writes,
duplicate requests, recoverable failure, stale messages, read-only/no-init
behavior, trainer isolation, exact history appearance, and 50-row snapshot
ordering/isolation. Device resource measurements are recorded in benchmarks.md.
The remaining arrow-navigation refinement is deferred to D23 as scheduled.

D22 final interaction refinement: navigation remains available during an encounter;
a model-wide single-flight token blocks duplicate encounters even after leaving
and reopening a section. Completion refreshes the current screen without showing
another trainer’s result. Minimum-height views reserve separate content and
control rows so controls cannot overwrite records/artwork.

D22 terminal verification: 21 running PTY scenarios passed, including mouse-only
encounter/history entry navigation, compact/minimum layouts, NO_COLOR artwork,
resize, live Follow Terminal replies and appearance return. All exits restored
terminal state. The five explicit test encounters added exactly five database
rows; the separate read-only browsing/theme/profile-chooser cases left the
seeded database byte-identical. Real terminal translucency and multiplexer
compatibility remain manual checks for D23.




## D23 visual controls and appearance

Flat rounded buttons use purpose colours in outlines and text, without surface
tints or shadow edges. Focus and selection have separate markers. Action groups
use bounded widths. The home town map uses static terminal characters; the
adjacent description follows the focused activity. Refresh and Quit occupy the
home footer. Search and trainer creation use one contextual hint strip.

Scroll and artwork controls appear only when their content exceeds the available
space. Decorative dot separators are removed. Terminal Native preserves the
terminal background. Light mode uses warm surfaces and darker accents.

Validation covers Go tests, vet, race checks and actual terminal scenarios.
Remaining release performance and terminal compatibility gates are separate.

## D23 control and frame corrections

Filter widths reserve focus/selection marker space; labels remain centred when
focus changes. Narrow filters use a status cycle and generation selector.
Search and Appearance use full outlined Quit controls where height permits.
The National Index has a contrasting focused row, with inverse terminal colours
in Native mode and a pointer in NO_COLOR. Encounter artwork and details have
separate complete frames and a consistent gutter. The home trainer card fits
its contents without overlapping the logo. Scroll controls use bounded padding
and centred arrow glyphs. Structural borders are subdued; the panel containing
keyboard focus receives a restrained accent. Panel titles remain readable.

Validation: full Go tests, vet, TUI race checks and actual terminal cases cover
wide/compact/minimum layouts, all appearance modes, focused filters, input,
resize, mouse Quit and recorded test encounters in an isolated database.
Browsing leaves the reference database unchanged and exits restore terminal
state. No animation, polling, image assets or dependencies are added.

## D23 Search arrow access and Dex label alignment

The accepted control/frame styling remains the visual baseline. Right Arrow
from the top-right on-screen keyboard key or a rightmost editing action reaches
header Quit. Input Left/Right still edits the cursor; narrow layouts retain the
footer route from Appearance to Quit. Navigation uses visible hit geometry.

Pokédex footer actions and tabs use widths that allow equal whole-cell padding
around labels. Labels retain their centre when focus/selection markers change.
Other screens, colours, frames and the town map are unchanged.

Validation includes Go tests, vet, TUI race checks and running-terminal scenarios,
with an explicit arrow-only Search Quit exit. Reference trainer data remains
unchanged by browsing, and terminal state is restored on exit.

## D23 Search footer, trainer hints and History spacing

Search Quit is in the lower action area alongside Appearance on roomy layouts;
short layouts keep Search, Cancel and Quit together in the lower action row.
Input Up stays in the field, while Left/Right retain cursor editing. The Search
header Quit route described in the previous checkpoint is superseded.

Trainer-list focus guidance includes Left/Right to reach the action buttons,
including narrow layouts. History Open, Previous and Next controls fit their
labels with bounded padding instead of stretching across the available width.
The accepted colours, frames, town map and remaining controls are retained.

Validation covers Go tests, vet, TUI race checks and running-terminal scenarios,
including wide/compact/minimum Search, footer Quit, trainer-entry hints and
History. Browsing leaves reference trainer data unchanged; exits restore
terminal state. No new timers, background work or dependencies are introduced.

## D23 Header and navigation guidance

Wide Pokédex headers allocate trainer names from the space remaining before the
right-aligned discovery count. Discovery counts use ordinary integers; Dex
entry identifiers retain their padded numbering.

Text page headers use the same red POKÉCRT branding. The home wordmark uses the normal E; other page headers retain É. Navigation guidance uses muted
single-line bracket keycaps, separated vertically from the action controls.
Action-button geometry, frame treatment and the town map remain unchanged.

Validation covers Go tests, vet, TUI race checks and terminal scenarios across
wide, compact and minimum layouts, appearance modes, keyboard and mouse exits.
No dependencies, timers or background work are added.

## D24 Full-terminal information presentation

The TUI canvas and Home, Pokédex and activity frames follow the terminal extent
instead of centering a 120-column by 40-row page. The prior 240-column by 100-row
canvas ceiling is removed. Setup and Appearance remain bounded dialogs within
the full canvas. Resize updates use the same geometry for painting and targets.
The National Index retains its existing width, row treatment and disclosure.

Information uses grouped headings, aligned label/value fields, restrained type
colours, local readable timestamps and count/progress bars. Pokédex Overview
separates artwork from field notes; Records groups discovery, species, observed
forms and collection progress. Trainer and achievement pages present progress
and goals in responsive columns. Encounter rewards and saved records have
separate sections. History entries have spacing and wrapped-row target tracking.

Evolution uses responsive cards with collected artwork and anonymous placeholders
for uncollected nodes. Parent links determine presentation order, including babies
with later National Dex numbers. Direct adjacent relationships have arrows;
child identifiers preserve branches without implying unrelated adjacency. Card
rows remain clickable and keyboard focus reveals the selected node. Artwork is
prepared in the existing asynchronous entry worker and retained only with that
entry; generation checks reject stale results. No View-time image decoding,
network access, dependency, timer or background loop is added.

The current catalog supplies family links but no evolution conditions or levels,
base stats or abilities. These facts are not fabricated. Additional verified
metadata is a separate dataset extension. Home artwork redesign remains deferred;
its current design is retained in the larger frame.

Validation includes tests, vet, TUI race checks, full-canvas extents through
320 by 120, collection privacy, NO_COLOR, family order and real-terminal checks
for normal/compact/minimum layouts, resizing, appearance modes and exits.

### D24 redraw samples

Brief 100 ms Go microbenchmark samples on linux/amd64, Go 1.27.1,
AMD EPYC 9V74, measured View construction rather than terminal I/O:

| View | Previous 120-column layout | D24 120-column layout |
| --- | ---: | ---: |
| Home | 0.63 ms | 0.69 ms |
| Pokédex Overview | 0.91 ms | 1.19 ms |
| Trainer | 0.51 ms | 0.69 ms |
| Achievement goals (50) | 0.61 ms | 1.35 ms |
| History (50) | 0.83 ms | 1.01 ms |

The 180 by 60 Trainer sample measured 1.14 ms and about 843 KB allocated per
View. Richer formatting adds processing/allocation cost. Larger terminal areas
require more cells; these samples are not end-to-end latency guarantees.

## D24 Evolution, appearance and achievement navigation

Evolution cards accept arrow navigation from the index or tab row, with explicit
card focus, geometry-based movement and predictable exits. Placeholder symbols
and captions centre independently. National Dex identifiers remain visible for
anonymous family nodes. Opening one shows a locked entry; Family or Escape
returns to the originating family and its prior appearance selection.

Variants use outlined appearance cards with collection status, current-selection
markers and keyboard focus. Card height and wrapping determine scroll reveal;
keyboard and mouse activation retain exact appearance selection.

On wide Achievements layouts, Earned Badges and Next Goals have independent
scroll offsets, focus borders, Scroll targets and per-panel arrow controls.
Left/Right switch panels; wheel scrolling uses the panel beneath the pointer.
Compact layouts retain the single combined journal and its existing scrolling.
No new timer, background loop or dependency is introduced.

## D24 Achievement wording and panel exits

Event-based locked goals without a numerical target show “Not yet earned”.
They remain earnable goals; absence of a progress bar does not mean that the
encounter inventory cannot support them. Earned Badges Left exits to Back;
Next Goals Right exits to Quit. Inward arrows still switch between the panels.
These routes preserve both independent scroll positions.

## D25 Home town illustration

The Home town artwork uses pitched-roof landmarks, window bays, doorways,
a central avenue, a location marker, tree silhouettes and a garden pond.
Three compositions adapt to the available artwork viewport; decorative strokes
are clipped to that viewport. The illustration uses terminal characters with
separate dark/light colours and terminal-native/NO_COLOR fallbacks. It adds no
image assets, dependencies, animation timers, navigation targets or storage writes.
Existing Home controls, layout and the accepted information panels are retained.

Validation: go test ./..., go vet ./... and build pass. Real terminal captures
cover wide, large, compact, short and minimum layouts, Dark, Light, Terminal
Native, Follow Terminal with a light-background reply and NO_COLOR. Terminal
state restores after exit; the trainer fixture remains unchanged.

The Home Choose frame fits its five options with no unused interior row below
Appearance, in both wide and compact layouts. The information panel keeps its
existing height, and control positions and navigation targets are retained.

On wide Home layouts, the information panel and Choose frame share the same
seven-row height and aligned lower borders. Information wrapping stays within
the resized interior. Compact stacked panels retain their existing heights.


### D25 - adaptive home town artwork refinement

- Town streets span the artwork viewport with connected avenue and building entrances.
- Landmark sizes and positions adapt to the viewport; outer lawns contain trees and the pond has aligned continuous borders.
- Home title receives one blank interior row above it on normal layouts. The information and Choose frames retain equal heights.
- Compact maps retain a simpler composition. No image assets, timers, or new focus targets.
- Build and running-terminal verification pending: Go is unavailable in the current workspace.


D25 accepted-artwork follow-up: removed the Verdant Town label in both map layouts. Replaced the obsolete half-block prohibition with route/location, removed-label, and viewport-containment checks; trees remain unchanged. Go execution remains pending in this workspace.


D25 resize follow-up: switch to a four-landmark compact map before detailed buildings fall below their minimum width or height. Short viewports show condensed labels, including Home and Pond. Added landmark-persistence regression cases across layout thresholds and one-row panels. Large artwork remains unchanged. Go execution pending.

### D26 - compact-mode audit fixes (ddc0d811 baseline)

Compact pages now reserve separate content, tab/action and hint rows across
40×12 through the wide-layout boundary. This resolves overlapping captions,
card content, footer frames and mouse targets without changing wide rendering.

- Pokédex: visible evolution exits, resize/overlay return-focus reconciliation,
  complete scrollable Art/Facts views, transparent-margin trimming in compact
  family cards, Full art opening the displayed collected appearance with Family
  return, selected card borders preserved, and empty-search recovery controls.
- Encounters/history: active Art/Details controls and wheel scrolling, explicit
  return labels, wrapped errors, and usable compact footer labels.
- Trainer, achievements and dialogs: reconciled scroll budgets, monotonic trainer
  list capacity, full Help/Status access for lengthy explanations/errors,
  context-specific hints, and visible NO_COLOR focus for tightly fitted buttons.
- Expanded compact filters expose one Search target. Family artwork rejects a
  stale card selection from another species. Accepted Home artwork is retained.

Validation with Go 1.27.1: go test ./..., go vet ./..., go build, and
race-enabled internal/tui tests pass. Added regressions for content/control
separation, filter targets, facts, resize and overlay focus, evolution exits,
empty results, exact collected artwork, stale family selection, trainer list
capacity, wheel routing, full errors and NO_COLOR focus.
264 deterministic wide renders match the baseline byte for byte, covering
100×24, 120×40 and 180×60 across four appearance modes and NO_COLOR.
98 running-terminal scenarios cover compact sizes, resize, filters, tabs,
forms, profiles, encounters/history, Help, empty/locked states and undersized
windows. Exit/terminal restoration checks pass; read-only fixture browsing
leaves the trainer database unchanged. Encounter writes use disposable fixtures.
These checks do not claim exhaustive platform or async-event coverage.

Delivery: changed-file ZIP against ddc0d811; no deletions, binary/assets,
dependency/schema changes or GitHub writes. Apply at repository root and run
the normal test/vet/build commands before committing.

### D27 - Home header actions and taller Town Map

Against a9136edc, wide Home (at least 90×28) puts Refresh and Quit in the
top-right header above a single-line trainer card. The card shows trainer name
and a gold Lv. value separated from the name by two spaces without a middle-dot separator. Loading and
no-trainer states remain readable. Header widths respect the central logo.
The trainer card sizes to its text and stays right-aligned, with long names
truncated within the header budget. Refresh/Quit retain their outlined frames.
Town Map gains four rows against the committed baseline by reclaiming the
lower action area. Information/Choose panel heights, title gap and bottom
keyboard hints remain unchanged. Narrow Home retains the lower action layout
with the one-row map expansion; layouts below 28 rows remain unchanged.
No artwork, storage, dependency or other-screen changes.
Validation: go test ./..., go vet ./... and build pass. Running-terminal checks
cover 120×40, 90×28, 64×40, 48×28 and 40×12; exit restores terminal state.
This package replaces the earlier D27 taller-map spacing ZIP in full.

D27 Encounter Log follow-up: use the full text viewport for READY TO EXPLORE
and wrapping when no artwork panel is displayed. Apply the half-width budget
only when the split artwork/details view is actually rendered. Regression
coverage checks full and split heading widths at 100, 120 and 180 columns.
Tests, vet and build pass; wide running-terminal capture verifies the divider.
This cumulative ZIP includes all uncommitted D27 Home and Encounter changes
against a9136edc and replaces every earlier D27 ZIP.


## D28 - release blocker audit

Restored independent wide Trainer/Achievements column widths after the D27
Encounter Log fix accidentally made them depend on artwork state. Preserve the
full-width READY TO EXPLORE behavior. New tests cover both panel
headings at 100/120/180 columns with/without encounter art.
README reflects implemented Home/setup/activity views, actual full-terminal
geometry and known compact limitations. Candidate v0.4 notes are included.
The published release remains v0.3; v0.4/v1 clearance is not claimed.

All 112 tracked baseline files matched GitHub. Go 1.27.1 test/vet/full race,
module verification and CGO-disabled storage/trainer tests pass. Embedded
asset/hash/coverage tests pass. 34 socket-blocked offline executable cases,
seven wide PTY captures and idle restoration checks pass. Measurements meet
existing public process/warm/render/RSS and approved 16 MiB size thresholds
on this host; representative history timing is recorded without new budgets.

The old available pinned-input cache is incomplete; raw regeneration and
package.sh remain pending, as do final archive/tag and real-terminal acceptance.
Reported remaining compact issues are not individually classified; wide
layouts are recommended and further refinement stays deferred to v2.
Only a reproduced crash, blocked essential action, disclosure or integrity
defect requires current-release correction. No gameplay/schema/budget changes
or GitHub writes. Next development step: complete source-cache verification
and final milestone archive smoke tests before publication.

Local document organization: the specification and detailed release audit are
under docs/local/, ignored as a folder. Progress, benchmarks and release notes
remain tracked; repository documentation has no links to the local audit.


## D29 - wide Home arrows follow header action placement

Baseline: 4f614a5e3620a647521273bda670aca5e9f7404b. At wide Home sizes
(90×28 and above), Up from Pokédex focuses Refresh; Left/Right moves between
Refresh and Quit; Down from either returns to Pokédex. Up at the header stays
there, and Down at Appearance stays at the bottom of Choose. j/k retain the
same directional routes. Tab and narrow footer routes remain unchanged.

Regression tests cover 90×28, 120×40 and 180×60 wide layouts and 89×40,
64×28 and 48×28 narrow layouts. Full tests, vet and build pass. Three
running-terminal captures verify Refresh, Quit and Choose focus; terminal
restoration passes. No artwork, layout, domain or storage changes.


## D30 - source-cache and local v0.4 candidate verification

Baseline: c2712017b0785e07f951f98301ef9b1e6b1d59a9. All 113 tracked files
matched GitHub before edits. Source comparison used read-only GitHub access. The stale TUI help
statement about forthcoming section views now describes the implemented views
and recommends wide layouts. Release notes reflect completed cache verification.

All 2,947 pinned inputs were downloaded and verified. Cache-only preparation
and checks pass with the existing dataset ID; generated tracked metadata has
no drift. The actual package.sh v0.4 command passes dataset checks, tests and
vet. Full race tests, module verification and CGO-disabled storage/trainer
tests also pass.

The local, untagged Linux amd64 candidate contains exactly the executable and
seven documented notice/readme/coverage/release files. SHA256SUMS, version and
dataset identity, 0755 executable mode and installation into an isolated folder
pass. No profiles, cache or source tree are present in the archive. The stripped
binary is 13,705,376 bytes (13.07 MiB), below the approved 16 MiB limit.
34 extracted-executable CLI cases pass with socket/connect denied by kernel
seccomp. Trainer encounter counts remain isolated (5/1); browsing preserves the
state hash and corrupt input remains untouched. Ten offline PTY cases cover
Home, Trainer, Achievements, locked/variant Pokédex, setup cancellation,
appearance, header arrows and Home/Pokédex resize recovery. Exit, terminal/mouse
restoration and unchanged browsing state pass.

Pinned-source regeneration and local candidate artifact checks are complete.
Final tagged-source packaging and real-terminal acceptance remain pending.
The candidate is unpublished; published v0.3 links remain unchanged. Remaining
compact refinement stays deferred to v2, without certifying all compact flows.
Gameplay, schema and performance budgets are unchanged.


## D31 - independent Trainer panels and neutral documentation

Baseline: c2712017b0785e07f951f98301ef9b1e6b1d59a9; D30 updates are retained.
Wide Trainer Card and Generation Progress now use independent scroll offsets,
focus targets and arrow buttons, matching the Achievements panel behavior.
Keyboard arrows/j/k, Page Up/Down, mouse wheels and clickable controls target
one panel at a time. Profile changes reset panel offsets; refresh and resize
clamp them to the available content. Compact mode retains its combined view.

Documentation describes project behavior and verification without conversational
approval/decision attribution. Licensing and distribution statements retain their
substantive scope. Generated notices and their generator/source terms agree;
third-party license and contributor quotations remain unchanged.

Full tests and vet pass, as do race-enabled TUI tests. Regression tests cover
independent offsets and rendered content at 100/120/180 columns. Fourteen
socket-blocked PTY cases pass, including Trainer arrow/page navigation and
resize recovery. Browsing preserves the database hash; terminal attributes,
alternate screen and mouse reporting restore on exit. All 2,947 pinned inputs
verify with the unchanged dataset ID and generated output check.

The D31 stripped Linux amd64 executable is 13,705,376 bytes (13.07 MiB).
D28 timing/RSS measurements and D30 archive smoke checks remain historical
measurements of those revisions, not fresh D31 results. Final D31 archive
packaging and tagged-source/real-terminal acceptance remain pending.



## D32 - published v0.4 documentation

v0.4 was published on 6 October 2026 (IST), from
f4464c7d60109935ccc734ff02de739c2a1ef948. README links now point to the
release, source tag, Linux amd64 archive, checksums and current release notes.
Current status reflects publication; historical benchmark and audit entries are
retained with their measurement scope. This update changes documentation only.
The existing tag and uploaded archive remain unchanged.


## D34 - Home screenshot review and responsive corrections

Baseline: cb2385c2f16cfbcab206c0a667e45442848696ea. A fresh source copy
matches every tracked GitHub blob before edits. Compact/lower-size refinements
remain in the v1.0 scope; earlier v2-deferral statements describe historical
plans. The published v0.4 tag and archive remain unchanged.

The four Home screenshots were reviewed. Large-logo clearance now reserves a
gap from Refresh/Quit at intermediate widths and preserves the centered position
where it fits. YOU and ROUTE signs have blank padding separating them from road
lines in both map compositions. Short-screen Refresh/Quit are grouped in the
top right at 44 columns and 16 rows or larger. Below that width they sit beneath
the trainer heading; below 16 rows they use single-row controls. All seven Home
actions remain visible and clickable at 40×12. Narrow hints show directional
arrows, Tab, Enter and Quit across two lines.

Header navigation follows placement: Up from Pokédex reaches Refresh; Left/Right
switches Refresh/Quit; Down from either returns to Pokédex. Appearance Down and
header Up stop at the group edges. j/k retain directional routes. Taller narrow
layouts retain footer actions and their navigation.

Full tests, vet, TUI race checks and build pass. Regression tests cover logo
clearance at 90–130 columns, short-header text clearance at 44–89 columns,
map-sign padding and short-screen arrow/j/k/Tab/mouse reachability. Twenty-three
network-blocked Home PTY scenarios cover 40×12 through 120×40, screenshot-like
94×36/94×32/70×40/80×24 layouts, header navigation and resizing. Database hashes
remain unchanged; terminal, alternate screen and mouse reporting restore on exit.
No other page layout, gameplay, schema, dataset, dependency or timer changes.
This is the first Home refinement pass, not complete compact/v1.0 acceptance.


## D35 - Achievements focus and responsive controls

Baseline: 3986f0c4dc3f9fe997fc20fc0ac4387f02a641cc. Every tracked
source blob matched the GitHub baseline before edits.

Achievements uses one focus marker on the selected control. Color layouts
highlight the journal or panel border for Scroll and both arrow controls;
frame titles have no duplicate marker. Monochrome layouts retain one visible
control marker without ANSI styling. Horizontal navigation follows each visible
panel's Scroll, Up and Down controls, then Back, Theme, Refresh and Quit.
Left from the first Scroll reaches Quit; Right from the last panel control
reaches Back. Tab/Shift-Tab follow the same control order. Up/Down on Scroll
and mouse-wheel scrolling retain independent panel positions.

Footer hints use bracketed key notation, consistent action names and two
reserved rows at all supported sizes. The scroll hint follows the focused
control. Compact action labels are centered; outlined actions are retained
when at least 56 columns and 24 rows are available. Smaller layouts use
single-row actions. At fewer than 16 rows, compact journal text omits blank
separator rows so the first badge remains visible at 40×12.

Full Go tests, vet, TUI race checks and executable build pass. Regression tests
cover control order, footer clearance, exact rendered dimensions, one focus
marker in color and monochrome, active panel styling and independent arrow
activation. Forty-five network-blocked PTY scenarios cover 40×12 through
120×40, every panel control and footer boundary, resizing, light appearance
and NO_COLOR. Stored trainer data remains unchanged; terminal attributes,
alternate screen and mouse reporting restore on exit. Historical performance
measurements are retained; no new benchmark or release claim is made.

Compact refinement remains required before v1.0. This increment addresses
Achievements; remaining pages still require their individual review.


### D35 screenshot follow-up

Achievements hints now use a single row when the complete hint fits the
available width, otherwise two rows. The two reserved footer rows prevent
clipping at smaller sizes. Single-row control interiors overwrite underlying
frame lines, preserve both delimiters, center labels with symmetric padding,
and place the sole focus marker inside the control. Arrow controls have a
blank separating column. The approved Home layout remains unchanged.

Full tests, vet, build and TUI race checks pass. Additional regression tests
check border-free control interiors and one-row/two-row hint selection. Six
additional offline PTY scenarios cover minimum, screenshot-like short and
large screens; browsing leaves trainer data unchanged and exit restores the
terminal. These checks supplement the preceding D35 scenarios.


### D35 footer anchoring

Achievements hints are anchored immediately above the bottom frame border,
with a horizontal divider separating them from content and action buttons.
Arrow directions share a single Move label whenever they all move focus;
Scroll focus retains separate vertical Scroll and horizontal Move hints.
Compact content and action positions reserve the divider and hint rows. At
the minimum height, the journal omits its summary heading so a badge remains
visible. Regression tests verify bottom anchoring and combined Move hints.


### D35 Home footer consistency

Home uses the same bottom-anchored hint area and horizontal divider as
Achievements. Key names have no internal bracket padding: [Enter], [Tab],
[Esc] and [Q]. Hints use one row when they fit, otherwise two. At 40×12 the
trainer summary shares the title row so all seven actions remain above the
footer divider. Narrow footer buttons and dialogue reserve the divider row.
Home arrow navigation and Town Map artwork remain unchanged.

Full tests, vet and build pass. Home regression coverage checks footer
anchoring, key formatting, all seven visible hit targets and divider clearance
across ten layouts from 40×12 through 120×40. Six additional offline Home PTY
scenarios verify actual rendering and terminal restoration without state changes.


Footer dividers now use the same quiet structural-border style as the outer
frame, including light, dark, terminal-native and NO_COLOR appearances.
The divider remains a thin line and does not acquire a focus accent.
TUI regression tests pass; layout and navigation are unchanged.
