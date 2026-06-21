# Contributing to Shandalar 30

Thanks for your interest! S30 is a hobby project, so contributions of any
size — bug reports, small fixes, art, design feedback, or bigger features —
are all welcome.

## Building from source

### Requirements

- **Go 1.25 or newer** (see `go.mod` for the exact version).
- **A C toolchain** and the X11 / OpenGL / ALSA headers that Ebiten needs
  at build time.
- **uv** for the card-art build pipeline.
- **Git** and **make**.

Runtime platform support is whatever
[Ebiten supports](https://ebitengine.org/en/documents/).

### Install system dependencies

Debian / Ubuntu:

```sh
make builddeps
```

Fedora:

```sh
make fedorabuilddeps
```

Nix:

There is a `flake.nix` in the project. Activate it by putting `use flake` in an `.envrc` file and running `direnv allow`.

Install the Python dependencies for the art/asset pipeline:

On macOS and Windows the Go toolchain + Ebiten's prebuilt dependencies are
normally enough — no extra system packages needed.

### Build & run

Clone and build the game:

```sh
git clone https://github.com/benprew/s30.git
cd s30
make       # downloads card art and builds self-contained dist/s30
make run   # runs the game with embedded card art
```

Run tests:

```sh
make test      # go test -count=10 ./...
```

Making builds for your OS:

```sh
make winbuild       # Windows x86_64 → dist/s30.exe
make macbuild       # macOS → dist/s30_mac
make build          # linux build → dist/s30
```

Every Make binary target caches its generated card archive under `assets/art/`
and embeds it in the output. Both the archive and `dist/` are ignored by Git.

Once `make build` and `make test` both run cleanly, you're ready to hack.

## Project layout

```
game/        Game UI, world, screens, sprites (see game/CLAUDE.md)
mtg/         MTG rules engine integration
cmd/         Standalone tools (mtg_test, parse_cards, tile_transitions)
utils/       Python scripts for card image processing and data parsing
assets/      Card data, sprites, fonts, configs (embedded via Go embed)
docs/        Design notes (architecture, dungeons, sound, tile transitions)
mobile/      Ebitenmobile glue for the Android build
```

## Useful dev commands

```sh
python3 utils/find_cards.py   # Find cards by name/keyword/ability
go run ./cmd/mtg_test         # AI-vs-AI sim
go run ./cmd/duel_test        # player-vs-AI sim
go run ./cmd/dungeon_test     # dungeon run with sample starting deck
```

## How to contribute

1. **Open an issue first** for anything non-trivial so we can agree on the
   approach before you write the code. For small fixes (typos, obvious
   bugs, one-file tweaks) just send a PR.
2. **Fork & branch** off `main`. Keep branches focused — one feature or fix
   per PR.
3. **Write tests** alongside code changes where it's practical. The project
   uses TDD as the default workflow (write the test, make it pass).
4. **Run the linter and tests** before pushing:
   ```sh
   golangci-lint run --fix
   make test
   ```
5. **Open a PR** with a short description of *what* changed and *why*.
   Screenshots or short clips are great for UI changes.

## Code style

### Go

- Run `golangci-lint run --fix` after changes.
- Let the code speak for itself — avoid inline comments that restate what
  the code already says.
- Comments should explain *why*, not *what*. Function header comments on
  exported functions are encouraged.
- Don't resize images inside `Draw()` or `Update()`; resize when the screen
  is constructed.
- Assets are embedded with `go:embed` — prefer adding new assets to
  `assets/` and loading them through the embed FS.
- Screens implement `screenui.Screen` and return the next screen name from
  `Update()` for transitions.

### Python (for `utils/`)

- Use type hints everywhere.
- Standard `black`/`ruff` formatting is fine.

### Commit messages

- Short imperative subject line ("Fix ante card filtering when deck is
  empty"), optional body explaining the motivation.
- One logical change per commit when you can manage it.

## Things to pick up

These are ideas I'd love to see in the game but haven't had time to get to.
None of them are reserved — grab whatever looks fun. Check existing issues
before starting to avoid duplicate work.

### Gameplay / engagement loop

- **Reward scaling / gold bounties** on higher-level enemies, plus a
  city-side "wanted board" of specific rogues.
- **More random encounters.** `RandomEncounterScr` exists but is thin.
  Treasure, healing springs, wandering merchants, and choice events
  ("trade 3 commons for a random rare") would all fit.
- **Quest-driven exploration.** Quest enemies with distinct map markers,
  chained quests, and world-altering rewards (clear a zone, unlock a shop
  tier).

### Rules engine / duels

- **Phase pacing.** When phases jump, the UI should roll them forward
  visibly (~50ms each) instead of snapping.
- **AI spell telegraphing.** Pause + show targets when the AI casts a
  spell or activates an ability, even if the player has no responses.

### Platforms & infra

- **WASM save/load via LocalStorage.** Saves currently don't persist in the
  browser build.
- **Android polish.** The APK builds but the on-screen controls, scaling,
  and input need a lot of work.
- **CI.** A GitHub Actions workflow that runs `make test` and
  `golangci-lint` on PRs would be great.

### Art & content

- **Shandalar-accurate world.** Make the generated world look more like the
  original Shandalar lore / map (see `shandalar_map.jpeg` and
  `projects.org`).
- **More wiseman boons**, **more rogue designs**, **more card art** — art
  contributions welcome.

If you want to work on something that isn't in this list, open an issue and
let's talk about it.

## Code of conduct

Be kind. Assume good faith. Leave the repo a little nicer than you found
it. That's it.
