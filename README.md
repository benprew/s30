# Shandalar 30th Anniversary Edition

A modern, open-source reimagining of MicroProse's 1997 *Magic: The Gathering* game
aka "Shandalar." Shandalar is a roguelike deck-building adventure: you wander an
isometric overworld, visit cities to trade and build your deck, hunt down rival
wizards, collect the five colored amulets, and eventually take down the big bad of
Shandalar.

Written in Go using the [Ebiten](https://ebitengine.org) game engine, with a full MTG rules engine under the
hood.

> Note: this repo started as a solo hobby project and is still very much a
> work in progress. Expect rough edges. Bug reports and patches are very
> welcome — see [CONTRIBUTING.md](CONTRIBUTING.md).

## Features

- **Isometric overworld** with Perlin-noise terrain (water, sand, marsh,
  plains, forest, mountains, snow), chunked loading, and autotiling.
- **Cities** (hamlets, towns, capitals) with card shops, wisemen, and quests.
- **Rival wizards** wander the map with chase/wander AI; duel them, bribe
  them, or avoid them.
- **Full MTG duels** powered by a rules engine based on [mage-go](https://github.com/benprew/mage-go) (itself
  derived from XMage), with combat visuals, aura rendering, stack
  visualization, and targeting UI.
- **Modern MTG rules** — duels follow modern MTG Comprehensive Rules
  (post-M10, so classic mechanics like mana burn are absent, combat damage
  does not use the stack, and cards follow modern Oracle text).
- **Deck building** with drag-and-drop deck editor and a multi-deck
  collection.
- **AI opponents** — heuristic-based today, with a MinMax-with-search AI in
  progress.
- **Sound** (press <kbd>M</kbd> to mute).
- **Collectible alternate art** — each owned card keeps its artwork across
  decks, duels, trading, and saves. Add Scryfall artwork by set in the card
  data.
- **Cross-platform**: Linux, Windows (x64 + ARM), macOS (Intel + Apple
  Silicon), WebAssembly, and a WIP Android build.

## Play it

- **Web:** a WASM build is hosted at <https://throwingbones.com/ben/s30/>
  (no install required — just a browser that can handle a few MB of Go
  WebAssembly).
- **Windows, macOS, Linux, Android:** grab the latest build from the
  [GitHub releases page](https://github.com/benprew/s30/releases).

Want to build from source or hack on the game? See [CONTRIBUTING.md](CONTRIBUTING.md).

## Versions and updates

The title screen and bug reports show the game version. GitHub release builds use their release tag (for example, `v1.2.3`), including Android and iOS builds. Local `make` builds use `git describe --tags --always --dirty`. Use `make VERSION=v1.2.3` to set a version explicitly. Plain `go build` uses `dev`; to stamp it, add `-ldflags "-X github.com/benprew/s30/game/version.Current=v1.2.3"`.

Release builds check `https://s30.throwingbones.com/api/releases/latest` once when the title screen first updates. Windows and standalone Linux builds offer **Update and restart**. Android offers **Install update** and opens the system installer, where the player approves the update. macOS keeps **Download update**, which opens the browser for manual installation. **Later** dismisses the notice for this session and cancels an active download. Failed checks do not interrupt play. Development builds, local builds with changes after a tag, iOS builds, and browser builds do not check for updates.

Automatic updates require a manifest signed with Ed25519. The public key is part of the installed game; a key from the API is never trusted. Each signed entry binds the version, platform, download URL, SHA-256 checksum, and file size. Windows and Linux entries also bind a small installation helper. Downloads go into a private directory and must pass verification before installation. Updates start from the title screen before a game is created or loaded. The desktop helper waits for S30 to exit, checks the files again, keeps a recovery copy, replaces the executable, and starts it with the original arguments and working directory. If replacement fails, it starts the unchanged executable. If process startup fails after replacement, it restores and starts the old executable. The next launch removes files from completed updates and downloads abandoned for more than 48 hours. Failures after S30 exits retain an `update.log` and any recovery copy in the `.s30-update-*` directory beside the game.

Linux updates require a writable standalone installation. Executables in system directories, AppImage, Flatpak, and Snap installations use their distribution's update method. Package maintainers can set `S30_DISABLE_AUTO_UPDATE=1`. Windows installations also need a writable game directory; the updater does not request administrator access. Android requires permission to install apps from S30 and an APK with the same application ID and a compatible signing certificate. See [Android releases](mobile/android/README.md).

Before publishing the first release with automatic updates:

1. Create the signing key outside the repository: `go run ./cmd/update-manifest -keygen -out /secure/path/s30-update`. Keep the `.key` file private and backed up. The `.pub` file contains the public key.
2. Set the GitHub Actions secret `UPDATE_SIGNING_KEY` to the `.key` file's base64 seed. Set the repository variable `UPDATE_PUBLIC_KEY` to the `.pub` file's base64 public key, without a trailing newline. These keys are separate from the Android APK signing key. Keep both signing identities for future releases.
3. Deploy the release endpoint from `../s30-web`. The release workflow builds and publishes the desktop helpers, `s30-update-manifest.json`, and its `.sig` file. Signing fails if the keys do not match or build artifacts are missing. Android version codes come from the stable release version and must exceed those already distributed.

Existing releases and builds without a pinned key continue to offer browser downloads. The first version with this updater must therefore be installed manually. Local release builds can use `make VERSION=v1.2.3 UPDATE_PUBLIC_KEY=<base64-public-key>`.

Set `S30_UPDATE_URL` to use a different release API endpoint during development. The response keeps `version` and a `downloads` map keyed by `GOOS/GOARCH`, plus `android` for the universal APK. Signed releases also include `manifest` (the base64-encoded original JSON bytes) and `signature` (base64 Ed25519). Download URLs must use `https://github.com/benprew/s30/releases/download/<version>/<file>`, with the same version as the release. Changing the endpoint does not change the pinned signing key.

## License

S30 is released under the **GNU General Public License v2** — see [LICENSE](LICENSE).

*Magic: The Gathering* is a trademark of Wizards of the Coast. This project is an
unaffiliated fan project for educational and non-commercial purposes. Card images
and data are not distributed with the source; you will need to provide your own.
