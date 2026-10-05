# Fan-project distribution policy

Policy recorded: 30 September 2026.
Application baseline: c3e184fe4a0bc9b867feee1cf81b6ad0b020a0b1.
Scope: v0.1 and subsequent PokéCRT releases, including v1.

PokéCRT distributes an offline executable containing explicitly mapped Pokémon
sprites as an unofficial, attributed fan project. The distribution policy accounts for source terms and comparable projects’
distribution practices. Source PNGs are prepared from pinned, hash-verified inputs during development
and builds, excluded from Git, and embedded into released binaries.

Original PokéCRT code is MIT licensed. Third-party metadata, artwork, trademarks,
and notices retain their own terms and respective ownership. No rights-holder
permission, legal clearance, affiliation, or endorsement is claimed.

## Verification and licensing

Distribution checks cover pinned source revisions and hashes, artwork provenance,
mapping quality, source terms and attribution. Required metadata and code
notices accompany release archives. Artwork ownership is separate from the MIT
license for original application code.

The distribution policy permits attributed, unofficial fan-project bundles.
Underlying image rights remain unresolved; this policy does not establish
rights-holder permission or relicense third-party artwork. Separate rights-holder
clearance is not a prerequisite defined by this project's release process.

Technical generation, coverage, reproducibility and release checks remain
required. Runtime never downloads missing artwork. Source changes receive an
explicit provenance and licensing review.

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

## Distribution contents

Release archives contain the executable, README, LICENSE, LICENSING.md,
THIRD_PARTY_NOTICES.md, COVERAGE.md, RELEASE_POLICY.md and RELEASE_NOTES.md. Archive checksums
accompany the release. Platform support and dataset coverage are stated explicitly.
Profiles, credentials, raw download caches and unrelated development files are
excluded from release archives.
