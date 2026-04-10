# Duel UI Screen Implementation Plan

## Context

Build the duel game screen shown in `duel_ui.png`. This is the screen displayed during an actual MTG duel between the player and an opponent. Currently, `DuelAnteScreen.startDuel()` randomly decides win/loss - this will instead transition to the new DuelScreen.

## Layout (1024x768)

```
|  Left Sidebar (240px)  |        Main Play Area (784px)              |
|                        |                                            |
| Opponent Life: 20      |   Opponent Board (art background)          |
| Mana: W0 U0 B0 R0 G0  |                          Opponent (7)      |
| [deck] [graveyard]     |                                            |
|                        |                                            |
| Phase Buttons:         |                                            |
|  [Untap]               |-------------------------------------------|
|  [Upkeep]              | Done | Main phase (before combat): cast... |
|  [Draw]                |-------------------------------------------|
|  [Main 1]  <-- active  |                                            |
|  [Combat]              |   Player Board (tiled mana bg)             |
|  [Main 2]              |                          Your hand (7)     |
|  [End]                 |   Unholy Strength        [card preview]    |
|                        |   Gray Ogre                                |
| Player Life: 20        |   Giant Spider                             |
| Mana: W0 U0 B0 R0 G0  |   Plains                                   |
| [deck] [graveyard]     |                                            |
```

## Files to Create

### 1. `game/screens/duel.go` (main file)

**DuelScreen struct:**
- `player *domain.Player`, `enemy *domain.Enemy`, `lvl *world.Level`, `enemyIdx int`
- `gameState *core.GameState`, `corePlayer *core.Player`, `coreOpponent *core.Player`
- Pre-rendered images: `sidebarBg`, `opponentBoardBg`, `playerBoardBg`
- Mana symbol sprites from Statbutt.spr.png (5 icons, scaled to ~16px at load time)
- Done button (3 states from Statbutt.spr.png indices 11-13)
- Phase button data (names + highlight state)
- `selectedCardIdx int`, `cardPreviewImg *ebiten.Image`

**Constructor `NewDuelScreen(player, enemy, lvl, idx)`:**
- Bridge domain decks to core.Player/core.Card using `core.NewCardFromDomain()` (pattern from `cmd/mtg_test/main.go`)
- Initialize `core.GameState`, call `StartGame()` to draw 7 cards each
- Load and pre-scale all images (Statbutt sprites, backgrounds)
- Build opponent background from `loadBackgroundForEnemy()` scaled to play area
- Build player background using `imageutil.TileImage()` with a mana symbol tile
- Build sidebar as a solid tan/stone colored rectangle

**Draw():** Render sidebar, both boards, phase bar, hand list, card preview
**Update():** Handle Done click (advance phase), hand card hover/click, Escape to exit
**IsFramed():** Returns `false`

### 2. `game/ui/screenui/consts.go` (modify)

Add `DuelScr` to ScreenName enum and ScreenNameToString.

### 3. `assets/embed.go` (modify)

Add embed for `Statbutt_png` (already exists at `art/sprites/Statbutt.spr.png`).

### 4. `game/screens/duel_ante.go` (modify)

Change `startDuel()` to return `screenui.DuelScr, NewDuelScreen(s.player, s.enemy, s.lvl, s.idx), nil` instead of random win/loss. The win/loss logic moves to DuelScreen's game-over handling later.

## Key Implementation Details

**Domain-to-Core bridge** (follows `cmd/mtg_test/main.go` pattern):
```
deck := player.GetActiveDeck()  // map[*Card]int
for card, count := range deck {
    for range count {
        coreCard := core.NewCardFromDomain(card, entityID, corePlayer)
        corePlayer.Library = append(corePlayer.Library, coreCard)
        entityID++
    }
}
rand.Shuffle(len(corePlayer.Library), ...)
```

Sprites categories in assets/art/sprites/duel
- Grave_* empty graveyard images, 1 per color
- Hand_* container image to hold cards in players hand, 1 per color
- Terr_* player board backgrounds
- Winbk_Manapool - players mana pool background image
- Winbk_Phase - list of player's phases
- Winbk_Spellchain - current stack of spells/abilities waiting to be processed
- Abilities - list of abilities/keyword badges to add to the card (flying, deathtouch, etc)
- Cardcounters - counters that can be added to cards (ex. +1/+1, haste, etc)

**Statbutt.spr.png** (768x48, 16 columns of 48x48):
- Indices 0-4: Mana symbols (Black skull, Blue drop, Red mountain, Green tree, White sun)
- Indices 11-13: DONE button (normal, hover, pressed)
- Load with `imageutil.LoadSpriteSheet(16, 1, assets.Statbutt_png)`
- Scale mana symbols to ~16x16 at load time

**Sidebar background:** Simple filled rectangle (tan/stone color ~RGB(180,160,130)). Can integrate Prdfrmc.pic.png sprites in a later polish pass.

**Player board tiled bg:** Create a small tile with mana symbols on dark gray, then use `imageutil.TileImage()`.

**Phase descriptions:**
- Main1: "Main phase (before combat): cast spells, play land"
- Combat: "Combat phase: declare attackers and blockers"
- Main2: "Main phase (after combat): cast spells, play land"
- Others: brief descriptions

**Initial scope (visual only):**
- Display all UI regions correctly positioned
- Show real data: life totals, hand card count, card names from initialized GameState
- Done button advances phase indicator through the cycle
- Hand card hover shows card preview image
- Escape transitions to WorldScr (temporary exit)

## Verification

1. `make` - Run the game, trigger a duel encounter, verify the DuelScreen appears
2. `golangci-lint run` - No new warnings
3. `make test` - All existing tests pass
