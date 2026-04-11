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
- **Full MTG duels** powered by a rules engine based on [mage-go] (itself
  derived from XMage), with combat visuals, aura rendering, stack
  visualization, and targeting UI.
- **Deck building** with drag-and-drop deck editor and a multi-deck
  collection.
- **AI opponents** — heuristic-based today, with a MinMax-with-search AI in
  progress.
- **Sound** (press <kbd>M</kbd> to mute).
- **Cross-platform**: Linux, Windows (x64 + ARM), macOS (Intel + Apple
  Silicon), WebAssembly, and a WIP Android build.

## Play it

- **Web:** a WASM build is hosted at <https://throwingbones.com/ben/s30/>
  (no install required — just a browser that can handle a few MB of Go
  WebAssembly).
- **Windows, macOS, Linux, Android:** grab the latest build from the
  [GitHub releases page](https://github.com/benprew/s30/releases).

Want to build from source or hack on the game? See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

S30 is released under the **GNU General Public License v2** — see [LICENSE](LICENSE).

*Magic: The Gathering* is a trademark of Wizards of the Coast. This project is an
unaffiliated fan project for educational and non-commercial purposes. Card images
and data are not distributed with the source; you will need to provide your own.
