<div align="center">

<h1><img src="docs/media/logo.png" alt="PokéCRT" width="620"></h1>

**Your terminal. Your Pokémon journey. Entirely offline.**

Natural-size Pokémon artwork, a searchable catalog, and a local trainer adventure-one executable.

[Website](https://mayanklad.github.io/pokecrt/) &nbsp; / &nbsp; [Download v1.0](https://github.com/mayanklad/pokecrt/releases/tag/v1.0) &nbsp; / &nbsp; [Install](#install-in-a-minute) &nbsp; / &nbsp; [Full guide](docs/guide.md) &nbsp; / &nbsp; [Release notes](docs/release-v1.0.md)

[![Release v1.0](https://img.shields.io/badge/release-v1.0-49d5ec?style=flat-square)](https://github.com/mayanklad/pokecrt/releases/tag/v1.0)
[![Linux amd64](https://img.shields.io/badge/platform-Linux%20amd64-76b4e2?style=flat-square)](docs/release-v1.0.md#platform-and-terminal-requirements)
[![Go 1.27+](https://img.shields.io/badge/Go-1.27%2B-00ADD8?style=flat-square)](docs/guide.md#prepare-a-source-checkout)
[![Source license MIT](https://img.shields.io/badge/source%20license-MIT-b79de0?style=flat-square)](LICENSE)

**Linux amd64 · Keyboard & mouse · No network at runtime**

</div>

![PokéCRT Home screen with the Town Map, trainer card and adventure menu](docs/media/home.png)

## A little adventure, wherever you open a terminal

Print a favorite Pokémon, add a sprite to your shell, or open the Adventure Menu and build a collection through encounters. Artwork and metadata are embedded; saved progress stays on your machine.

| Artwork & catalog | Your trainer journey |
| --- | --- |
| Print named, random or filtered Pokémon. Choose exact forms, shiny palettes and supported visual genders. | Create local trainers, record encounters, gain XP and unlock 50 achievement goals. |
| Browse all 1,025 species across nine generations. Public catalog access is independent of saved progress. | Explore your private Pokédex, collected appearances, evolution families and encounter history. |
| Natural-size truecolor artwork, transparent backgrounds and `NO_COLOR` support. | Responsive pages, keyboard and mouse controls, and four live appearance choices. |

## See it in action

**A private Pokédex that grows with your discoveries.** Browse collected appearances, explore evolution families and keep track of each encounter.

![Pokédex showing Scyther artwork, the National Index and discovery details](docs/media/pokedex.png)

**Exact artwork, straight from the command line.** Forms, shiny palettes and supported visual genders are first-class choices.

| Shiny Mega X Charizard | Shiny female Meowstic |
| --- | --- |
| ![Terminal command printing shiny Mega X Charizard](docs/media/cli-artwork.png) | ![Terminal command printing shiny female Meowstic](docs/media/cli-forms.png) |
| `pokecrt print --name charizard --form mega-x --shiny` | `pokecrt print --name meowstic --gender female --shiny` |

**Every encounter adds to your journey.** Track trainer progress and work toward 50 achievement goals.

<details>
<summary>See encounters, trainer progress and achievements</summary>

![Encounter result with artwork and progression details](docs/media/encounter.png)

![Trainer card and generation collection progress](docs/media/trainer.png)

![Earned badges and next collection goals](docs/media/achievements.png)

</details>

### A quick tour

![Animated tour through Home, Pokédex, Encounters, Trainer, Achievements and Appearance](docs/media/demo.gif)

[Watch or download the short video](docs/media/demo.mp4). Screenshots and the tour use a separate sample trainer; the interface and artwork are captured from the running app.

## Install in a minute

Download the **Linux amd64 archive** and **SHA256SUMS** from [v1.0](https://github.com/mayanklad/pokecrt/releases/tag/v1.0), then run from their directory:

```bash
sha256sum --ignore-missing -c SHA256SUMS
tar -xzf pokecrt_v1.0_linux_amd64.tar.gz
mkdir -p "$HOME/.local/bin"
install -m 0755 pokecrt "$HOME/.local/bin/pokecrt"
"$HOME/.local/bin/pokecrt" --version
```

If `~/.local/bin` is not on your PATH, add `export PATH="$HOME/.local/bin:$PATH"` to your shell configuration and open a new shell.

### Download a Linux package

Download a `.deb` for Ubuntu/Kubuntu or Debian, `.rpm` for Fedora, or
`.pkg.tar.zst` for Arch Linux from the v1.0 release.
[Package installation, updates and removal](docs/guide.md#native-linux-packages).

### Install from source

Use Go 1.27.0 or newer. Asset preparation downloads verified inputs once; the built app runs offline.

```bash
git clone https://github.com/mayanklad/pokecrt.git
cd pokecrt
git checkout v1.0
go run ./tools/dataset \
  --sources tools/dataset/sources.json \
  --cache .cache/dataset \
  --mappings tools/dataset/mappings.json \
  --out . --fetch --prepare-assets --check
go build -o bin/pokecrt ./cmd/pokecrt
sh scripts/install.sh ./bin/pokecrt
```

[Build checks and packaging](docs/guide.md#build-and-verify) · [Uninstall or reset](docs/guide.md#uninstall)

## Start exploring

```bash
# Open the interactive adventure; first launch offers trainer creation.
pokecrt tui

# Print a favorite, or let the app pick one.
pokecrt print --name charizard
pokecrt print

# Explore exact appearances and the public catalog.
pokecrt print --name charizard --form mega-x --shiny
pokecrt list --name eevee --details
```

### Record an encounter

```bash
# Create and activate a local trainer, then record an encounter.
pokecrt trainer create Ash
pokecrt trainer use Ash
pokecrt encounter

# Review progress without recording another encounter.
pokecrt trainer
pokecrt dex
```

`encounter` records the encounter and awards progression. Printing, catalog listing and private browsing do not record encounters. [Profiles and encounter options](docs/guide.md#encounters) explain how to switch trainers and choose output modes.

The Adventure Menu brings together **Pokédex, Encounter, Trainer and Achievements**. Use arrows or Tab to move, Enter to select, and Esc to return. Mouse controls are available throughout. Switch among Dark, Light, Follow Terminal and Terminal Native appearances; saving a default is explicit.

For shell integration, filters, trainer commands and detailed controls, open the [full guide](docs/guide.md).

## Good to know

- **Terminal:** UTF-8 required; truecolor gives the intended artwork. TUI requires terminal input/output and at least **40×12**. Wide views provide more space for artwork and simultaneous panels.
- **Coverage:** 1,025 catalog species and 2,669 accepted exact assets. Some appearances are unavailable; PokéCRT never substitutes another sprite. See [coverage](tools/dataset/coverage.md).
- **Local data:** printing and the public catalog need no trainer. Encounters are explicit; browsing does not change progression. Updating the binary preserves profiles and settings.
- **Artwork:** sprites retain their natural dimensions; narrow terminals may wrap public output. Follow Terminal depends on your terminal’s background replies and falls back to native colors.

## Read more

[Guide](docs/guide.md) · [v1.0 notes](docs/release-v1.0.md) · [Progress](docs/progress.md) · [Benchmarks](docs/benchmarks.md)

PokéCRT is an unofficial fan project, inspired by [pokemon-colorscripts](https://gitlab.com/phoneybadger/pokemon-colorscripts), [pokego](https://github.com/rubiin/pokego) and [poketerm](https://github.com/chris-wood-mo/poketerm). Artwork provenance is recorded in the [source audit](https://github.com/mayanklad/pokecrt/blob/v1.0/tools/dataset/README.md).

Original source is [MIT licensed](LICENSE). Third-party code, metadata and Pokémon artwork retain their respective terms. See [licensing](LICENSING.md), [notices](THIRD_PARTY_NOTICES.md) and [release policy](docs/release-policy.md). No endorsement or rights-holder permission is claimed.
