# PokéCRT v0.3 - Trainer CLI

PokéCRT v0.3 adds persistent trainer gameplay to the offline public engine.
Linux amd64 is the supported binary platform; UTF-8 is required and truecolor
is recommended. The interactive TUI remains a later milestone.

- Create explicitly named profiles, select an active trainer and browse another
  profile's statistics without changing the active selection.
- Record unrestricted encounters with uniform selection at each species, form
  and visual-gender step. Shiny chance is 1/4096 when the exact appearance supports it.
- Collect exact species/form/gender/palette variants, gain XP and advance levels.
- Earn 50 numerical, themed and completion achievements, with retained unlock
  dates and progress browsing. Achievements grant no additional XP or advantage.
- Choose full, compact, no-title, achievements or sprite encounter output.
- Browse a private Pokédex that hides undiscovered identities, forms, genders,
  evolution names and uncollected artwork.
- Use public printing or encounter artwork in shell/Fastfetch integrations.

History, discoveries, XP and unlocks commit together. Each encounter grants
10 XP, plus 40 for a new species, 20 for a new exact variant and 100 for shiny.
Levels advance every 1,000 XP. Output failure after commit does not undo or
repeat an encounter; broken pipes exit quietly. Encounter modes record identical
state. Read-only browsing never grants rewards.

Trainer state uses CGO-free SQLite schema 1. This is the first trainer milestone;
existing development schema-1 profiles remain compatible without a migration or
reset. State resides beneath an absolute POKECRT_DATA_DIR, otherwise absolute
XDG_DATA_HOME/pokecrt, otherwise ~/.local/share/pokecrt. Directories use mode 0700
and databases mode 0600. Unsupported/corrupt databases are not replaced.
Updating the executable preserves profiles; uninstalling it does not erase data.

The executable embeds metadata and artwork and needs no network at runtime.
Public print/list/help/version never open trainer storage and remain usable with
missing or corrupt trainer state. First-run gameplay requires explicit profile
creation and selection. No trainer database is distributed in the archive.

Coverage is unchanged: 1,025 catalog species across nine generations, 1,448
metadata forms, 1,017 encounter-eligible species, 1,327 collectible forms and
2,669 exact assets. Standard regular printing covers 1,013 species. There are
1,334 shiny slots. Missing exact artwork is never substituted.

Known limitations: 12 standard species appearances and 122 catalog form
appearances lack accepted regular artwork. Separately, 53 metadata varieties lack
resolved catalog/source identities. Minior meteor has no distinct accepted shiny
artwork; Oinkologne male/female candidates remain excluded as generated artwork.
Natural-size sprites may wrap in narrow terminals. Encounters use local write
transactions and may wait for another writer; an unresolved commit is not retried.

Achievement evidence checks reuse stored form flags and distinct shiny discoveries
rather than rescanning encounter history for those predicates. No new index,
schema or gameplay rule is introduced. Benchmarks, methods and verification
limits are recorded in [benchmarks.md](benchmarks.md).

Dataset ID:
`8e5073aeeff3f7761e26ccdf9f068189516ae7eded9c0998a9d970b7d37e6b60`.

PokéCRT is an unofficial fan project. Original code is MIT licensed; artwork
retains its respective owners' rights. Included licensing, provider credits and
third-party notices apply. No affiliation, endorsement or rights-holder permission
is claimed.
