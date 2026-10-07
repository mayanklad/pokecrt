# v1.0 - PokéCRT

[Release](https://github.com/mayanklad/pokecrt/releases/tag/v1.0) · [Source tag](https://github.com/mayanklad/pokecrt/tree/v1.0) · [Linux amd64 archive](https://github.com/mayanklad/pokecrt/releases/download/v1.0/pokecrt_v1.0_linux_amd64.tar.gz) · [SHA256SUMS](https://github.com/mayanklad/pokecrt/releases/download/v1.0/SHA256SUMS)

PokéCRT combines offline Pokémon artwork, a public catalog, local trainer progression and an interactive Adventure Menu.

## Features

- Print named, random or filtered Pokémon artwork with exact form, gender and shiny selection.
- Browse the public catalog independently of trainer data.
- Create and switch local trainers; record explicit encounters with XP, levels and 50 achievement goals.
- Browse a private Pokédex with discovery-safe identities, collected appearances, evolution families and records.
- Inspect encounter history, trainer statistics and achievements with keyboard or mouse.
- Preview Dark, Light, Follow Terminal and Terminal Native appearances live; save a default explicitly.

## Changes since v0.4

Home, Achievements, Encounter, Trainer, Appearance and Pokédex have responsive controls, consistent footer actions and contextual hints. Frame controls appear inline only when needed. Outlined actions reserve their full height; smaller layouts use centered rounded controls. Focus and active-selection indicators remain separate.

Pokédex navigation reaches the visible index, entry, frame, tab and footer controls. Up from a tab enters its associated entry pane, including panes without scroll controls. Wide National Index reclaims space for entry content. Artwork Scroll supports horizontal panning. Trainer and achievement panes retain independent scrolling. Locked Evolution panels use normal frame navigation and cannot open a stale card through Scroll.

Appearance remains with footer actions at every supported size. Narrow action rows follow their visual order with arrow keys. Home retains header Refresh/Quit actions and the expanded Town Map.

The README provides installation and quick-start instructions. The detailed [guide](guide.md) is also included as `docs/guide.md` in the release archive.

## Platform and terminal requirements

Linux amd64 is the verified distribution platform. The executable embeds artwork and metadata and requires no network at runtime. TUI input and output must be terminals. A UTF-8 terminal is required; fonts and truecolor support affect artwork presentation.

The minimum supported TUI size is 40×12. Smaller windows show a resize instruction and Quit. Wide Home starts at 90×28; wide Pokédex and activity pages start at 100×24. Narrow layouts adapt the same workflows to the available space. Natural-size artwork may require scrolling or wrap in public CLI output; it is not rescaled.

Follow Terminal relies on terminal background replies and uses native colors when replies are unavailable. Terminal Native preserves configured terminal defaults; it does not add transparency.

## Dataset coverage

Dataset ID: `8e5073aeeff3f7761e26ccdf9f068189516ae7eded9c0998a9d970b7d37e6b60`.

Coverage remains 1,025 catalog species, 1,448 metadata forms, 1,017 encounter-eligible species, 1,327 collectible forms and 2,669 exact assets. All 2,947 pinned inputs are verified. Twelve standard species appearances and 122 catalog form appearances lack accepted regular artwork; 53 metadata varieties remain unresolved separately. Missing artwork is never substituted, and uncollected artwork is not previewed in private views.

## Updating and data

Trainer schema remains 1. Existing schema-1 profiles are retained without a migration or reset. Replacing or uninstalling the executable does not erase trainer progress. Appearance settings use a separate version-1 configuration file; only Save default writes it. Invalid or future-version settings remain untouched.

Public printing and catalog commands do not require trainer storage. Browsing does not record encounters, grant rewards or reveal undiscovered identities. Explicit encounters share atomic recording and captured-trainer semantics across CLI and TUI.

## Licensing

Original source is MIT licensed. Third-party code, metadata and artwork retain their own terms; bundled notices and coverage describe that scope. PokéCRT is an unofficial fan project. No endorsement or rights-holder permission is claimed. See [release policy](release-policy.md).


## Linux package distribution

Downloadable Linux x86-64 files include `.deb` for Ubuntu/Kubuntu and Debian,
`.rpm` for Fedora and `.pkg.tar.zst` for Arch Linux. The
[guide](guide.md#native-linux-packages) covers checksum verification, local-file
installation, updates and removal. Packages install the executable and documentation;
user-owned trainer data and appearance settings survive removal. Release preparation
is manual and requires the cross-distribution verification workflow to pass.
