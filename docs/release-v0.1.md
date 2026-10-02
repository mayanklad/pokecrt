# v0.1 — Basic offline printer

Published 30 September 2026 for Linux amd64, with an archive and SHA256SUMS.
[Release](https://github.com/mayanklad/pokecrt/releases/tag/v0.1).
Source: `b978f71e355a58e69dfcb47cbc3122c5a995d4f0`.

PokéCRT v0.1 prints Pokémon artwork offline in a Linux terminal.

- Print a named Pokémon or choose uniformly among available species.
- Choose compact output (sprite plus number/name) or sprite-only output.
- Truecolor half blocks preserve source-pixel scale and terminal-background transparency.
- NO_COLOR disables ANSI colors; colors otherwise survive piping.
- Help/version, strict invocation errors, and quiet broken-pipe handling.
- No trainer setup, network access, source checkout, or runtime data files required.

Coverage: 9 catalog species; 3 standard regular sprites: Bulbasaur, Charizard,
Squirtle. Six catalog species have no selected artwork. No shiny or distinct
visual gender artwork is included. Missing artwork fails without substitution.

Not implemented: metadata filters, form/gender/shiny selectors, public list,
trainer profiles, encounters, achievements, and TUI. Supported target: Linux
amd64. No trainer storage or state migration is introduced.

Dataset ID:
`5bb40e33703ffd1b07855ba3552cff88edd4f8d2c0d03f2860872aee279803e4`

PokéCRT is an unofficial fan project. Original code is MIT licensed. Pokémon
artwork retains its respective owners' rights. PokéSprite v2 and inherited
artwork provenance are credited in the included notices. Attributed fan-project distribution follows
the release policy; rights-holder clearance and endorsement are not claimed.
