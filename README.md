# PokéCRT

Offline Pokémon terminal artwork, written in Go. Print one named or randomly
selected Pokémon with truecolor Unicode half blocks at the original pixel scale.

Release: [v0.4 - Interactive Adventure Menu](https://github.com/mayanklad/pokecrt/releases/tag/v0.4).
[Source tag](https://github.com/mayanklad/pokecrt/tree/v0.4) · [Release notes](docs/release-v0.4.md).
[Linux amd64 archive](https://github.com/mayanklad/pokecrt/releases/download/v0.4/pokecrt_v0.4_linux_amd64.tar.gz) · [SHA256SUMS](https://github.com/mayanklad/pokecrt/releases/download/v0.4/SHA256SUMS).

## Current coverage

The release catalog contains all generation-1 through generation-9 species, their complete
connected evolution families (including later-generation relatives):
1,025 species and 1,448 metadata forms.

Audited artwork covers 1,017 species and 1,327 collectible forms, with 2,669 exact
regular/shiny assets. Standard regular printing covers 1,013 species; 12 species
currently lack accepted standard artwork. 122 form appearances remain unavailable.
Minior meteor has regular artwork only because its source shiny pixels are
identical; the reviewed exclusion is hash-verified. Printing never substitutes
another appearance.

Eight species have distinct male/female regular and shiny slots within one
standard form, derived from verified inherited source declarations. There are 1,021 standard regular slots across 1,013 printable
species; random printing samples species uniformly. Reviewed community artwork
from `bamq/pokemon-sprites` adds Generation 9 printing, verified against the exact
commit imported by PokéSprite-v2. Generated candidates remain excluded. Some
species have accepted alternate artwork but no accepted standard artwork.
Oinkologne’s source male/female records are one standard metadata form; both
artwork candidates remain excluded as generated. This adds no printable slots.
Form tags and achievement inventory are generated from exact metadata and
accepted regular artwork: 38 eligible regional forms, 80 Mega/Gigantamax forms,
18 supported types and 17 fully supported branching families. Achievement
gameplay is included in the trainer CLI. Coverage separately reports 53 pinned
metadata varieties without resolved catalog/source identities, in addition to
122 unavailable catalog appearances. These are different categories; the 53
varieties do not imply 53 distinct missing sprites. No artwork is inferred.
Public selectors, catalog listing and dataset validation are implemented.
Sprite lookup uses an exact-key index derived automatically from the generated
manifest; performance measurements are documented in docs/benchmarks.md.

Public metadata filters and explicit form/shiny/gender printing are implemented.
v0.3 adds local trainer profiles, fair encounters, XP/levels, 50 achievements,
private Pokédex browsing and trainer statistics. Public printing and listing
remain independent of trainer state. v0.4 adds the
Adventure Menu: interactive trainer creation/selection, private Pokédex,
encounters/history, trainer statistics, achievements and saved live appearance
choices. The Interactive Adventure Menu is published in v0.4. Compact-mode issues remain known and further
refinement is planned before v1.0. Use a wide terminal for the best interface.

The earlier v0.1 release included three standard sprites.
The initial tested platform is Linux amd64. A UTF-8 terminal is required;
truecolor support gives the intended artwork. Narrow terminals may wrap the
natural-size output; sprites are not resized automatically.

PokéCRT is an unofficial fan project. Attributed releases with bundled sprites
follow the [release policy](docs/release-policy.md).
Original PokéCRT code is MIT licensed; Pokémon artwork retains its respective
owners' rights. No rights-holder permission or endorsement is claimed. See
[licensing scope](LICENSING.md), [source audit](tools/dataset/README.md),
and [third-party notices](THIRD_PARTY_NOTICES.md).

## Prepare a source checkout

Use Go 1.27.0 or newer. Development verification currently uses Go 1.27.1.
Run from the repository root. PNGs are generated locally and excluded from Git.
Prepare them before compiling or testing a fresh checkout; Go embeds their bytes
into the executable. Pinned inputs and hashes make preparation reproducible.

```bash
go run ./tools/dataset \
  --sources tools/dataset/sources.json \
  --cache .cache/dataset \
  --mappings tools/dataset/mappings.json \
  --out . --fetch --prepare-assets --check
```

Only this explicit developer `--fetch` step downloads sources. Inputs have pinned
revisions, sizes, and SHA-256 hashes. Valid cached inputs are reused; corrupt
inputs fail visibly. Downloads stay in the ignored `.cache/dataset` directory. Artwork inputs use
the source ID and cache folder `pokesprite-v2`; inherited `msikma/pokesprite`
provenance identifies the original artwork provider and is preserved. Reviewed
community inputs are also pinned under `bamq` to verify unchanged imports.
With verified inputs already cached, omit `--fetch` to generate offline.

For an intentional dataset/schema update, use `--generate --check` instead of
`--prepare-assets --check`. Preparation verifies committed metadata; generation
updates it. Review generated changes before committing.

Generation owns catalog and manifest Go files, cropped PNGs, coverage reports,
and notices. Do not edit them manually. Cropping removes fully transparent
outer margins without resizing visible pixels. Partial alpha is rejected.

Check for generated drift without downloading or overwriting files:

```bash
go run ./tools/dataset \
  --sources tools/dataset/sources.json \
  --cache .cache/dataset \
  --mappings tools/dataset/mappings.json \
  --out . --check
```

## Build and verify

```bash
gofmt -w cmd internal tools
go test ./...
go vet ./...
go test -race ./...
CGO_ENABLED=0 go test ./internal/trainer ./internal/storage
go build -o ./bin/pokecrt ./cmd/pokecrt
```

The source pins CGO-free SQLite and Unicode modules in `go.mod`/`go.sum`.
The executable embeds artwork and metadata: runtime needs no source checkout,
download cache or network. Public print/list/help/version need no trainer or data
directory. Profile creation and encounters use local SQLite storage.

The Linux amd64 integration test builds a trimmed executable, runs it from an
empty directory, checks piped output and a closed pipe, verifies version/dataset
identity, and confirms that public commands create no trainer directories.

## Print

```bash
./bin/pokecrt --help
./bin/pokecrt --version
./bin/pokecrt print --help
./bin/pokecrt print
./bin/pokecrt print --name charizard
./bin/pokecrt print --name squirtle --output sprite
```

Default `compact` output is the sprite, a blank line, and a heading such as
`#006 Charizard`. `sprite` emits only artwork and line breaks. Name lookup is
case-insensitive and accepts exact canonical names or generated unambiguous
aliases; it does not guess partial names. Scalar flags may appear only once.

Selection defaults to `standard`, regular palette and the selected form's declared
default gender. Random printing samples matching renderable species uniformly;
missing artwork never falls back to another form, gender or palette.

```bash
./bin/pokecrt print --name charizard --form mega-x --shiny
./bin/pokecrt print --name meowstic --gender female --shiny
./bin/pokecrt print --gen 1,2 --type fire --type-any flying,dragon
./bin/pokecrt print --color red,blue --stage 1,2
./bin/pokecrt print --legendary
./bin/pokecrt print --mythical
./bin/pokecrt print --baby
```

| Selector | Meaning |
| --- | --- |
| `--gen`, `--color`, `--stage` | Any listed introduction generation, species color or evolution stage |
| `--type` | Selected form must contain every listed type |
| `--type-any` | Selected form must contain at least one listed type |
| `--form` | One exact canonical catalog form slug |
| `--gender` | Exact distinct visual gender: male or female |
| `--shiny` | Actual shiny artwork; default/false selects regular |
| `--legendary`, `--mythical`, `--baby` | True requires the source trait; false imposes no restriction |

Categories combine with AND. Comma lists trim, deduplicate and reject empty
elements. Selector values are case-insensitive; numeric lists require positive
decimal integers supported by bundled metadata. Long flags use `--`; `-h` is
the help alias. Boolean values use `=true` or `=false`; bare flags mean true.
Repeated flags, unknown static selectors and invalid values return status 2.
Valid constraints without a matching renderable appearance return status 1.
For example, standard Charizard does not match `--type dragon`; its Mega X form
does. A globally known form absent on a species is a valid empty selection.
Gender flags require distinct visual slots; biological gender alone is insufficient.

Compact variant headings append form, explicitly requested gender, then Shiny:
`#006 Charizard · Mega X · Shiny`. Unspecified/default gender stays omitted.

Nonempty `NO_COLOR` disables ANSI colors while retaining block glyphs:

```bash
NO_COLOR=1 ./bin/pokecrt print --name bulbasaur
```

Unset or empty `NO_COLOR` preserves truecolor sequences, including through pipes.
Transparent halves use the terminal's default background. Output does not clear
the screen, move the cursor, or change the terminal title. Errors go to stderr;
successful artwork goes to stdout. Invalid invocations return status 2,
operational failures return 1, and broken pipes exit quietly with status 0.

## Public catalog

`list` uses the same selectors and exact appearance defaults as `print`. It is
state-free, deterministic and sorted by National number. Compact rows contain
number/name, introduction generation, selected-form types and artwork availability.
Species names and form labels are never truncated to fit terminal width.

```bash
./bin/pokecrt list
./bin/pokecrt list --name charizard
./bin/pokecrt list --gen 1,2 --type fire --type-any flying,dragon
./bin/pokecrt list --name charizard --form mega-x --shiny --details
./bin/pokecrt list --name meowstic --gender female --shiny --details
./bin/pokecrt list --name oinkologne --details
```

Regular metadata entries remain visible when selected artwork is unavailable.
Explicit gender requires a declared distinct visual identity; shiny selection
requires an actual shiny asset. A valid empty query prints
`No Pokémon match the specified filters.` and exits successfully; invalid
selectors still return status 2. `--output` is a print flag, not a list flag.

`--details` displays the selected heading, one sprite or an unavailable message,
verified generation/types/color/stage/status flags, the complete connected
evolution family including branches, and all catalog form IDs/names with their
available regular/shiny/gender slots. It never renders all form sprites or
substitutes an alternate when standard artwork is missing. Broad detail queries
can be long; no results are capped and no pager is launched. No trainer or
discovery state limits the public catalog. `NO_COLOR` applies to detail sprites.

## Public-engine verification

Dataset tests cross-check generated coverage against the embedded catalog/assets and public
queries, including declared form/gender ownership, regular/shiny eligibility,
missing appearances, completion denominators and fully supported branching
families. Installed-binary tests run outside the repository/cache and check
status codes, stdout/stderr boundaries, ANSI/NO_COLOR and real closed pipes.

| Example | Expected result |
| --- | --- |
| `print --gen 1,2 --type fire --type-any flying,dragon` | One matching species; standard form and default gender |
| `list --name charizard --type dragon` | Empty list, status 0 |
| `print --name charizard --type dragon` | No candidate, status 1 |
| `list --name charizard --form mega-x --type fire,dragon` | Mega X with Fire / Dragon typing |
| `list --name oinkologne --gender female --details` | Metadata retained; selected artwork unavailable |
| `list --name minior --shiny` | Empty list; no fabricated shiny meteor palette |
| `print --gen 1,,2` | Invalid empty list member, status 2 |
| `list --form unknown` | Unknown static form, status 2 |

Metadata generation, color and stage remain species attributes across forms,
shiny palettes and visual genders. Coverage limits remain explicit: 53 metadata
varieties lack resolved catalog/source identities, separately from 122
unavailable catalog appearances. Public output never fills those gaps by guess.

The public suite includes `go test ./...`, `go vet ./...` and
`go test -race ./...`. Measured performance baselines, initial regression budgets,
repeatable Go benchmarks and measurement limits are recorded in
[docs/benchmarks.md](docs/benchmarks.md). The lookup index improves random-print
and full-list performance while preserving artwork and selection rules.

## Local installation

Build first, then install the executable into `~/.local/bin`:

```bash
sh scripts/install.sh ./bin/pokecrt
```

If that directory is not on PATH, add this to your shell configuration:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

Open a new shell or reload its configuration, then run:

```bash
pokecrt --version
pokecrt print --output sprite
```

`POKECRT_INSTALL_DIR` can override the installation directory. To update, build
and rerun the installer. To uninstall, remove only `~/.local/bin/pokecrt`.
Installation and removal do not erase trainer data.

## Project documents

- [Implementation progress](docs/progress.md)
- [v0.4 release notes](docs/release-v0.4.md)
- [v0.3 release notes](docs/release-v0.3.md)
- [v0.2 release notes](docs/release-v0.2.md)
- [Performance baseline and comparison](docs/benchmarks.md)
- [v0.1 release record](docs/release-v0.1.md)
- [Dataset generation and audit](tools/dataset/README.md)
- [Generated coverage](tools/dataset/coverage.md)
- [Third-party notices](THIRD_PARTY_NOTICES.md)
- [Code and artwork licensing](LICENSING.md)
- Local project documents, including the specification, live under `docs/local/`
  and are excluded from Git history.

## Developer variant preview

Inspect exact bundled appearances without adding temporary public CLI flags:

```bash
go run ./tools/render-preview --name charizard --form mega-x --palette shiny
go run ./tools/render-preview --name venusaur --form gmax --palette regular
```

The preview tool accepts exact form and palette IDs and an optional visual gender.
It never records trainer state and fails if the exact asset is unavailable.

Compare genuine visual gender variants in developer previews:

```bash
go run ./tools/render-preview --name hippowdon --gender female
go run ./tools/render-preview --name pyroar --gender male
go run ./tools/render-preview --name pyroar --gender female
go run ./tools/render-preview --name meowstic --gender female --palette shiny
```

Routine dataset inventory is derived automatically from pinned inputs. The mapping
configuration contains selection policy and reviewed exceptions; see the
[dataset guide](tools/dataset/README.md).

Spinda’s unofficial blank/filled pattern templates are excluded source records,
not collectible forms. Castform weather typings and Deoxys aliases come from
pinned metadata and source identities.

Developer dataset downloads show progress on stderr by default. Add `--verbose`
to the dataset command for individual downloaded and cache-verified filenames.

Arceus’s 18 supported type forms use exact pinned form-specific typing. Its
unsupported `unknown` appearance is excluded explicitly; base Normal typing
is never substituted for another type form.


Transformation identity is checked against pinned metadata before source alias
folding. Tatsugiri Curly Mega and Droopy Mega remain distinct metadata forms;
the source's ordinary-image aliases do not provide accepted transformation art.
Their artwork is explicitly unavailable, with no fallback to ordinary forms.

## Trainer profiles

Create and select local SQLite trainer profiles explicitly:

```bash
pokecrt trainer --help
pokecrt trainer create Mayank
pokecrt trainer create "Professor Oak"
pokecrt trainer list
pokecrt trainer use "Professor Oak"
pokecrt trainer
pokecrt trainer --name Mayank
pokecrt trainer achievements
```

Only the first creation automatically selects a trainer. Names accept 1–32
Unicode code points after trimming; NFC normalization and Unicode case folding
prevent duplicate identities while preserving display spelling. Quoted internal
spaces are supported. Listing and viewing do not initialize missing storage.

Data is stored in `trainers.sqlite3` beneath an absolute `POKECRT_DATA_DIR`,
otherwise absolute `XDG_DATA_HOME/pokecrt`, otherwise
`~/.local/share/pokecrt`. A relative explicit override is rejected; a relative
XDG base uses the home fallback. The data directory is protected with mode 0700
and the database with mode 0600. Existing unsupported or corrupt state is never
replaced automatically.

Profile views show name, UTC creation time, active status, XP/level, collection
statistics and generation progress. `trainer --name` views another profile without
switching. `trainer achievements` shows the active trainer’s earned and locked
goals. Profile commands never record discoveries or encounters. Public print/list/help/version
remain independent of trainer storage. Dependency licenses are retained in
[LICENSING.md](LICENSING.md).

The encounter service selects uniformly by species, form and
visual gender from accepted regular artwork, with a 1/4096 shiny chance when
that exact appearance supports it. It prepares artwork before atomically
recording history, discoveries, XP and achievement unlocks for the captured
trainer. Completion goals are derived from eligible catalog artwork and evolution
relationships. Each encounter grants 10 XP, plus 40 for a first species, 20 for a
first exact variant and 100 for shiny artwork. Levels advance every 1,000 XP.

## Encounters

An encounter requires an explicitly created, active trainer:

```bash
pokecrt encounter --help
pokecrt encounter
pokecrt encounter --output compact
pokecrt encounter --output no-title
pokecrt encounter --output achievements
pokecrt encounter --output sprite
```

| Output mode | Display |
| --- | --- |
| `full` (default) | Artwork, appearance heading, selected-form types, progress and new achievements |
| `compact` | Artwork and appearance heading |
| `no-title` | Artwork, progress and new achievements |
| `achievements` | Artwork and only newly earned achievements |
| `sprite` | Artwork alone |

Every mode records one unrestricted encounter with the same discovery, XP and
achievement rules. No name, generation, type, form, gender or shiny selector is
accepted. `NO_COLOR` disables artwork color. Headings identify the exact form,
visual gender and shiny palette where applicable; full output uses that form's
types. Progress distinguishes new species, new exact variants and repeats, shows
current eligible species/variant completion, and reports XP and level changes.
A first shiny variant does not imply collection of its regular counterpart.

All state commits before output. An output error leaves the encounter recorded;
there is no automatic repeat. Broken pipes exit quietly. With no active trainer,
the command fails with setup/selection guidance and creates no implicit profile.
Help and invalid invocations do not open storage. Hidden notices still persist
as achievements and remain visible in `trainer achievements`.

## Achievement catalog

Current inventory supports **50 achievements** per trainer: 19 numerical
milestones, 21 themed goals, nine generation completions and National Researcher.
All are optional, unlock once and grant no extra XP or encounter advantage.
New unlocks appear in encounter output. `trainer achievements` groups earned
entries with UTC dates and locked goals with current progress. Browsing does
not award a newly introduced goal already satisfied by old history; it unlocks
on the next committed encounter. The v0.2 binary contains only the public engine.

| Numerical milestones | Thresholds |
| --- | --- |
| First Contact, then `<n> Encounters` | 1, 10, 25, 50, 100, 250, 500, 1,000 encounters |
| `<n> Species Discovered` | 5, 10, 25, 50, 100, 250, 500 distinct species |
| `<n> Variants Collected` | 10, 25, 50, 100 distinct exact variants |

An exact variant includes species, form, visual gender and regular/shiny palette.
Repeats advance encounter milestones but never distinct-species/variant goals.

| Themed achievement | Requirement |
| --- | --- |
| Shiny Discovery | Record a shiny encounter. |
| Type Explorer | Encounter forms covering all 18 currently supported types. |
| Regional Discovery | Encounter a supported regional form. |
| Transformation Discovery | Encounter a supported Mega or Gigantamax form. |
| Branching Out | Discover every species in one fully supported branching evolution family. |
| Across Generations | Discover species from five different generations. |
| World Traveler | Discover a species from every supported generation, currently nine. |
| Type Sampler | Encounter forms covering eight distinct types. |
| Type Specialist | Discover ten species through encounters sharing one type. |
| Dual-Type Collector | Discover ten species through dual-type form encounters. |
| Rainbow Collection | Discover species covering six distinct Pokédex colors. |
| Growing Collection | Discover a species at every supported evolution stage, currently 1, 2 and 3. |
| Regional Explorer | Encounter regional forms of three distinct species. |
| Transformation Explorer | Encounter Mega or Gigantamax forms of three distinct species. |
| Changing Faces | Encounter nonstandard forms of five distinct species. |
| Form Collector | Collect three distinct forms of one species. |
| Family Reunion | Discover every species in one fully supported evolution family containing at least three species. |
| Legendary Encounter | Encounter a Legendary Pokémon. |
| Mythical Encounter | Encounter a Mythical Pokémon. |
| Small Beginnings | Encounter a baby Pokémon. |
| Shiny Collection | Collect five distinct shiny variants. |

Type goals use forms actually encountered, without inferring unseen alternate
forms. Dual-Type Collector requires two types on one encountered form. Different
genders and palettes do not become additional forms for Form Collector.
Family Reunion includes ordinary evolution chains; Branching Out requires a
branch. A completed branching family may earn both. Special discoveries and
shiny collecting are longer-term rewards, not prerequisites for other goals.

| Completion achievement | Current eligible species |
| --- | ---: |
| Generation 1 Researcher | 151 |
| Generation 2 Researcher | 100 |
| Generation 3 Researcher | 135 |
| Generation 4 Researcher | 107 |
| Generation 5 Researcher | 156 |
| Generation 6 Researcher | 72 |
| Generation 7 Researcher | 88 |
| Generation 8 Researcher | 89 |
| Generation 9 Researcher | 119 |
| National Researcher | 1,017 |

Any encountered variant satisfies its species membership. Qualifying targets
are derived from catalog metadata and exact accepted regular-artwork inventory;
unsupported goals are suppressed. Earned unlocks retain their date, dataset and
applicable target across inventory updates. New goals already satisfied by
history unlock on the next committed encounter; viewing alone grants nothing.

## Trainer Pokédex

`pokecrt dex` summarizes the active trainer's eligible species and appearance
collection, encounters, shiny collections and generation progress.

```bash
pokecrt dex list --seen
pokecrt dex list --unseen --gen 1,2
pokecrt dex list --type fire --type-any dragon,flying
pokecrt dex show --number 6
pokecrt dex show --name charizard --form mega-x --shiny
```

Lists include anonymous National slots for undiscovered species. Generation is
safe to filter before discovery; type, color, stage and source status filters
include discovered species only and cannot accompany `--unseen`. Type conditions
must match one actually encountered form.

Entry views reveal species facts after discovery, known forms and observed
genders, collection counts and UTC first/last seen times. Evolution nodes remain
anonymous until discovered. Artwork appears only for the exact collected
appearance. A shiny-first or alternate-form discovery does not substitute artwork
for an uncollected standard regular selection. Removed artwork retains its
collection record with an explicit unavailable notice. Browsing requires an
active trainer, opens storage read-only and never grants XP or achievements.

## Trainer statistics

`pokecrt trainer` and `pokecrt trainer --name NAME` show total XP, level,
progress to the next level, encounters, historical species/variant collections,
shiny encounter count, distinct shiny collections and UTC first/last encounter
times. An empty history displays `None` for encounter times.

Current eligible completion is separate from historical discoveries: removing
artwork does not erase collection history. Generation rows show discovered
species and current eligible completion by species introduction generation.
Historical species without current generation metadata are counted separately;
the application does not infer missing metadata. Statistics and achievement
progress reveal counts and generic goal descriptions, never unseen identities
or form names. Earned dates and targets are retained across inventory updates.
All browsing opens existing storage read-only and grants no rewards.

## Shell and Fastfetch

For artwork without recording trainer activity:

```bash
pokecrt print --output sprite
pokecrt print --output sprite | fastfetch --file-raw -
```

For one recorded encounter, select a trainer first and use:

```bash
pokecrt encounter --output sprite | fastfetch --file-raw -
```

Each execution records an encounter, including when Fastfetch runs on shell
startup. Use public `print` when shell startup should leave trainer data unchanged.
Fastfetch's JSONC configuration also supports a raw command logo:

```json
{
  "logo": {
    "type": "command-raw",
    "source": "pokecrt print --output sprite"
  }
}
```

Merge the logo object into an existing configuration. `command-raw` is a JSONC
configuration option; piping uses `--file-raw -`. Both forms are documented in
[Fastfetch's logo options](https://github.com/fastfetch-cli/fastfetch/wiki/Logo-options).
PokéCRT preserves ANSI colors through pipes; nonempty `NO_COLOR` disables color.
Natural-size artwork can wrap if the terminal has too few columns.

## Interactive interface development

```bash
pokecrt tui
pokecrt tui --appearance dark
pokecrt tui --appearance light
pokecrt tui --appearance follow-terminal
pokecrt tui --appearance terminal-native
```

The Adventure Menu shows a static Town Map above dialogue and a four-section
menu. Wide Home layouts (at least 90 columns and 28 rows) put outlined Refresh
and Quit controls above a single-line trainer card in the top right. Taller
compact layouts stack the dialogue and menu below the map. Short layouts omit
the map and group Refresh/Quit above the menu, in the top right where space
permits. Left/Right switches these actions; Down enters Pokédex, and Up from
Pokédex returns to Refresh. The interface fills the terminal; below 40 columns or 12 rows it shows
a resize instruction and clickable Quit. Pokédex and activity wide layouts
require at least 100 columns and 24 rows.

Compact mode has known layout and navigation issues under review before v1.0;
the 40×12 minimum indicates the resize fallback, not that every compact flow
has release-quality validation. Wide layouts are the preferred v0.4 experience.

Arrows, Tab/Shift+Tab and j/k move focus; Enter or Space activates it. Click any
control, or use the mouse wheel to move focus. A opens appearance settings. Esc
returns from settings and quits at the main menu; Q quits at the main menu, and
Ctrl+C quits anywhere. The shell restores terminal modes on exit.

Dark and Light select fixed palettes. Follow Terminal requests the terminal
background without changing its palette; supported replies update the app live.
After a successful reply, it checks about every two seconds while focused. A
missing reply falls back to default terminal colors and stops periodic queries;
focus regain or reselecting Follow Terminal probes again. Terminal Native uses
default foreground/background colors, preserving terminal-configured
transparency; the application does not create transparency. Nonempty NO_COLOR
removes styling while retaining visible focus and selection markers. Ordinary
text in Native mode uses your terminal foreground; purpose-colour borders and
visible focus markers remain. Changes apply live. **Save as default** explicitly
remembers the choice; browsing alone never creates a configuration file.

Without `--appearance`, startup uses the saved choice, then Follow Terminal if
none exists. An explicit flag overrides the saved choice for that run until you
save it. Preferences live in `$XDG_CONFIG_HOME/pokecrt/appearance.conf`, falling
back to `~/.config/pokecrt/appearance.conf`; absolute `POKECRT_CONFIG_DIR` overrides
the containing directory. The tiny versioned text file is saved atomically with
0600 permissions. Malformed, oversized, symlinked or unsupported-version files
are left unchanged; Appearance displays the error and live switching still works.

Both stdin and stdout must be usable terminals. Help and redirected invocation
never open trainer data or appearance settings. The interface reads trainer
status asynchronously through an existing read-only database. Merely navigating
or typing does not initialize or modify trainer data.

D20 adds interactive setup and selection. First run opens **New Trainer**;
Enter in the name field or **Create trainer** submits the name. The first profile
becomes active; additional profiles remain inactive until **Use trainer**. Open
**Trainer** from the main menu to create or switch profiles. Arrows and mouse
wheel scroll the chooser; click a row to select, then click **Use trainer**, or
press Enter on the focused list. Refresh rereads existing trainer state.

Names use the same Unicode validation and duplicate-name rules as the CLI.
The field supports typing, paste, arrows, Home/End, Backspace/Delete and Ctrl+U.
`q`, `a`, `j` and `k` are literal while typing. Tab moves focus. The on-screen
keyboard supplies lowercase/uppercase letters, digits, punctuation, Space and
Backspace for mouse-only creation; clicking the field places the cursor. Esc or
Cancel before submission leaves trainer paths absent. During an in-flight
request, Cancel requests cancellation and refreshes status; a completed commit
is retained. No uncertain operation is automatically retried. Setup/select never
records encounters, discoveries, XP awards or achievements. Appearance and quit
remain available while requests run.
Home uses static town artwork; it never previews an uncollected Pokémon.
Missing/corrupt trainer data leaves navigation, appearance and quit usable.
See docs/progress.md for implementation status and docs/benchmarks.md for
measured verification limits.

### Trainer activity screens (D22)

**Encounters** has an explicit Encounter button and a committed-result display.
The existing shared engine prepares exact artwork before committing the encounter,
discovery, XP and achievement unlocks together. Only that button records an
encounter; it is disabled while reading or saving. Held Enter repeats are ignored.
The displayed trainer is captured for the action so an external trainer switch
cannot redirect it. Navigation remains available while saving, with one global
in-flight guard preventing another encounter across screens. Ctrl+C exits gracefully. Errors leave a recoverable message and allow retry.

**History** shows the newest 50 encounters, ordered by recorded time then ID.
Names/forms use stored encounter snapshots; removed artwork does not erase
history. Click a row or use Previous/Next and Up/Down on the selected-entry control,
then Open selected entry to browse that exact appearance in the safe Pokédex.
Browsing never creates another encounter.

**Trainer** displays creation time, level, XP bar, encounter/collection counts,
shiny counts, eligible completion, generation progress, and encounter dates.
Choose / create trainer opens the existing trainer chooser. The Adventure Menu
also displays the active trainer's level. **Achievements** separates earned
badges and their dates from locked goals and current progress, using the same
disclosure-safe registry as the CLI. Reading never awards a badge. Achievements
uses one focus marker on the selected control; the active journal/panel border
is highlighted in color modes. Left/Right visits Scroll, Up and Down in each
visible panel, then the footer actions. Up/Down on Scroll moves only that
panel's content. Footer hints use bracketed key names, one row when they fit and two when needed; compact
labels remain centered. Hints sit above the bottom border with a separating
divider, combining arrow directions when their actions match. Refresh uses the same name in every Achievements layout.


Wide views use framed device/card panes; compact views scroll, and compact
encounter results switch between natural-size artwork and Details. Tab selects
controls; arrows follow their positions, scroll focused records or pan focused
artwork. Clickable scroll/pan controls and mouse wheel support mouse-only use.
PageUp/PageDown scroll records. Appearance remains live in all screens.
Unsupported terminals retain the existing resize hint and Quit action. Screen
transitions clear once before redraw to avoid stale frame fragments.

Home and Achievements footer hints sit immediately above the bottom frame,
separated by a divider. Key labels use compact brackets such as `[Enter]`;
matching arrow actions share one Move label. Hints wrap only when needed.

Navigation, Home and Achievements refinements are implemented through D35. The current
release audit records remaining packaging, compatibility and verification gates.


### Terminal controls and appearance

Controls use flat rounded borders and purpose colours. Focus and
selection have separate visible markers, including with NO_COLOR. Action rows
precede bottom navigation hints. Compact screens use rounded single-row caps.

Appearance changes apply live. **Save as default** remembers the selected mode
for later launches. **Refresh** rereads local records without creating a discovery.
The Route 01 home scene is static character artwork; no image protocol or new
animation timer is required. Terminal fonts determine glyph shape and joins.

