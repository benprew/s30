# Dungeon System Design Doc

## Overview

Dungeons are fixed, hidden locations on the world map containing the game's most powerful restricted cards as rewards. Unlike random encounters that spawn dynamically, dungeons are placed once during world generation and persist until cleared. Entering a dungeon shifts gameplay from open-world exploration to turn-based grid movement through narrow hallways with static enemies, treasure, dice, and scrolls.

## Dungeon Discovery

### Clue System

Players learn about dungeons through three sources:
- **Creatures** (defeated enemies on the world map)
- **Wise Men** (in cities, via the existing `WisemanScreen`)
- **Lairs** (a type of random encounter)

Each dungeon has **three clues**:

| Clue | Description | Example |
|------|-------------|---------|
| Location | Approximate map position | "A dungeon lies in the forests to the north" |
| Population | Size and color of creatures inside | "It is filled with small black creatures" |
| Effect | The enchantment/artifact in play during duels | "A Meekstone holds sway within" |

Clues are revealed one at a time. The player must gather clues to locate dungeons on the map.

### Dungeon Placement

- Fixed number of dungeons per game (e.g., 5-8), placed during `GenerateLevel()`
- Placed on valid terrain tiles, away from cities, spread across the map
- Visually hidden until the player walks over them (or optionally revealed by clues)
- If a player leaves a dungeon without collecting all restricted cards, the dungeon **relocates** to a random map position and the player must obtain new clues to find it again

## Dungeon Structure

### Grid Layout

Dungeons are tile-based grids of narrow hallways. Unlike the isometric world map, dungeon movement is **turn-based**: the player moves one tile per turn, and nothing else moves (enemies are static).

```
+---+---+---+---+---+---+---+
| E |   |   | D |   |   | T |
+---+   +---+---+---+   +---+
|   |   | S |   |   |   |   |
+---+---+---+   +---+---+---+
| T |   | E |   |   | D |   |
+---+   +---+---+---+---+   |
|   |   |   |   | E |   | T |
+---+---+---+---+---+---+---+

E = Enemy    T = Treasure chest
D = Dice     S = Scroll
```

Hallways branch and dead-end. The player chooses paths, encountering events along the way. Some paths can be avoided by taking alternate routes, but restricted card rewards may require fighting through enemies.

### Fog of War

The dungeon is **not** fully visible on entry. The player can only see tiles in their direct line of sight -- they cannot see around corners or past enemies.

**Visibility rules:**
- The player has line-of-sight down straight hallways in all four cardinal directions
- Corners and T-junctions block visibility beyond the turn
- Enemies block visibility -- the player cannot see what is behind an enemy until it is defeated
- Once a tile has been seen, it stays revealed (no re-fogging)
- Unrevealed tiles render as solid black or a dark wall texture

**Tile states:**

| State | Rendering | When |
|---|---|---|
| Hidden | Black / not drawn | Never been in line of sight |
| Revealed | Fully lit, shows contents | Has been in line of sight at any point |

Since enemies are static and nothing moves except the player, there is no need for a third "previously seen but not currently visible" state. Once revealed, a tile stays revealed and fully visible for the rest of the dungeon visit.

**Line-of-sight algorithm:**

Since dungeons are narrow grid hallways (not open rooms), a simple raycasting approach works:

1. From the player's position, cast rays in each cardinal direction (N, S, E, W)
2. Walk tile-by-tile along the ray
3. Mark each tile as `Revealed = true`
4. Stop the ray when hitting: a wall, an enemy, or the grid boundary
5. At junctions, the player can see the junction tile itself but not down the branching corridor until they step into it

This runs once each time the player moves (turn-based, so no per-frame cost).

**Gameplay impact:**
- Forces the player to explore cautiously -- unknown branches could contain enemies or rewards
- Enemies blocking sight create tension: the player doesn't know what's behind them
- After clearing an enemy, the next move reveals the corridor behind it
- Encourages strategic backtracking to explore alternate paths

### Scaling by Difficulty Level

Higher difficulty levels increase:
- Number of enemies in the dungeon
- Strength of enemies (higher-level rogues)
- Severity of dice disadvantages
- Number of scroll answer choices
- Card type restrictions during duels

## Dungeon Events

### 1. Creatures (Enemy Duels)

Enemies block hallways. The player must either fight or backtrack.

**Duel rules inside dungeons differ from overworld duels:**
- **No ante from enemies.** The player does not win cards from defeating dungeon creatures.
- **Losing costs ante + eviction.** If the player loses, they lose their ante card AND are ejected from the dungeon.
- **Dungeon enchantment in play.** Every duel starts with a pre-set enchantment or artifact on the battlefield (see Dungeon Effects below).
- **Card restrictions may apply.** On higher levels, certain card types are forbidden (see Card Restrictions below).
- **Life carries over** between dungeon fights (not reset to starting life).

Enemy color and size are consistent within a dungeon (matching the "population" clue).

### 2. Treasure Chests

Treasure chests are placed at hallway dead-ends or behind enemies.

| Dungeon Type | Possible Contents |
|---|---|
| Dungeon | Restricted card OR gold + amulets |
| Castle | Gold + amulets only (no restricted cards) |

Restricted cards are the primary motivation for dungeon exploration. Each dungeon contains a fixed set of restricted cards spread across its treasure chests.

### 3. Dice

Dice grant temporary advantages or disadvantages for the **next duel only** within the current dungeon. Effects are lost if the player leaves the dungeon.

**Advantages:**
- **+X life.** Additive bonus to the player's life for dungeon duels. Easier levels grant +3-4, harder levels grant +1-2.
- **Start with a card.** A card begins in play at the start of the duel. Easier levels give powerful creatures; harder levels give weaker creatures or moxen/mana creatures.

**Disadvantages (higher levels only):**
- **-X life.** Reduces player life for the next duel.

On the easiest difficulty, dice never give disadvantages.

**Important:** Advantages brought into a dungeon from outside (e.g., world magic effects) persist. Only dungeon-granted advantages are lost on exit.

### 4. Scrolls (Trivia)

Scrolls present MTG trivia questions about cards in the game. Answering correctly clears the scroll. Answering incorrectly **replaces the scroll with an enemy** that must be fought.

**Question types:**
- "Which of these creatures has flying?"
- "What special ability does [creature] have?"
- "What is the casting cost / power / toughness of [creature]?"

**Rules:**
- Questions use only the **base printed abilities** of a card. Activated abilities don't count (e.g., Goblin Balloon Brigade does not have flying).
- Easier levels: 5 answer choices
- Harder levels: more choices (6-8)

**Data source:** Card data is already available in `assets/card_info/cards.json` with keywords, power, toughness, and mana cost.

## Dungeon Effects

Each dungeon has a **color-aligned enchantment or artifact** that is in play at the start of every duel within that dungeon. This matches the dungeon's color.

Requirements:
- Must be an enchantment matching the dungeon's color, OR a non-creature artifact without upkeep
- Placed on the battlefield before the game starts (not cast, so it can't be countered)
- Affects both players

Examples: Meekstone, The Abyss, Gloom, Blood Moon, Karma, Crusade, Bad Moon.

### Card Restrictions (Higher Levels)

On higher difficulty levels, dungeons may forbid certain card types:
- A color (e.g., "no white cards")
- A card type (e.g., "no fast effects / instants")

**Mechanic:** Forbidden cards are removed from the player's deck before the duel. The deck is **not** backfilled to minimum size -- if restrictions drop the deck below `MinDeckSize`, the player duels with a smaller deck.

This integrates with the existing `Player.GetDuelDeck()` method, adding a filter step before the minimum-size land fill (or rather, the restriction is applied after deck retrieval and no fill-up occurs).

## Data Model

### Dungeon Definition

```go
type Dungeon struct {
    Name             string
    Level            int            // difficulty tier
    Color            ColorMask      // dungeon color (W/U/B/R/G)
    Grid             [][]DungeonTile
    Enchantment      *Card          // card in play during all duels
    CardRestriction  *CardRestriction // nil or restriction for higher levels
    CreatureSize     CreatureSize   // Small or Large
    RestrictedCards  []*Card        // rewards available
    MapTile          TilePos        // current world map position
    Cleared          bool           // all restricted cards collected
    Clues            [3]DungeonClue
}

type DungeonTile struct {
    Type    DungeonTileType // Empty, Wall, Enemy, Treasure, Dice, Scroll
    Enemy   *Character      // if enemy tile
    Reward  *DungeonReward  // if treasure tile
    Scroll  *ScrollQuestion // if scroll tile
    Dice    *DiceEffect     // if dice tile
    Visited bool
}

type DungeonReward struct {
    Type          RewardType // RestrictedCard or GoldAmulets
    Card          *Card      // if restricted card
    Gold          int        // if gold
    Amulets       []Amulet   // if amulets
}

type DiceEffect struct {
    Type     DiceType  // Advantage or Disadvantage
    LifeMod  int       // +/- life
    Card     *Card     // start-with card (nil if life mod)
}

type ScrollQuestion struct {
    Question string
    Choices  []string
    Answer   int
}

type CardRestriction struct {
    ForbiddenColor *ColorMask // nil or forbidden color
    ForbiddenType  *string    // nil or forbidden card type
}
```

### Player Dungeon State

```go
// Add to existing Player struct
type DungeonState struct {
    CurrentDungeon  *Dungeon
    Position        TilePos         // player position in dungeon grid
    DungeonLife     int             // life total carrying over between fights
    DiceAdvantages  []DiceEffect    // active advantages from dice
    CollectedCards  []*Card         // restricted cards found this visit
}
```

### Clue Tracking

```go
type DungeonClue struct {
    Type     ClueType // Location, Population, Effect
    Text     string
    Revealed bool
}

// Add to Player
type PlayerClues struct {
    RevealedClues map[string][]DungeonClue // dungeon name -> revealed clues
}
```

## Screen Flow

```
LevelScreen
  ├─→ (step on dungeon tile) → DungeonEntryScreen
  │     └─→ (Enter) → DungeonScreen (turn-based grid)
  │           ├─→ (step on Enemy) → DuelAnteScreen → DuelScreen
  │           │     ├─→ Win → back to DungeonScreen (life carries over)
  │           │     └─→ Lose → lose ante, ejected → LevelScreen
  │           ├─→ (step on Treasure) → DungeonRewardScreen → back to DungeonScreen
  │           ├─→ (step on Dice) → DungeonDiceScreen → back to DungeonScreen
  │           ├─→ (step on Scroll) → DungeonScrollScreen
  │           │     ├─→ Correct → cleared → back to DungeonScreen
  │           │     └─→ Wrong → becomes Enemy → DungeonScreen
  │           └─→ (Leave / reach exit) → LevelScreen
  │                 (if not all cards collected, dungeon relocates)
  │
  ├─→ (Wiseman / Creature / Lair gives clue) → clue revealed in PlayerClues
```

## New Screens Needed

| Screen | Purpose |
|---|---|
| `DungeonEntryScreen` | Shows dungeon name, known clues, confirm entry |
| `DungeonScreen` | Turn-based grid movement, renders dungeon map, player position, visible events |

Treasure chests, dice rolls, and scroll questions are rendered as **overlays** on top of the `DungeonScreen`, not as separate screens. This keeps the player oriented in the dungeon while interacting with events -- similar to how `BuyCardsScreen` shows a large card preview overlay for purchase confirmation.

| Overlay | Contents |
|---|---|
| Treasure overlay | Shows reward (restricted card image or gold/amulet summary), confirm button to collect |
| Dice overlay | Shows roll result and effect (+/- life or start-with card), confirm button to continue |
| Scroll overlay | Shows trivia question with multiple choice answer buttons |

## Duel Modifications for Dungeons

The existing duel system needs these changes when fighting inside a dungeon:

1. **Pre-place enchantment.** Before `gameState.StartGame()`, add the dungeon's enchantment to the battlefield.
2. **Apply card restrictions.** Filter the player's deck before creating the `core.Player`, removing forbidden cards. Do NOT fill back to `MinDeckSize`.
3. **No enemy ante rewards.** On win, do not call `selectRewardCards()`.
4. **Life persistence.** Use `DungeonState.DungeonLife` instead of `Player.Life` as starting life. Update it after each duel.
5. **Apply dice effects.** Add life modifier and/or start-with card. Clear single-use dice effects after the duel.
6. **Loss handling.** On loss, remove ante card, clear `DungeonState`, return to `LevelScreen`, and relocate the dungeon if incomplete.

## Dungeon Generation

During `GenerateLevel()`:

1. Determine number of dungeons based on difficulty
2. Distribute restricted cards across dungeons
3. For each dungeon:
   - Pick a color (distribute across 5 colors)
   - Select an appropriate enchantment/artifact for that color
   - Generate a grid layout (hallways, dead-ends, branching paths)
   - Place enemies matching dungeon color and size, scaled by level
   - Place treasure chests (restricted cards at harder-to-reach locations)
   - Place dice and scrolls in intermediate positions
   - Place on world map at a valid terrain tile
   - Generate three clues

### Grid Generation Algorithm

1. Start with a filled grid (all walls)
2. Carve a main corridor from entrance
3. Branch off side corridors at random intervals
4. Place dead-ends at corridor terminuses
5. Place treasure at dead-ends, enemies along corridors
6. Ensure at least one path exists to each restricted card (but may require fighting enemies)
7. Ensure the dungeon is completable (no unreachable rewards)

## Integration with Existing Systems

### Wiseman Clue Delivery

Extend `WisemanScreen` with a new state for dungeon clues. When no quest is active, the wiseman may offer a dungeon clue instead of a quest (or in addition to stories).

### Enemy Clue Delivery

After winning a duel on the world map, there's a chance the defeated enemy drops a dungeon clue (shown on `DuelWinScreen`).

### Rogue Configs for Dungeon Enemies

Dungeon enemies can reuse existing rogue TOML configs, filtered by color. "Small creatures" use lower-level rogues of that color; "large creatures" use higher-level rogues.

### World Map Rendering

Dungeons need a hidden/revealed sprite on the world map. When a player has the location clue, the dungeon becomes visible (or a general area is highlighted on the minimap).

## Implementation Order

1. Define dungeon data structures (`game/domain/dungeon.go`)
2. Implement dungeon grid generation
3. Add dungeon placement to `GenerateLevel()`
4. Create `DungeonScreen` with turn-based grid movement
5. Create `DungeonEntryScreen`
6. Wire dungeon duels with enchantment pre-placement and card restrictions
7. Implement treasure, dice, and scroll event screens
8. Add life carryover and dungeon ejection on loss
9. Add dungeon relocation on incomplete exit
10. Integrate clue system into wiseman, enemy rewards, and lairs
11. Add scroll trivia question generation from card data
12. Balance enemy counts, dice values, and card restrictions per difficulty level
