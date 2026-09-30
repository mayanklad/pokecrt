# PokéCRT

An offline Pokémon terminal app built in Go.

The planned product combines colorful sprite printing and a public catalog
with optional trainer profiles, encounters, achievements, and an interactive
Pokédex.

## Development status

D01 CLI foundation is complete. D02a adds a developer-only pinned source
download and verification tool. Artwork and trainer commands are not
implemented yet. No runtime Pokémon metadata or sprites are bundled.

Initial target: Linux amd64.

## Build and check

```bash
go test ./...
go vet ./...
go build -o ./bin/pokecrt ./cmd/pokecrt
```

## Current usage

```bash
./bin/pokecrt
./bin/pokecrt --help
./bin/pokecrt --version
```

## Developer source preparation

Run from the repository root:

```bash
go run ./tools/dataset --sources tools/dataset/sources.json --cache .cache/dataset --fetch --check
go run ./tools/dataset --sources tools/dataset/sources.json --cache .cache/dataset --check
```

`--fetch` is a developer-time network action. Every input must match its pinned
revision, size, and SHA-256. Valid cache files are reused; a corrupt file fails
visibly and is not silently replaced. `--check` alone verifies cached inputs
without downloading or writing. Generation and generated drift checking will
be added in D02b. Runtime commands do not use this cache or contact either source.

## Development documents

- [Implementation progress](docs/progress.md)
- [Dataset source audit](tools/dataset/source-audit.md)
- The product specification is maintained locally and excluded from Git history.
