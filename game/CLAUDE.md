# Game Package

Game UI, world generation, screens, and sprites built on Ebiten. Handles isometric world exploration, city interactions, deck building, and MTG duels.

## Package Structure

```
game/
├── game.go             # Main game loop, screen management, camera
├── domain/             # Core game entities and data types
│   ├── card.go         # Card definition, image loading, sale pricing
│   ├── card_loader.go  # JSON card database parsing from embedded assets
│   ├── card_image_fetcher.go # Async card image fetching/caching
│   ├── player.go       # Player state (gold, food, amulets, world magics, deck)
│   ├── character.go    # Base character (sprite animation, movement)
│   ├── enemy.go        # NPC opponents with chase/wander AI
│   ├── collection.go   # Multi-deck card collection management
│   ├── deck.go         # Deck type with ante card filtering
│   ├── starting_deck.go # Deck generation from difficulty/color/seed
│   ├── city.go         # Cities (hamlet/town/capital), shops, amulet colors
│   ├── quest.go        # Quest system (delivery, defeat enemy)
│   ├── amulet.go       # Color amulets (Order/Knowledge/Power/Passion/Life)
│   ├── worldmagic.go   # Purchasable world abilities
│   └── rogue.go        # Enemy character definitions (loaded from TOML configs)
├── screens/            # UI screens (implement screenui.Screen interface)
│   ├── level.go        # World exploration screen
│   ├── world_frame.go  # Status overlay (gold, food, life, amulets)
│   ├── duel.go         # MTG battle screen (hand, board, stack, targeting)
│   ├── duel_ante.go    # Pre-duel ante/wager selection
│   ├── duel_win.go     # Victory rewards
│   ├── duel_lose.go    # Defeat consequences
│   ├── city.go         # City interaction (quests, shopping, NPCs)
│   ├── buycards.go     # Card shop
│   ├── edit_deck.go    # Deck editor with drag-and-drop
│   ├── wiseman.go      # Info/quest giver
│   └── random_encounter.go # Special encounters
├── world/              # World/level generation and management
│   ├── level.go        # Game world (tiles, enemies, cities, encounters)
│   ├── tile.go         # Isometric tile with layered sprites
│   ├── generate.go     # Perlin noise terrain generation
│   ├── autotiling.go   # Terrain sprite selection based on neighbors
│   ├── gen_city.go     # City placement and road generation
│   ├── random_encounters.go # Encounter spawning
│   ├── spritesheet.go  # Sprite loading from embedded assets
│   └── errors.go       # Custom error types
├── minimap/            # Minimap screen
│   └── minimap.go
├── save/               # Save/load game
│   ├── save.go         # SaveGame/LoadGame functions
│   └── types.go        # SaveData struct
├── sprites/            # Sprite-based game objects
│   └── land.go         # Land enchantment sprites
├── brawl/              # Brawl format (future)
│   └── brawl.go
└── ui/                 # UI components
    ├── screenui/consts.go    # Screen interface + screen name enum
    ├── elements/button.go    # Button with states (normal/hover/pressed)
    ├── elements/text.go      # Text rendering
    ├── elements/scrollable_list.go # Scrollable list widget
    ├── dragdrop/             # Drag-and-drop system (interfaces, manager, widgets)
    ├── fonts/text.go         # Font face management
    ├── imageutil/            # Image loading and sprite sheet handling
    ├── layout/anchor.go      # Anchor-based positioning
    └── touch.go              # Touch/mouse input handling
```

## Architecture

### Screen System

All screens implement `screenui.Screen`:
- `Update()` — process input, return next screen name on transition
- `Draw()` — render to screen
- `IsFramed()` — if true, `WorldFrame` draws the status overlay

`game.go` manages screen transitions via a `screenMap` of name → screen instance.

### World

Isometric diamond grid using Perlin noise terrain. Terrain types by noise threshold: Water → Sand → Marsh → Plains → Forest → Mountains → Snow. Tiles have layered sprites (terrain, roads, foliage, cities, encounters).

Coordinate conversion: `TileToPixel()` / `PixelToTile()` for isometric ↔ screen coords.

### Duel Screen

Bridges `game/domain` cards with `mtg/core` game engine. Manages hand display, battlefield layout, targeting UI, combat declarations, and stack visualization. Input flows through `player.InputChan`.

### Domain Entities

- **Player** extends Character with gold, food, amulets, world magics, card collection
- **Enemy** has chase/wander AI movement, level-based bribe cost
- **City** has tier (hamlet/town/capital), card shop, amulet color, quest assignments
- **CardCollection** supports multiple decks, tracks per-deck card counts

## Code Style

- Don't resize images in `Draw()` or `Update()`; resize when creating screens
- Screen transitions return the new screen name from `Update()`
- Assets are embedded via Go `embed` package
