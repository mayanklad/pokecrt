# PokéCRT

Offline Pokémon terminal artwork, written in Go. Print one named or randomly
selected Pokémon with truecolor Unicode half blocks at the original pixel scale.

## Current coverage

This development build prints standard regular Bulbasaur, Charizard, and
Squirtle. The catalog contains nine starter-family species; six have no selected
artwork. Random printing chooses uniformly among the three available species.
Missing artwork produces an error instead of substituting another appearance.

Public filters, alternate forms, shiny palettes, visual gender selectors, catalog
listing, trainers, encounters, achievements, and the TUI are not implemented yet.
The initial tested platform is Linux amd64. A UTF-8 terminal is required;
truecolor support gives the intended artwork. Narrow terminals may wrap the
natural-size output; sprites are not resized automatically.

A public release is pending. Image redistribution review remains open, and the
project code license has not yet been selected. Generated PNGs stay local until
that review is complete. See [source audit](tools/dataset/source-audit.md) and
[third-party notices](THIRD_PARTY_NOTICES.md).

## Prepare a source checkout

Use Go 1.27.0 or newer. Development verification currently uses Go 1.27.1.
Run from the repository root. A fresh checkout needs local PNG generation before
building because the executable embeds those files.

```bash
go run ./tools/dataset \
  --sources tools/dataset/sources.json \
  --cache .cache/dataset \
  --mappings tools/dataset/mappings.json \
  --out . --fetch --generate --check
```

Only this explicit developer `--fetch` step downloads sources. Inputs have pinned
revisions, sizes, and SHA-256 hashes. Valid cached inputs are reused; corrupt
inputs fail visibly. Downloads stay in the ignored `.cache/dataset` directory.
With verified inputs already cached, omit `--fetch` to generate offline.

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
gofmt -w cmd/pokecrt internal/cli internal/query
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

After selecting a project code license and adding `LICENSE`, prepare an archive:

```bash
sh scripts/package.sh v0.1
(cd dist && sha256sum -c SHA256SUMS)
```

The script checks generated drift, tests, and vet; builds Linux amd64 with CGO
disabled, trimmed paths, and the requested version; and packages the executable,
README, LICENSE, third-party notices, and coverage notes. It writes
`dist/pokecrt_v0.1_linux_amd64.tar.gz` and `dist/SHA256SUMS`. An existing archive
is not overwritten. It never downloads sources or publishes anything.

Before distribution, complete image-rights review and the
[release checklist](docs/release-v0.1.md). Cross-compilation alone does not prove
support for another platform.

## Project documents

- [Implementation progress](docs/progress.md)
- [Release candidate checklist](docs/release-v0.1.md)
- [Dataset source audit](tools/dataset/source-audit.md)
- [Generated coverage](tools/dataset/coverage.md)
- [Third-party notices](THIRD_PARTY_NOTICES.md)
- The specification is maintained locally and excluded from Git history.
