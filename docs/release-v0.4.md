# v0.4 — Interactive Adventure Menu

Release candidate notes. Source-cache verification, final tagged archive checks
and terminal acceptance remain pending. No v0.4 tag or archive is recorded here.

PokéCRT adds `pokecrt tui` to the existing offline Trainer CLI:

- Create, select and switch local trainer profiles with keyboard or mouse.
- Browse a private Pokédex, collected appearances, evolution families and records.
- Record an explicit encounter and inspect recent history, trainer statistics
  and the 50 achievement goals without changing the shared gameplay rules.
- Preview Dark, Light, Follow Terminal and Terminal Native palettes live, and
  explicitly save an appearance default.
- Navigate the Home Town Map, outlined header actions and single-line trainer
  card; the artwork remains static terminal characters.

The executable embeds artwork and metadata and needs no network at runtime.
Public printing/listing remain independent of trainer state. Browsing does not
record an encounter, grant rewards or reveal uncollected artwork. Explicit
encounters share the CLI's atomic storage and captured-trainer semantics.

Linux amd64 is the initial verified platform. TUI input and output must be
terminals. Wide Home requires 90×28; wide Pokédex/activity layouts require
100×24. The resize fallback starts below 40×12. Natural-size sprites use Unicode
half blocks; terminal fonts and truecolor support affect presentation.

## Known limitations

Compact mode still has reported layout/navigation issues. Further compact
refinement is deferred to v2 by the owner; wide layouts are recommended. The
reported remaining compact issues have not all been individually reproduced or
classified, so this release audit does not certify every compact flow.
Follow Terminal depends on background replies supported by the terminal. Native
mode preserves configured terminal defaults; it does not create transparency.

Dataset coverage is unchanged: 1,025 catalog species, 1,448 metadata forms,
1,017 encounter-eligible species, 1,327 collectible forms and 2,669 exact assets.
12 standard species appearances and 122 catalog form appearances lack accepted
regular artwork; 53 metadata varieties remain unresolved separately. Missing
artwork is never substituted. Dataset ID:
`8e5073aeeff3f7761e26ccdf9f068189516ae7eded9c0998a9d970b7d37e6b60`.

Trainer schema remains 1. Existing v0.3 schema-1 profiles are retained without a
migration or reset. Updating/removing the executable does not erase data.
Appearance settings use a separate version-1 file under the config directory;
only Save default writes it. Invalid settings remain untouched.

The source is MIT licensed; third-party metadata/artwork retain their own terms
and ownership. PokéCRT remains an unofficial fan project, with the existing
attribution and distribution policy. No endorsement or rights-holder permission
is claimed.
