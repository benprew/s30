# Shandalar 30 — 4+1 Architectural View

This document describes the architecture of Shandalar 30 using the [4+1 Architectural View Model](https://en.wikipedia.org/wiki/4%2B1_architectural_view_model).

---

## 1. Logical View

The logical view describes the key abstractions, their responsibilities, and relationships.

### Layer Diagram

```
┌─────────────────────────────────────────────────────┐
│                    main.go                          │
│              (Ebiten bootstrap)                     │
├─────────────────────────────────────────────────────┤
│                   Game                              │
│         (screen router, camera, player)             │
├──────────────┬──────────────┬───────────────────────┤
│   Screens    │     World    │     Domain            │
│  (UI layer)  │  (map/gen)   │  (entities/rules)     │
├──────────────┴──────────────┴───────────────────────┤
│                  UI Toolkit                         │
│  (buttons, lists, drag-drop, fonts, layout, touch)  │
├─────────────────────────────────────────────────────┤
│              External: mage-go                      │
│        (complete MTG rules engine + AI)             │
├─────────────────────────────────────────────────────┤
│              External: Ebiten                       │
│           (rendering, input, audio)                 │
└─────────────────────────────────────────────────────┘
```

### Key Abstractions

**Game** (`game/game.go`) — Top-level coordinator. Owns the screen map, camera state, player reference, and world. Implements `ebiten.Game` interface (`Update`, `Draw`, `Layout`). Routes update/draw calls to the current screen.

**Screen** (`game/ui/screenui/consts.go`) — Central interface for all game states:

```go
type Screen interface {
    Update(W, H int, scale float64) (ScreenName, Screen, error)
    Draw(screen *ebiten.Image, W, H int, scale float64)
    IsFramed() bool
}
```

Screens are a state machine. `Update()` returns the next `ScreenName` and optionally a new `Screen` instance, enabling transitions without screens knowing about each other. Twelve screen types exist:

| Screen | Responsibility |
|--------|---------------|
| `StartScr` | Title/new game menu |
| `WorldScr` | Isometric world exploration |
| `MiniMapScr` | Overview map |
| `CityScr` | City interaction hub (shop, quest, food, deck) |
| `BuyCardsScr` | Card purchasing |
| `EditDeckScr` | Drag-and-drop deck editor |
| `DuelAnteScr` | Pre-duel ante/wager selection |
| `DuelScr` | Active MTG duel (bridges to mage-go) |
| `DuelWinScr` | Victory rewards |
| `DuelLoseScr` | Defeat consequences |
| `WisemanScr` | NPC quest assignments |
| `RandomEncounterScr` | Special terrain encounters |

**Player** (`game/domain/player.go`) — The player's full state: gold, food, amulets (5 MTG colors as bitmask), world magics, active deck, quest, and day counter. Embeds `Character` (identity + sprites) and `CharacterInstance` (position + animation).

**Character / CharacterInstance** (`game/domain/character.go`) — `Character` holds identity (name, tier, level, color identity, deck, sprites, life). `CharacterInstance` holds mutable runtime state (position, direction, animation frame, movement speed). Both players and enemies compose these.

**Enemy** (`game/domain/enemy.go`) — NPC opponent. Composes `Character` + `CharacterInstance`. Has simple chase/wander AI: chases player within 150 units, wanders randomly beyond 200 units.

**Card** (`game/domain/card.go`) — MTG card data: name, mana cost, colors, type, keywords, power/toughness, rarity, price, parsed abilities. Cards are loaded from a compressed Scryfall database and enriched with parsed ability data.

**CardCollection / Deck** (`game/domain/collection.go`, `deck.go`) — `CardCollection` maps cards to counts across multiple deck slots. `Deck` is a simple `map[*Card]int` with ante-card filtering. `GetDuelDeck()` pads with basic lands if below minimum size.

**City** (`game/domain/city.go`) — Tiered settlement (Hamlet/Town/Capital) with cards for sale, an amulet color, optional world magic, and quest cooldown.

**Quest** (`game/domain/quest.go`) — Delivery or defeat-enemy quests with rewards (mana link, amulet, or card).

**Level** (`game/world/level.go`) — The game world: a 47×63 isometric tile grid with enemies, cities, random encounters, and the player. Handles collision detection and encounter triggering.

**mage-go** (external) — Complete MTG rules engine. Handles the stack, priority, combat phases, card abilities, and AI decision-making. The game treats it as a black box, communicating via channels.

### Domain Relationships

```
Player ──owns──> CardCollection ──contains──> Card(s)
Player ──has───> Amulet(s), WorldMagic(s), Quest
Player ──on────> Level ──contains──> Tile(s), Enemy(s), City(s)
Enemy  ──has───> Character ──has──> Deck
City   ──sells─> Card(s)
City   ──offers> Quest, WorldMagic
DuelScreen ──creates──> mage.Game ──uses──> AIPlayer, HumanPlayer
```

---

## 2. Process View

The process view describes concurrency, communication, and runtime behavior.

### Threading Model

Shandalar 30 runs primarily on a **single main goroutine** driven by Ebiten's game loop at **10 TPS** (ticks per second). Additional goroutines handle I/O-bound work.

```
Main Goroutine (Ebiten)
│
├── Game.Update()  ─── CurrentScreen.Update()
│                       └── (input handling, state transitions)
├── Game.Draw()    ─── CurrentScreen.Draw()
│                       └── (rendering to screen buffer)
│
├── [goroutine] MTG Game Loop (during duels)
│   └── interactive.RunGameLoop() — 300ms tick
│       ├── human.ToTUI() channel ──> DuelScreen reads messages
│       └── human.ChoiceRequests() channel ──> DuelScreen reads prompts
│
├── [goroutine pool] Card Image Preloader (6 workers)
│   └── Buffered channel (cap 64) distributes cards to workers
│       └── Each worker: HTTP GET → decode PNG → store in sync.Map
│
└── [ad-hoc goroutines] On-demand card image fetches
    └── Triggered when CardImage() finds uncached card
        └── fetchingSet (sync.Map) prevents duplicate fetches
```

### Synchronization Primitives

| Primitive | Location | Purpose |
|-----------|----------|---------|
| `sync.Map` (×3) | `card_image_fetcher.go` | `cardImages` cache, `fetchingSet` dedup, `labeledBlankCards` cache |
| `sync.Once` | `card_image_fetcher.go` | Lazy decode of blank card template |
| `sync.WaitGroup` | `card_image_fetcher.go` | Wait for 6 preload workers to finish |
| Channels | `duel.go` / mage-go | Game state messages and choice requests between MTG engine and UI |
| Buffered channel (64) | `card_image_fetcher.go` | Card distribution to worker pool |

### Communication Pattern: Duel

The duel screen uses a **producer-consumer** pattern via Go channels:

```
┌──────────────────────┐          ┌──────────────────────┐
│   MTG Game Loop      │          │    DuelScreen         │
│   (goroutine)        │          │    (main goroutine)   │
│                      │          │                       │
│  human.ToTUI() ──────┼─ chan ──>│  drainMessages()      │
│  ChoiceRequests() ───┼─ chan ──>│  drainChoiceRequests() │
│                      │          │                       │
│  <── action ─────────┼─ chan ──<│  player input          │
└──────────────────────┘          └───────────────────────┘
```

`DuelScreen.Update()` performs **non-blocking** channel reads (`select/default`) so the UI never stalls waiting for the engine.

### Lifecycle

```
Program start
  │
  ├── Parse flags (-v mtg,duel)
  ├── Ebiten window setup (1024×768, resizable, 10 TPS)
  ├── game.NewGame()
  │     ├── LoadCardDatabase() — decompress zstd, parse JSON
  │     ├── LoadParsedAbilities() + ApplyParsedAbilities()
  │     ├── Load rogues from TOML configs
  │     ├── Generate world (Perlin noise terrain, cities, roads)
  │     ├── Create player (starting deck by color + difficulty)
  │     └── go PreloadCardImages() — background fetch
  │
  ├── ebiten.RunGame(g)
  │     └── Main loop: Update() → Draw() → repeat at 10 TPS
  │
  └── Exit on window close or fatal error
```

---

## 3. Development View

The development view describes source code organization, build structure, and dependencies.

### Package Structure

```
s30/
├── game/                  Core game package
│   ├── domain/            Entity models (Card, Player, Enemy, City, Quest, Deck,
│   │                      Amulet, WorldMagic, Rogue). No UI dependencies — the
│   │                      innermost layer. Includes card DB loading and async
│   │                      image fetching.
│   ├── screens/           One Screen implementation per game state (world
│   │                      exploration, city, duel, deck editor, shop, quests,
│   │                      win/lose, random encounters). Heaviest package — the
│   │                      duel screen bridges to the mage-go MTG engine.
│   ├── world/             Procedural world generation (Perlin noise terrain),
│   │                      isometric tile grid, autotiling, city/road placement,
│   │                      enemy spawning, and random encounter distribution.
│   ├── ui/                Reusable UI toolkit, subdivided into:
│   │   ├── screenui/      Screen interface and ScreenName enum
│   │   ├── elements/      Widgets: Button, ScrollableList, Text
│   │   ├── dragdrop/      Drag-and-drop system (deck editor)
│   │   ├── fonts/         Font face management (Planewalker TTF)
│   │   ├── imageutil/     Image loading and spritesheet slicing
│   │   └── layout/        Anchor-based positioning
│   ├── minimap/           Minimap overview screen
│   ├── save/              Game save/load serialization
│   └── sprites/           Land enchantment sprite helpers
│
├── assets/                All game data, compiled into the binary via go:embed.
│                          Card DB (zstd-compressed Scryfall JSON), parsed
│                          abilities, sprites organized by screen, Planewalker
│                          fonts, rogue TOML configs, and game text.
│
├── cmd/                   Standalone CLI tools (AI-vs-AI simulation, tile
│                          transition helper)
│
├── mobile/                Ebiten mobile binding entry point + full Android
│                          project (Gradle, manifest, MainActivity)
│
├── logging/               Subsystem-toggled logging (mtg, world, duel)
├── utils/                 Python scripts for card image fetching and parsing
└── docs/                  Documentation
```

### Dependency Graph

```
main.go
  └── game/
        ├── game/domain/         (no game/ imports — leaf package)
        │     └── assets/        (embedded data)
        ├── game/screens/
        │     ├── game/domain/
        │     ├── game/world/
        │     ├── game/ui/
        │     └── mage-go        (MTG engine)
        ├── game/world/
        │     ├── game/domain/
        │     └── game/ui/imageutil/
        ├── game/ui/             (no domain imports — pure UI toolkit)
        ├── game/minimap/
        ├── game/save/
        └── game/sprites/
```

`game/domain/` is the innermost layer — it depends only on `assets/` for embedded data and standard library. Screens depend on domain, world, and UI. This creates a clean layering where domain logic is testable in isolation.

### External Dependencies

| Dependency | Purpose |
|------------|---------|
| `ebiten/v2` v2.9.9 | Game engine (rendering, input, audio, mobile) |
| `mage-go` (forked) | Complete MTG rules engine with AI |
| `go-perlin` v1.1.0 | Perlin noise for terrain generation |
| `compress` v1.18.0 | Zstd decompression for card database |
| `toml` v1.5.0 | Rogue config file parsing |
| `uuid` v1.6.0 | Unique identifiers for game objects |
| `x/image` v0.31.0 | Image format support |

The `mage-go` dependency uses a `replace` directive pointing to a personal fork (`~benprew/mage-go`), allowing custom MTG engine modifications.

### Test Strategy

Tests run with `go test -count=10 ./...` (10 iterations to catch flakes). Test files colocate with source:

- `game/domain/` — Unit tests for cards, decks, players, cities, amulets, rogues, starting decks
- `game/screens/` — Integration tests for duel (attackers, blockers, autopass, mage integration), ante, buy cards, deck editor, wiseman
- `game/world/` — Tests for level generation, autotiling, amulet placement
- `cmd/mtg_test/` — Standalone AI-vs-AI game simulation

---

## 4. Physical View

The physical view describes how the software maps to hardware and deployment targets.

### Deployment Topology

```
┌─────────────────────────────────────────────────┐
│                  Build Host                     │
│            (GitHub Actions runner)               │
│                                                 │
│  ┌──────────┐ ┌──────────┐ ┌────────────────┐  │
│  │ Linux    │ │ macOS    │ │ Windows        │  │
│  │ x86_64   │ │ Intel    │ │ x86_64         │  │
│  │ ARM64    │ │ ARM64    │ │ ARM64          │  │
│  └────┬─────┘ └────┬─────┘ └───────┬────────┘  │
│       │             │               │            │
│  ┌────┴─────────────┴───────────────┴────────┐  │
│  │         GitHub Release Artifacts          │  │
│  └───────────────────────────────────────────┘  │
│                                                 │
│  ┌──────────┐  ┌───────────────────────────┐   │
│  │ Android  │  │ WebAssembly               │   │
│  │ APK      │  │ s30.wasm + main.html      │   │
│  └────┬─────┘  └────────────┬──────────────┘   │
│       │                     │                    │
└───────┼─────────────────────┼────────────────────┘
        │                     │
        ▼                     ▼
   Android Device     teamvite.com web server
                      /var/www/html/throwingbones/ben/s30/
```

### Platform Build Matrix

| Target | OS | Arch | Output | CGO |
|--------|----|------|--------|-----|
| Linux desktop | linux | amd64 | `s30` | Yes |
| Linux ARM | linux | arm64 | `s30` | Yes |
| macOS Intel | darwin | amd64 | `s30_mac` | Yes |
| macOS Apple Silicon | darwin | arm64 | `s30_mac_arm` | Yes |
| Windows | windows | amd64 | `s30.exe` | Yes |
| Windows ARM | windows | arm64 | `s30_arm64.exe` | Yes |
| WebAssembly | js | wasm | `s30.wasm` | No |
| Android | android | multi-arch | `app-release.apk` | Yes (NDK) |

### Asset Packaging

All game assets (sprites, fonts, card data, configs) are **compiled into the binary** via Go's `embed` package. The resulting executable is fully self-contained — no external asset files needed at runtime.

The one exception is **card artwork**: card images are fetched at runtime from the Scryfall API via HTTP and cached in memory (`sync.Map`). This is the only network dependency.

```
Binary (self-contained)
├── Embedded: sprites, fonts, card DB, configs, UI art
└── Runtime fetch: card artwork from Scryfall API (HTTP GET)
                   └── Cached in sync.Map (memory only)
```

### Android Deployment

```
ebitenmobile bind → s30.aar (Go library)
         │
         ▼
  Gradle build (SDK 34, min SDK 23)
         │
         ▼
  APK (com.benprew.s30)
  ├── MainActivity (lifecycle management)
  ├── Requires OpenGL ES 3.2
  └── Landscape-only orientation
```

### WebAssembly Deployment

```
GOOS=js GOARCH=wasm go build → s30.wasm
         │
         ▼
  main.html (loads wasm_exec.js + s30.wasm)
         │
         ▼
  SCP to throwingbones.com:/var/www/html/throwingbones/ben/s30/
```

---

## +1. Scenarios

Scenarios (use cases) tie the four views together by tracing key user interactions through the system.

### Scenario 1: World Exploration

**User action:** Player moves through the isometric world using keyboard/mouse.

```
Input (Ebiten)
  │
  ▼
Game.Update()
  │
  ▼
LevelScreen.Update()
  ├── Player.Move()          — reads input, updates CharacterInstance position
  ├── Player.Update()        — animation frame, time/food tracking
  ├── Level.UpdateWorld()    — updates all enemies (chase/wander AI)
  │     └── Enemy collision detection
  │           ├── City tile? → return CityScr
  │           └── Enemy engaged? → return DuelAnteScr
  └── return WorldScr (no transition)
  │
  ▼
Game.Draw()
  ├── LevelScreen.Draw()     — renders visible tiles (isometric projection)
  │     ├── Tile.sprites      (terrain base)
  │     ├── Tile.roadSprites  (roads)
  │     ├── Tile.positionedSprites (foliage, cities)
  │     ├── Enemy sprites     (walking animation)
  │     └── Player sprite     (walking animation)
  └── WorldFrame.Draw()      — status bar (gold, food, life, amulets)
```

### Scenario 2: MTG Duel

**User action:** Player collides with enemy, selects ante card, plays a full MTG duel.

```
1. LevelScreen detects enemy collision
   └── returns DuelAnteScr with new DuelAnteScreen(enemy, player, level)

2. DuelAnteScreen
   ├── Displays enemy character + catchphrase
   ├── Player selects ante card (or bribes to avoid)
   └── On "Duel" button → returns DuelScr with new DuelScreen

3. DuelScreen.initGameState()
   ├── Creates mage.Game with MTG rules
   ├── Adds player's deck cards to library
   ├── Adds AI's deck cards to library
   └── Launches goroutine: interactive.RunGameLoop(game, 300ms)

4. Duel loop (runs at 10 TPS):
   ┌─────────────────────────────────┐
   │ DuelScreen.Update()             │
   │   ├── drainMessages()           │  ← non-blocking read from ToTUI()
   │   │     └── updates hand, board, phase display
   │   ├── drainChoiceRequests()     │  ← non-blocking read from ChoiceRequests()
   │   │     └── creates choice buttons
   │   ├── Handle player input:      │
   │   │     ├── Click card in hand → show actions
   │   │     ├── Select action → send to human player
   │   │     ├── Declare attackers → pendingAttackers map
   │   │     ├── Declare blockers → pendingBlockers map
   │   │     └── Target selection → selectedTargetID
   │   └── Check game.IsGameOver()   │
   │         ├── Player won → return DuelWinScr
   │         └── Player lost → return DuelLoseScr
   └─────────────────────────────────┘

5. DuelWinScreen
   ├── Awards ante card from enemy
   ├── Updates player gold/rewards
   ├── Removes enemy from Level
   └── returns WorldScr → back to exploration
```

### Scenario 3: Deck Building

**User action:** Player enters a city, opens deck editor, rearranges cards.

```
1. LevelScreen → player on city tile → returns CityScr

2. CityScreen
   ├── Shows city name, tier, available actions
   └── "Edit Deck" button → returns EditDeckScr

3. EditDeckScreen
   ├── Left panel: CardCollection (ScrollableList)
   │     └── Grouped by card name, shows count per deck
   ├── Right panel: Current deck cards (DeckCardDisplay grid)
   ├── DragManager handles:
   │     ├── Drag from collection → drop on deck area → AddCardToDeck()
   │     ├── Drag from deck → drop on collection → MoveCardFromDeck()
   │     └── Double-click shortcuts for add/remove
   ├── Deck size validation (minimum enforced)
   └── "Done" button → returns CityScr

4. On next duel, Player.GetDuelDeck() returns the modified deck
   └── Pads with basic lands if below minimum size
```

### Scenario 4: Quest Completion

**User action:** Player accepts quest from wiseman, completes it, collects reward.

```
1. CityScreen → "Quest" button → returns WisemanScr

2. WisemanScreen
   ├── If no active quest: offers new quest
   │     ├── QuestType: Delivery (go to target city) or DefeatEnemy
   │     ├── Reward: ManaLink, Amulet, or Card
   │     └── DaysRemaining countdown
   ├── If active quest completed:
   │     ├── Awards reward to player
   │     │     ├── ManaLink → city.IsManaLinked = true
   │     │     ├── Amulet → player.AddAmulet(color)
   │     │     └── Card → add to collection
   │     └── Clears quest, sets QuestBanDays on city
   └── returns CityScr

3. Quest progress tracked in Player.ActiveQuest
   └── Delivery: checked when entering target city
   └── DefeatEnemy: checked when winning duel against target
```

### Scenario 5: New Game Setup

**User action:** Player starts a new game, choosing color and difficulty.

```
1. StartScreen
   ├── Color selection (White/Blue/Black/Red/Green)
   ├── Difficulty selection (Easy/Medium/Hard/Expert)
   └── "Start" → triggers game initialization

2. Game.NewGame()
   ├── LoadCardDatabase()
   │     └── Decompress zstd → parse JSON → 30k cards sorted by name
   ├── LoadParsedAbilities() + ApplyParsedAbilities()
   │     └── Enriches cards with structured ability data
   ├── LoadRogues()
   │     └── Parse TOML configs → Character structs with decks
   ├── world.GenerateLevel()
   │     ├── Perlin noise → terrain types (water/sand/plains/forest/mountain/snow)
   │     ├── Autotiling → sprite selection by neighbors
   │     ├── GenCities() → place cities with roads
   │     ├── SpawnEnemies() → distribute rogues on map
   │     └── SpawnEncounters() → place random encounters
   ├── NewPlayer(color, difficulty)
   │     ├── Starting gold (250/200/150/100 by difficulty)
   │     ├── Min deck size (30/35/40/40)
   │     └── GenerateStartingDeck(color) → themed 40-card deck
   └── go PreloadCardImages(priorityCards)
         └── 6 workers fetch card art from Scryfall in background
```
