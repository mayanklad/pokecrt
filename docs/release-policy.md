# Fan-project distribution policy

Owner decision: Mayank Lad, 30 September 2026.
Application baseline: c3e184fe4a0bc9b867feee1cf81b6ad0b020a0b1.
Scope: v0.1 and subsequent PokéCRT releases, including v1.

PokéCRT distributes an offline executable containing explicitly mapped Pokémon
sprites as an unofficial, attributed fan project. The owner approved this
approach after reviewing the source terms and comparable projects' distribution
practices. Source PNGs are prepared from pinned, hash-verified inputs during development
and builds, excluded from Git, and embedded into released binaries.

Original PokéCRT code is MIT licensed. Third-party metadata, artwork, trademarks,
and notices retain their own terms and respective ownership. No rights-holder
permission, legal clearance, affiliation, or endorsement is claimed.

## Amendment to specification section 13

Before distribution, verify pinned source revisions and hashes, artwork
provenance and mapping quality, and the published source terms and credits.
Preserve required metadata/code notices and accurately distinguish artwork
ownership from software licensing. The owner authorizes attributed fan-project
bundles while acknowledging that underlying image rights remain unresolved;
absence of separate rights-holder clearance is not a release-blocking condition
under this revised specification. Do not manufacture a permission grant or
relicense third-party artwork as PokéCRT code.

All technical generation, coverage, reproducibility, and release checks remain
required. Runtime never downloads missing artwork. Source changes receive an
explicit audit rather than inheriting approval merely from a filename.

## Current source record

Direct artwork source: darknesspwnsu/pokesprite-v2, revision
32ab52ea6b61871da34d9a3c61c7760c65a37af7. Its README distinguishes Pokémon image
copyright from MIT terms for other material; its contributors file invites
project reuse. The three selected inherited regular images have pinned hashes
and source-index provenance referring to msikma/pokesprite. That upstream
provenance does not change the direct source used by PokéCRT.

Metadata source: PokeAPI/pokeapi, revision
bc92d3b6029ef1abe9e7ad424c400b338f3c11fe. Its BSD-3-Clause notice is retained in
THIRD_PARTY_NOTICES.md.

Records: tools/dataset/sources.json, mappings.json, README.md,
coverage.json, coverage.md, LICENSING.md, and THIRD_PARTY_NOTICES.md.

## Package requirements

Include the executable, README, LICENSE, LICENSING.md, THIRD_PARTY_NOTICES.md,
COVERAGE.md, and RELEASE_POLICY.md. Publish archive checksums. State actual
platform support and dataset coverage. Do not include profiles, credentials,
raw download caches, or unrelated development files.

The repository owner performs commits, tags, and GitHub release publication.
The assistant's GitHub access remains read-only.
