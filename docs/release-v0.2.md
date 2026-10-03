# v0.2 - Complete public engine

Published 2 October 2026 for Linux amd64, with an archive and SHA256SUMS.
[Release](https://github.com/mayanklad/pokecrt/releases/tag/v0.2).
Source: `59fea7c10a1de45d5732f4bd84617757e6e57490`.

PokéCRT v0.2 expands the offline Linux printer into the complete public engine.

- Compose generation, type, color, evolution-stage, baby, legendary and mythical filters.
- Select exact forms, distinct visual genders and regular/shiny artwork.
- Print a named Pokémon or sample matching renderable species uniformly.
- Use compact or sprite-only print output with natural-size truecolor half blocks.
- Browse the public catalog with `list`, or metadata, evolution family and actual
  form/artwork inventory with `list --details`.
- Use the bundled executable without a source checkout, cache, network or trainer setup.
- Respect NO_COLOR, preserve piped colors and exit quietly on a closed output pipe.

Coverage: all 1,025 species across generations 1–9 and 1,448 metadata forms;
2,669 accepted assets covering 1,017 species and 1,327 collectible forms.
Standard regular printing covers 1,013 species, with 1,021 gender-aware slots;
1,334 shiny slots are available. Eight species have accepted distinct visual
male/female artwork. Metadata-only entries remain visible in regular catalog queries.

Known limitations: 12 standard species appearances and 122 catalog form appearances
lack accepted regular artwork. Separately, 53 pinned metadata varieties lack
resolved catalog/source identities; these are not 53 distinct missing sprites.
Minior meteor has no accepted distinct shiny sprite. Oinkologne's male/female
candidates remain excluded as generated artwork. Printing never substitutes a
different form, palette or gender when exact artwork is unavailable.

The sprite lookup index is derived automatically from the generated manifest.
Recorded laptop warm medians show about 3.9× faster random print and 3.5× faster
full catalog listing after indexing. Fresh-process startup and memory costs are
recorded separately; performance varies with machine and output destination.
See the repository benchmark report for methods and limits.

Not implemented in v0.2: trainer profiles, encounters, progression, achievements,
private Pokédex or TUI. Those remain later milestones. Initial supported binary:
Linux amd64. UTF-8 is required; truecolor is recommended. Natural-size sprites
can wrap in narrow terminals and are not automatically resized.

Dataset ID:
`8e5073aeeff3f7761e26ccdf9f068189516ae7eded9c0998a9d970b7d37e6b60`.

PokéCRT is an unofficial, attributed fan project. Original code is MIT licensed;
Pokémon artwork retains its respective owners' rights. Included licensing,
provider credits and third-party notices remain applicable. No affiliation,
endorsement or rights-holder permission is claimed.
