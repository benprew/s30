# Shandalar 30

MTG card game clone using Go and Ebiten. Roguelike deck-building adventure with a complete MTG rules engine.

## Build & Test

```bash
make test         # Run tests (go test -count=10 ./...)
make              # Run game
make winbuild     # Windows executable
make macbuild     # macOS Intel
make macarmbuild  # macOS ARM
make webbuild     # WebAssembly
```

## Structure

- `game/` - Game UI, world, screens, sprites
- `mtg/` - MTG rules engine (see `mtg/CLAUDE.md` for details)
- `utils/` - Python scripts for card images and parsing
- `assets/` - Card data and images

## Utilities

```bash
python3 utils/find_cards.py --keyword Flying   # Find cards by keyword
python3 utils/find_cards.py --list-keywords    # List all keywords
```

## Code Style

### Go
- Run `golangci-lint run` after changes
- Avoid inline comments; make code self-explanatory
- Comments should explain "why", not "what"
- Don't resize images in `Draw()` or `Update()` methods; resize when creating screens

### Python
- Use type hints everywhere
