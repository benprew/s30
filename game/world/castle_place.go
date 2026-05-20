package world

import (
	"fmt"
	"image"
	"math/rand"
	"slices"

	"github.com/benprew/s30/game/domain"
	"github.com/hajimehoshi/ebiten/v2"
)

// castleRogues maps each MTG color to the level-11 wizard rogue who lives in
// that color's castle. The names must match keys in domain.Rogues.
var castleRogues = map[domain.ColorMask]string{
	domain.ColorWhite: "Sainted One",
	domain.ColorBlue:  "Astral Visionary",
	domain.ColorBlack: "Azaar - Lichlord",
	domain.ColorRed:   "Kzzy'n - The Dragon Lord",
	domain.ColorGreen: "Great Druid",
}

// castleSpec records the (row, col) of each of the four sprites a colored
// castle needs across its sheet: intact castle + shadow, and the destroyed
// pair. Most colors put intact/destroyed on neighboring columns and the
// shadow two rows below the castle, but Black on Castles1 uses a different
// layout (castle/shadow side-by-side; intact/destroyed on neighboring rows),
// so we encode each sprite explicitly rather than deriving it.
type castleSpec struct {
	sheet           int // 1 → Castles1, 2 → Castles2
	intactCastle    [2]int
	intactShadow    [2]int
	destroyedCastle [2]int
	destroyedShadow [2]int
}

var castleSpecs = map[domain.ColorMask]castleSpec{
	domain.ColorWhite: {
		sheet:           1,
		intactCastle:    [2]int{0, 0},
		intactShadow:    [2]int{2, 0},
		destroyedCastle: [2]int{0, 1},
		destroyedShadow: [2]int{2, 1},
	},
	domain.ColorBlue: {
		sheet:           1,
		intactCastle:    [2]int{1, 0},
		intactShadow:    [2]int{3, 0},
		destroyedCastle: [2]int{1, 1},
		destroyedShadow: [2]int{3, 1},
	},
	domain.ColorBlack: {
		sheet:           1,
		intactCastle:    [2]int{4, 0},
		intactShadow:    [2]int{4, 1},
		destroyedCastle: [2]int{5, 0},
		destroyedShadow: [2]int{5, 1},
	},
	domain.ColorGreen: {
		sheet:           2,
		intactCastle:    [2]int{0, 0},
		intactShadow:    [2]int{2, 0},
		destroyedCastle: [2]int{0, 1},
		destroyedShadow: [2]int{2, 1},
	},
	domain.ColorRed: {
		sheet:           2,
		intactCastle:    [2]int{1, 0},
		intactShadow:    [2]int{3, 0},
		destroyedCastle: [2]int{1, 1},
		destroyedShadow: [2]int{3, 1},
	},
}

// castleZoneTerrain maps each color to the terrain type that gets stamped
// around its castle.
var castleZoneTerrain = map[domain.ColorMask]int{
	domain.ColorWhite: TerrainPlains,
	domain.ColorBlue:  TerrainSand,
	domain.ColorBlack: TerrainMarsh,
	domain.ColorRed:   TerrainMountains,
	domain.ColorGreen: TerrainForest,
}

const (
	castleZoneRadius      = 5
	castleMinDist         = 12
	castlePlayerSafeRange = 10
)

// castleNames pairs nicely with the wizard's color so each castle has a
// readable label. Order must match domain.GetAllAmuletColors() (W,U,B,R,G).
var castleNames = []string{
	"Citadel of Order",
	"Tower of Knowledge",
	"Bastion of Power",
	"Keep of Passion",
	"Hold of Life",
}

// placeCastles picks one tile per MTG color, stamps its zone terrain, attaches
// a Castle to the tile, and adds the castle/shadow sprites. Sheets passed in
// are loaded as 2 cols × 6 rows of 309×309 cells.
func (l *Level) placeCastles(seed int64, castles1, castles2 [][]*ebiten.Image,
	ss *SpriteSheet, foliage, Sfoliage [][]*ebiten.Image) {
	candidates := l.castleCandidateTiles()
	if len(candidates) == 0 {
		fmt.Println("Warning: no candidate tiles for castle placement.")
		return
	}

	rng := rand.New(rand.NewSource(seed))
	rng.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})

	colors := domain.GetAllAmuletColors()
	placed := []image.Point{}

	for idx, color := range colors {
		loc, ok := pickCastleLocation(candidates, placed, []int{castleMinDist, 9, 6, 0})
		if !ok {
			fmt.Printf("Warning: no location available for %s castle.\n", domain.ColorMaskToString(color))
			continue
		}

		terrain := castleZoneTerrain[color]
		l.stampZone(loc, terrain, castleZoneRadius, ss, foliage, Sfoliage, rng)

		castle := &domain.Castle{
			Name:      castleNames[idx%len(castleNames)],
			Color:     color,
			RogueName: castleRogues[color],
			MapTile:   loc,
		}
		tile := l.Tile(loc)
		tile.IsCastle = true
		tile.Castle = castle

		spec := castleSpecs[color]
		sheet := castles1
		if spec.sheet == 2 {
			sheet = castles2
		}
		if sheet != nil {
			addCastleSprites(tile, sheet, spec, false)
		}

		l.Castles = append(l.Castles, castle)
		placed = append(placed, loc)
	}
}

// pickCastleLocation walks the candidate list and returns the first tile that
// is at least minDist tiles away from every already-placed castle. minDists is
// tried in order; if no candidate satisfies the strictest distance, the next
// (looser) value is tried.
func pickCastleLocation(candidates, placed []image.Point, minDists []int) (image.Point, bool) {
	for _, d := range minDists {
		for _, c := range candidates {
			if slices.Contains(placed, c) {
				continue
			}
			if d == 0 || farFrom(c, placed, d) {
				return c, true
			}
		}
	}
	return image.Point{}, false
}

// addCastleSprites stamps the shadow first (lower z-order), then the castle on
// top. The exact (row, col) for each sprite is taken from the spec so each
// color's quirky layout is honored.
func addCastleSprites(tile *Tile, sheet [][]*ebiten.Image, spec castleSpec, destroyed bool) {
	castleRC := spec.intactCastle
	shadowRC := spec.intactShadow
	if destroyed {
		castleRC = spec.destroyedCastle
		shadowRC = spec.destroyedShadow
	}
	addSpriteAt(tile, sheet, shadowRC)
	addSpriteAt(tile, sheet, castleRC)
}

func addSpriteAt(tile *Tile, sheet [][]*ebiten.Image, rc [2]int) {
	r, c := rc[0], rc[1]
	if r < 0 || r >= len(sheet) {
		return
	}
	if c < 0 || c >= len(sheet[r]) {
		return
	}
	tile.AddCitySprite(sheet[r][c])
}

// stampZone overrides terrain in a square around center, replacing each tile's
// base sprite and foliage with the target terrain. Water tiles and tiles
// adjacent to water are skipped so coastlines stay visually consistent. When
// ss is nil only the TerrainType is updated (used by tests that don't load
// sprite assets).
func (l *Level) stampZone(center image.Point, terrain, radius int,
	ss *SpriteSheet, foliage, Sfoliage [][]*ebiten.Image, rng *rand.Rand) {
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			p := image.Point{X: center.X + dx, Y: center.Y + dy}
			if p.X < 0 || p.Y < 0 || p.X >= l.W || p.Y >= l.H {
				continue
			}
			t := l.Tile(p)
			if t == nil || t.TerrainType == TerrainWater {
				continue
			}
			if neighborIsWater(l, p) {
				continue
			}
			if ss == nil {
				t.TerrainType = terrain
				continue
			}
			t.sprites = nil
			t.positionedSprites = nil
			folRow := rng.Intn(11)
			paintTerrain(t, terrain, ss, foliage, Sfoliage, folRow)
		}
	}
}

func neighborIsWater(l *Level, p image.Point) bool {
	for _, n := range Directions[p.Y%2] {
		np := image.Point{X: p.X + n.X, Y: p.Y + n.Y}
		if np.X < 0 || np.Y < 0 || np.X >= l.W || np.Y >= l.H {
			continue
		}
		t := l.Tile(np)
		if t != nil && t.TerrainType == TerrainWater {
			return true
		}
	}
	return false
}

// castleCandidateTiles returns tiles eligible to anchor a wizard's castle:
// non-water, not city, not road, away from the player's spawn at map center.
func (l *Level) castleCandidateTiles() []image.Point {
	cx, cy := l.W/2, l.H/2
	var out []image.Point
	for y := 0; y < l.H; y++ {
		for x := 0; x < l.W; x++ {
			t := l.Tiles[y][x]
			if t == nil {
				continue
			}
			if t.IsCity || t.IsRoad() || t.IsCastle {
				continue
			}
			if t.TerrainType == TerrainWater {
				continue
			}
			if absInt(x-cx)+absInt(y-cy) <= castlePlayerSafeRange {
				continue
			}
			out = append(out, image.Point{X: x, Y: y})
		}
	}
	return out
}

// castleTileLocations returns the tile coordinates of every placed castle so
// city/dungeon placement can keep their distance.
func (l *Level) castleTileLocations() []image.Point {
	out := make([]image.Point, 0, len(l.Castles))
	for _, c := range l.Castles {
		out = append(out, c.MapTile)
	}
	return out
}
