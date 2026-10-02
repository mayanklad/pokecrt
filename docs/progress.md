# Implementation progress

Specification: v1, including approved sections 25–27 and implementation clarifications
Current release target: v0.2 — Complete public engine
Last published milestone: v0.1
Current increment: D06 reviewed Generation 9 community provider
Current source commit: 39abfc30e855ac8d0aeddf3a71667b7eb3e0e63d
Commit note: verified baseline before this increment; owner verification/commit pending.
Published release: https://github.com/mayanklad/pokecrt/releases/tag/v0.1

## Runtime

- Root/help/version; named or uniformly random standard regular printing
- Compact/sprite output, natural-size truecolor half blocks, transparency
- NO_COLOR, piped colors, quiet broken pipes; no trainer state or runtime downloads
- No new public flags; explicit form/shiny/gender selectors remain D07

## Dataset

Development dataset ID:
`0ecbbb58dff8f03b2ae2e168ffe88943470ad94a28dc4922bfcadf3c7abb3955`

- 1,025 catalog species: all generation-1 through generation-9 species plus full evolution-family closure
- 1,447 metadata forms; 1,327 collectible forms; 2,657 exact regular/shiny assets
- 1,017 eligible species; 1,013 standard-printable species; 1,015 standard regular slots; 1,328 shiny slots
- 4 distinct visual gender slots across Pyroar/Meowstic, with male defaults
- 120 unavailable metadata form appearances; 12 species lack accepted standard artwork
- 2,935 pinned inputs; coverage.json/coverage.md record availability and quality
- Runtime images remain pinned PokéSprite-v2; community imports also match original-provider pins
- Complete evolution references, aliases, typing, stages, flags, and name corrections

## Tooling and verification

- Pinned/hash-verified developer downloads; deterministic generation and ignored asset preparation
- Exact source/palette provenance; inherited edit/retained-generation flags audited
- Reject provisional candidates, unknown inherited flags, duplicate pixels, and unsupported layouts
- Explicit name corrections require exact pinned source/canonical spellings and reasons
- Assistant tests/vet/race and generation checks passed; all 254 new sprites visually reviewed
- Tests cover family completeness, baby stages, source quality gates, name aliases,
  uniform species boundaries, missing artwork, and installed-binary behavior
- Routine species/form/asset entries are derived automatically; only policy and exceptions are maintained
- Provenance modules/tests and dataset documentation consolidated
- Static selection moved into catalog with injected sprite availability
- Earlier metadata and 2,403 PNGs remain intact; 254 exact community-provider assets added
- Pinned form table resolves Unown/Pichu; eleven reasoned form exceptions remain
- Source defaults checked against metadata defaults, including Dudunsparce
- Explicit --update-lock derives and validates new artwork pins; ordinary generation preserves locks
- Owner verification and commit remain pending

Storage schema version: none.

## Next

Continue D06 wider inventory, remaining visual genders, and provider/quality audits.
D06 is not complete; D07 selectors and D08 public listing remain unimplemented.
Trainers, encounters, achievements, and TUI remain later milestones.

## Decisions and workflow

- Section 25 governs ignored build-time PNGs, offline embedding, and distribution policy
- Source locks/mappings/generated metadata/reports are tracked; image/cache files are ignored
- Specification remains local and excluded from Git; changes delivered as ZIP bundles
- Owner applies files, verifies, commits, pushes, tags, and publishes
- Assistant GitHub access remains strictly read-only

Generation-3 verification covers Castform weather typing, Deoxys alias folding,
Spinda source-template exclusions, generation completeness and evolution closure.
Source exclusion evidence is tested; no additional package or source file added.


## Developer progress increment

The progress feature was implemented in the preceding increment. It adds
stderr progress for downloads and verified cache hits, a terminal bar/spinner,
plain redirected logs, and optional per-file `--verbose`. Concurrent artwork
lock updates use the same serialized reporter. Tests cover cache corruption,
stdout/stderr separation, concurrent reporting, and reporter shutdown; tests,
vet, and race checks pass. That preceding increment left generated files, pins, dataset identity and runtime
unchanged and was committed at c4d8712bbd3183262060d043ff2c8d464ddb66d5.


## Generation-4 increment

Exact form-type overrides come from pinned pokemon_form_types.csv and verified
metadata form/variety ownership. Arceus's 18 supported forms are typed correctly;
its unsupported unknown source appearance is excluded automatically. Tests also
cover Wormadam cloak typing, Rotom appliances, Shaymin Sky, source aliases,
malformed type rows and wrong owners. Existing artwork is unchanged. Generation
checks, tests, vet, race checks and visual review passed. Owner committed this
increment at d6b69f375179090cdb71bb2a4b9e9bf3e4ffc37b. No additional source file, package, dependency or runtime flag added.

Generation-5 verification: Darmanitan exact variety exception and Noble Lilligant
typing exception reviewed; seasonal forms, Meloetta typing, Genesect drives, and
canonical aliases tested. No generator source files or packages added.


## Generation-6 increment

Generation 5 was owner-committed at 058a78b5c62dfb068127ca9ee7dc645ac1f3df0e.
Generation 6 selects all six generations; redundant family seeds are removed.
Tests cover Vivillon patterns, flower colors, Furfrou trims, Hoopa typing,
Zygarde forms, and canonical aliases. Noble Avalugg's exact Hisuian typing
exception is reviewed. Seven new metadata appearances remain unavailable under
unchanged source-quality rules. Generation checks, tests, vet, build, fresh asset
preparation, and visual review pass; owner verification/commit pending.
No generator source files, packages, dependencies, or runtime flags added.
Generation 6 owner-committed at dd51b1520c576e882587928f23cd9a64250b5f0d.


## Generation-7 increment

Routine inventory remains automatic. Eight reviewed historical source-only
artwork records are excluded; no typing overrides added. Minior meteor shiny
pixels equal regular, so one reviewed duplicate-palette exclusion removes only
that shiny asset after pinned/provenance/hash verification. Coverage retains the
exclusion and exact normalized hash. Both source inputs stay locked.
Rules: d06-auto-5. Specification section 13 documents these evidence requirements.
Tests, vet, build, deterministic generation, fresh asset preparation, and visual
review pass. No new source files, packages, dependencies or public flags added.
Owner verification/commit pending. Next: generation 8 and remaining D06 audits.


## Generation-8 increment

Generation 7 owner-committed at 38662346418fb5f22265db340625c9e53eaace44.
Alcremie forms use one ownership-checked metadata suffix rule. A reviewed default
artwork alias uses exact metadata identity and four normalized hash pins; both
raw palettes remain locked, with original provenance verified. Nine unofficial
plain templates stay excluded. Four variety spelling exceptions preserve
Toxtricity/Urshifu Gigantamax and Calyrex fusion typing. Coverage records evidence.
Rules: d06-auto-6. Specification section 13 documents the correction gates.
Tests, vet, build, generation/fresh preparation and visual review pass.
Maximum dimensions: 67 columns, 56 source pixel rows (28 terminal rows).
No new source files, packages, dependencies or runtime flags added.
Owner verification/commit pending. Next: generation 9 and remaining D06 audits.


## Generation-9 metadata increment

Generation 8 owner-committed at 73fc88ac445deff81bb5b59ad88d217c5af7a6d9.
All nine source generations are selected automatically. No new corrections,
source pins, asset bytes, generator files or runtime features added. Generation
9 artwork is unavailable under the current provider/generated-image policy.
Tests cover exact mask types, default identities, source mode aliases, species
completeness and explicit missing-artwork failures. Tests, vet, build,
deterministic generation and fresh asset preparation pass.
Rules: d06-auto-6. Specification remains unchanged.
Owner verification/commit pending. D06 remains incomplete.
Next: additional-provider/source-quality audit, including generation-9 artwork;
then remaining form/gender semantics and achievement-tag audit gates.


## Generation 9 provider increment

Metadata increment owner-committed at 39abfc30e855ac8d0aeddf3a71667b7eb3e0e63d.
The bamq provider is reviewed at the exact v2 import commit. Acceptance remains
automatic for nongenerated canonical records with exact identity and palette
provenance. Provider/import evidence and credits are pinned; original/imported
PNGs must be byte-identical. Generated candidates stay excluded. Community
artwork and upstream resizing are disclosed in notices and licensing scope.
Rules: d06-auto-7. Specification section 13 documents provider acceptance gates.
Twelve standards and 120 metadata appearances remain unavailable; no fallback.
Tests, vet, race, build, deterministic generation, fresh preparation and visual
review pass. No new source files, packages, dependencies or runtime flags.
Owner verification/commit pending. D06 next reviews remaining provisional artwork,
form/gender semantics and achievement tags; D07 remains later.
