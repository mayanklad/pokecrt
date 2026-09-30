# PokéCRT

An offline Pokémon terminal app built in Go.

The planned product combines colorful sprite printing and a public catalog
with optional trainer profiles, encounters, achievements, and an interactive
Pokédex.

## Development status

D01 provides root help and version handling. D02a provides pinned source
downloads and cache verification. D02b adds a minimal normalized catalog,
exact embedded asset lookup, deterministic generation, coverage, and notices.

The initial catalog contains nine starter-family species. Three standard
regular sprite candidates are bundled locally: Bulbasaur, Charizard, and
Squirtle. The other six remain metadata entries with unavailable artwork.
This is an initial development inventory, not complete Pokémon coverage.

Printing, rendering, trainer commands, and the TUI are not implemented yet.
Source-image redistribution verification remains open; this increment does
not claim release-rights clearance.

Initial supported target: Linux amd64.

## Prepare generated files

Run from the repository root. Generate before testing a checkout that does
not yet contain generated Go files and PNGs.

```bash
go run ./tools/dataset \
  --sources tools/dataset/sources.json \
  --cache .cache/dataset \
  --mappings tools/dataset/mappings.json \
  --out . \
  --fetch --generate --check
```

`--fetch` downloads only pinned developer inputs and verifies their size and
SHA-256. Existing valid inputs are reused; corrupt cache files fail visibly.
Downloads stay in the ignored cache and are not runtime dependencies.

`--generate` uses verified inputs to write the catalog, cropped PNGs, asset
manifest, coverage reports, and third-party notices. Cropping removes only
fully transparent outer margins; visible pixels are preserved. Partial alpha
requires an explicit normalization rule and is rejected by the initial rules.

## Check generated drift offline

```bash
go run ./tools/dataset \
  --sources tools/dataset/sources.json \
  --cache .cache/dataset \
  --mappings tools/dataset/mappings.json \
  --out . \
  --check
```

This command does not download or overwrite anything. It reports missing,
changed, or unexpected generated assets with a nonzero exit status.
Generation and checking can also run together without network when cached
inputs exist. An input-only check remains available by omitting both
`--mappings` and `--out`.

Generated Go files and PNGs, coverage reports, and THIRD_PARTY_NOTICES.md are
generator-owned. Do not edit them manually. A future redistribution must
satisfy the documented source-image verification gate.

## Build and check

```bash
go test ./...
go vet ./...
go build -o ./bin/pokecrt ./cmd/pokecrt
```

No external Go modules are required yet.

## Current runtime usage

```bash
./bin/pokecrt
./bin/pokecrt --help
./bin/pokecrt --version
```

Version output now includes the generated content-based dataset ID.
Runtime does not fetch sources, read the development cache, or initialize
trainer storage.

## Development documents

- [Implementation progress](docs/progress.md)
- [Dataset source audit](tools/dataset/source-audit.md)
- [Generated coverage](tools/dataset/coverage.md)
- [Third-party notices](THIRD_PARTY_NOTICES.md)
- The product specification is maintained locally and excluded from Git history.
