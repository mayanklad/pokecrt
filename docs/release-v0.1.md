# v0.1 — Basic offline printer

Publication status: prepared; the repository owner has not yet confirmed
creation of the tag or GitHub release.

## Release notes

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
artwork provenance are credited in the included notices. The owner has approved
attributed fan-project distribution; underlying rights-holder clearance and
endorsement are not claimed.

## Verification record

- [x] MIT code license committed.
- [x] Owner approved the project-wide fan-project policy and section 13 amendment.
- [x] Pinned generation checks, tests, vet, and Linux build passed on the candidate.
- [x] All three sprites visually reviewed on a light terminal background.
- [x] NO_COLOR and piped ANSI checked; closed-pipe integration test passed.
- [x] Extracted v0.1 candidate ran outside the checkout with matching dataset ID.
- [x] User's unshare -Urn check rendered Charizard with network isolated.
- [x] Candidate archive checksum passed.
- [ ] Final policy/notice changes regenerated, tested, and committed by owner.
- [ ] Final tag created and pushed by owner.
- [ ] Fresh archive built from the clean final tag; checksum and extraction checked.
- [ ] GitHub release published by owner with archive and SHA256SUMS.

The screenshot run reused an existing candidate archive. The final archive must
be rebuilt after committing the policy, notices, and build-preparation changes.
Generated PNGs are ignored and prepared locally from verified pinned inputs.

Narrow-terminal wrapping is expected at natural size; no destructive resizing
is performed. v0.1 has no filter or catalog command to include in its smoke test.

## Publication record

Published 30 September 2026 as a normal release, with Linux amd64 archive and
SHA256SUMS attached. Release: https://github.com/mayanklad/pokecrt/releases/tag/v0.1
Source: `b978f71e355a58e69dfcb47cbc3122c5a995d4f0`.
The checklist above records the preparation history; publication is complete.
