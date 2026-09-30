# PokéCRT

An offline Pokémon terminal app built in Go.

The planned product combines colorful sprite printing and a public catalog
with optional trainer profiles, encounters, achievements, and an interactive
Pokédex.

## Development status

D01: CLI foundation. Artwork and trainer commands are not implemented yet.
No Pokémon metadata or sprites are bundled.

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

## Development documents

- [Product specification](docs/specification.md)
- [Implementation progress](docs/progress.md)
