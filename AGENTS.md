# Shandalar 30

MTG card game clone using Go and Ebiten. Roguelike deck-building adventure with a complete MTG rules engine. Isometric world exploration, city trading, deck building, and full MTG duels against AI opponents.

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

- `game/` - Game UI, world, screens, sprites (see `game/CLAUDE.md`)
- `mtg/` - MTG rules engine (see `mtg/CLAUDE.md`)
- `cmd/` - Standalone tools (mtg_test, parse_cards, tile_transitions)
- `utils/` - Python scripts for card images and parsing
- `assets/` - Card data, sprites, fonts, rogue configs (embedded via Go embed)

## Utilities

```bash
python3 utils/find_cards.py --keyword Flying        # Find cards by keyword
python3 utils/find_cards.py --name "Lightning Bolt"  # Find cards by name
python3 utils/find_cards.py --list-keywords          # List all keywords
python3 utils/find_cards.py --list-types             # List all ability types
go run ./cmd/parse_cards --card "Lightning Bolt"     # Parse a card's abilities
go run ./cmd/parse_cards --unparsed                  # Show cards with unparsed text
go run ./cmd/mtg_test                                # Run AI vs AI game simulation
```

## Code Style

### Go
- Run `golangci-lint run` after changes
- Avoid inline comments; make code self-explanatory
- Comments should explain "why", not "what"
- Don't resize images in `Draw()` or `Update()` methods; resize when creating screens

### Python
- Use type hints everywhere
