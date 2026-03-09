# Tile Transition System Design Doc

## Overview

Smooth terrain transitions between adjacent tiles on the isometric world map. When two different terrain types meet (e.g., plains next to water), overlay sprites blend the boundary instead of showing a hard edge. The autotiling logic and sprite metadata already exist -- this doc covers integrating them into the main game.

## Current State

### What exists

- **Autotiling logic** (`game/world/autotiling.go`) -- `GetTransitions()` determines which edges of a tile need transition overlays based on neighboring terrain types. Tested in `autotiling_test.go`.
- **Transition sprite sheets** -- `Cstline1.spr.png` (4x7, 28 sprites) and `Cstline2.spr.png` (4x14, 56 sprites), each sprite 102x52 pixels (half-tile sized).
- **Sprite metadata** (`assets/art/sprites/world/land/Cstline_map.json`) -- maps each sprite to its edge coverage and compatible pairing sprites.
- **Prototype tool** (`cmd/tile_transitions/`) -- standalone Ebiten app that generates random 5x5 grids, renders transitions, and outputs PNGs for validation.
- **Python analysis tools** (`cmd/tile_transitions/analyze_edges.py`, `visualize_connections.py`, `visualize_pairs.py`) -- extract edge data from sprites and generate the JSON metadata.

### What doesn't exist yet

The main game (`game/screens/level.go`, `game/world/tile.go`) does **not** use any of this. Tiles currently render as: base terrain sprite + foliage overlay. No coastline/transition sprites are drawn.

## How Autotiling Works

### Coordinate System

Tiles are arranged in a zigzag isometric grid. Each tile is a diamond (206x102 pixels) with 4 corners:

```
        Corner 0 (Top)
       /              \
Corner 3 (Left)    Corner 1 (Right)
       \              /
        Corner 2 (Bottom)
```

Four edges: NW (0-3), NE (0-1), SE (1-2), SW (2-3). Neighbor offsets differ for even vs odd rows due to the zigzag layout.

### Transition Detection

`GetTransitions(pos, tileMap)` checks all 4 neighbors. A transition is needed when `currentTile.TerrainType > neighbor.TerrainType` (higher-priority terrain overlaps onto lower). This one-directional rule prevents drawing duplicates on both sides.

Each `Transition` includes:

```go
type Transition struct {
    Side       string   // "NW", "NE", "SE", "SW"
    SourceType TileType // The neighbor's terrain type
    Open1      bool     // Is corner 1 of this edge shared with another transition?
    Open2      bool     // Is corner 2 of this edge shared with another transition?
}
```

The `Open` flags indicate whether a corner is "open" (the adjacent tile around that corner also has a transition), meaning the overlay sprite should connect at that corner rather than terminate.

### Sprite Pair Selection

Each transition edge is covered by **two half-tile sprites** placed at the boundary. The algorithm (`findSmartTransitionPair` in the prototype):

1. Find all candidate sprites whose "full" edges match the required direction
2. For each candidate pair, check connectivity in `Cstline_map.json`
3. Score pairs based on corner matching:
   - Open corner + sprite connects at that corner = good
   - Closed corner + sprite doesn't connect = good
   - Mismatches reduce score
4. Select the highest-scoring pair

### Sprite Positioning

Each half-tile sprite (102x52) is positioned at the midpoint of the tile edge:

| Edge | Sprite 1 Position | Sprite 2 Position |
|------|-------------------|-------------------|
| NW | Between corners 0 and 3 | At corner 3 |
| NE | Between corners 0 and 1 | At corner 1 |
| SE | Between corners 1 and 2 | At corner 2 |
| SW | Between corners 2 and 3 | At corner 3 |

## Integration Plan

### 1. Load transition sprites in the world package

Add `Cstline1` and `Cstline2` sprite sheet loading alongside existing terrain sprites in `spritesheet.go`. Load `Cstline_map.json` as the pairing metadata.

### 2. Add transition sprites to Tile struct

```go
type Tile struct {
    sprites           []*ebiten.Image
    transitionSprites []*PositionedSprite  // new: transition overlays
    positionedSprites []*PositionedSprite
    roadSprites       []*ebiten.Image
    // ...
}
```

Transition sprites render **after** base terrain but **before** foliage/cities/roads, so transitions sit on the ground plane.

### 3. Compute transitions at world generation time

During `GenerateLevel()`, after all tiles have their terrain types assigned:

```
for each tile in grid:
    transitions := GetTransitions(pos, tileMap)
    for each transition:
        pair := findSmartTransitionPair(transition, metadata)
        tile.transitionSprites = append(tile.transitionSprites, pair...)
```

This is a one-time cost at generation, not per-frame.

### 4. Render transitions in Tile.Draw()

Update `Tile.Draw()` to draw `transitionSprites` between base sprites and positioned sprites:

```
1. Draw base terrain sprites
2. Draw transition sprites (new)
3. Draw road sprites
4. Draw positioned sprites (foliage, cities)
5. Draw encounter sprites
```

### 5. Port sprite pair selection from prototype

Move `findSmartTransitionPair()` and supporting logic from `cmd/tile_transitions/main.go` into `game/world/`. This includes:

- Sprite metadata parsing (the JSON)
- Pair scoring and selection
- Screen position calculation for the two sprites per edge

## Terrain Priority Order

Transitions are drawn from higher-priority terrain onto lower. Current ordering by noise threshold:

| Priority | Terrain | Noise Threshold |
|----------|---------|----------------|
| 0 (lowest) | Water | < 0x35 |
| 1 | Sand | 0x42-0x43 |
| 2 | Marsh | 0x43-0x45 |
| 3 | Plains | < 0x60 |
| 4 | Forest | < 0x75 |
| 5 | Mountains | < 0x95 |
| 6 (highest) | Snow | >= 0x95 |

## Sprite Asset Pipeline

If new terrain transitions are needed or sprites are updated:

1. Edit sprite sheets in an image editor
2. Run `python3 cmd/tile_transitions/analyze_edges.py` to regenerate `Cstline_map.json`
3. Validate with `python3 cmd/tile_transitions/visualize_connections.py` and `visualize_pairs.py`
4. Test with `go run ./cmd/tile_transitions` to render sample grids
5. Inspect output PNGs for visual correctness

## Implementation Order

1. Move sprite pair selection logic from `cmd/tile_transitions/` into `game/world/`
2. Load `Cstline1`, `Cstline2`, and `Cstline_map.json` in the world package
3. Add `transitionSprites` field to `Tile`
4. Compute transitions during `GenerateLevel()`
5. Render transitions in `Tile.Draw()`
6. Test with the full game map and verify visual quality
7. Remove redundant code from `cmd/tile_transitions/` that was moved into the game
