# PokéCRT

Offline Pokémon terminal artwork, written in Go. Print one named or randomly
selected Pokémon with truecolor Unicode half blocks at the original pixel scale.

Published release: [v0.1](https://github.com/mayanklad/pokecrt/releases/tag/v0.1).
The current development branch targets v0.2.

## Current coverage

The development catalog contains all generation-1 through generation-9 species, their complete
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
Form tags and future achievement targets are generated from exact metadata and
accepted regular artwork: 38 eligible regional forms, 80 Mega/Gigantamax forms,
18 supported types and 17 fully supported branching families. Achievement
gameplay remains a later milestone. Coverage separately reports 53 pinned
metadata varieties without resolved catalog/source identities, in addition to
122 unavailable catalog appearances. These are different categories; the 53
varieties do not imply 53 distinct missing sprites. No artwork is inferred.
D06 is verified for the current reviewed inventory; public selection work is next.

Public filters and explicit form/shiny/gender printing flags arrive in D07;
variants can currently be inspected through the developer preview tool. Catalog
listing, trainers, encounters, achievements, and the TUI are not implemented yet.
Published v0.1 retains its original three-sprite coverage.
The initial tested platform is Linux amd64. A UTF-8 terminal is required;
truecolor support gives the intended artwork. Narrow terminals may wrap the
natural-size output; sprites are not resized automatically.

PokéCRT is an unofficial fan project. The owner has approved attributed releases
with bundled sprites under the [release policy](docs/release-policy.md).
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
gofmt -w cmd/pokecrt internal/cli internal/catalog
go test ./...
go vet ./...
go build -o ./bin/pokecrt ./cmd/pokecrt
```

No external Go modules are required yet. The executable embeds artwork and
metadata: runtime needs no source checkout, download cache, network, trainer,
or data directory. Help and version also work without trainer state.

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

Nonempty `NO_COLOR` disables ANSI colors while retaining block glyphs:

```bash
NO_COLOR=1 ./bin/pokecrt print --name bulbasaur
```

Unset or empty `NO_COLOR` preserves truecolor sequences, including through pipes.
Transparent halves use the terminal's default background. Output does not clear
the screen, move the cursor, or change the terminal title. Errors go to stderr;
successful artwork goes to stdout. Invalid invocations return status 2,
operational failures return 1, and broken pipes exit quietly with status 0.

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

## Local release candidate

Prepare a local candidate archive:

```bash
sh scripts/package.sh v0.1
(cd dist && sha256sum -c SHA256SUMS)
```

The script prepares ignored sprites from verified cached inputs, rejects generated
metadata drift, runs tests and vet; builds Linux amd64 with CGO
disabled, trimmed paths, and the requested version; and packages the executable,
README, LICENSE, licensing scope, third-party notices, and coverage notes. It writes
`dist/pokecrt_v0.1_linux_amd64.tar.gz` and `dist/SHA256SUMS`. An existing archive
is not overwritten. It never downloads sources or publishes anything.

Before distribution, complete the
[release checklist](docs/release-v0.1.md) and preserve the approved fan-project
policy and third-party notices. Cross-compilation alone does not prove
support for another platform.

## Project documents

- [Implementation progress](docs/progress.md)
- [Release candidate checklist](docs/release-v0.1.md)
- [Dataset generation and audit](tools/dataset/README.md)
- [Generated coverage](tools/dataset/coverage.md)
- [Third-party notices](THIRD_PARTY_NOTICES.md)
- [Code and artwork licensing](LICENSING.md)
- The specification is maintained locally and excluded from Git history.

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
