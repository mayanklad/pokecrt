# v0.1 release candidate

Status: unpublished; release checks remain open.

## Implemented

- Offline named and uniform random standard regular printing
- Compact and sprite output
- Truecolor half blocks, transparency, and natural source-pixel scale
- NO_COLOR, piped ANSI, quiet broken pipes, and strict invocation errors
- Root and print help, version, and generated dataset identity

Coverage: 9 catalog species; 3 standard regular assets (Bulbasaur, Charizard,
Squirtle); no shiny or distinct visual gender assets.

Dataset ID: `5bb40e33703ffd1b07855ba3552cff88edd4f8d2c0d03f2860872aee279803e4`

Supported release target: Linux amd64. No trainer storage or state migration is
introduced. Filters, public catalog listing, forms, shiny/gender selection,
trainers, encounters, achievements, and TUI are not part of v0.1.

## Before publication

- [x] Owner selected MIT for original PokéCRT code; LICENSE is prepared.
- [ ] Complete specification section 13 source-image redistribution review.
- [ ] Run generation drift checks, tests, and vet on the final source.
- [ ] Review all three colored sprites in a real Linux terminal, including a
      nonblack background and a narrow terminal; confirm no stale colors.
- [ ] Check installed output with NO_COLOR unset, empty, and nonempty.
- [ ] Build the archive from the final tagged source and verify SHA256SUMS.
- [ ] Extract and run the included executable outside the checkout, with no
      development cache or trainer directory and network unavailable.
- [ ] Verify version, dataset ID, executable mode, notices, and coverage notes.

v0.1 has no filter or catalog command to include in its smoke test. Those checks
apply once the corresponding features land.

## Archive smoke commands

These commands prepare and test a local candidate. Public distribution remains
pending the artwork review and final smoke checks:

```bash
sh scripts/package.sh v0.1
(cd dist && sha256sum -c SHA256SUMS)
smoke_dir=$(mktemp -d)
tar -xzf dist/pokecrt_v0.1_linux_amd64.tar.gz -C "$smoke_dir"
(
  cd "$smoke_dir"
  ./pokecrt --help
  ./pokecrt --version
  ./pokecrt print --help
  ./pokecrt print
  ./pokecrt print --name charizard
  ./pokecrt print --name squirtle --output sprite
  NO_COLOR=1 ./pokecrt print --name bulbasaur
)
```

Repeat the extracted-binary checks with network unavailable. The binary should
work with only its embedded data. Keep the temporary directory until review is
complete, then remove it manually.

Tagging and GitHub release publication are performed by the repository owner
after the gates pass. No release has been created by the assistant.
