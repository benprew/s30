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

// CastleWizard returns the configured wizard for a colored castle.
func CastleWizard(color domain.ColorMask) (*domain.Character, bool) {
	name, ok := castleRogues[color]
	if !ok {
		return nil, false
	}
	wizard, ok := domain.Rogues[name]
	return wizard, ok
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

// placeCastles picks one biome-compatible tile for each MTG color. If the map
// does not contain the preferred terrain, it uses another land tile.
func (l *Level) placeCastles(seed int64, castles1, castles2 [][]*ebiten.Image,
	_ *SpriteSheet, _, _ [][]*ebiten.Image) {
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
		terrain := castleZoneTerrain[color]
		preferred := castleCandidatesForTerrain(l, candidates, terrain)
		loc, ok := pickCastleLocation(preferred, placed, []int{castleMinDist, 9, 6})
		if !ok {
			loc, ok = pickCastleLocation(candidates, placed, []int{castleMinDist, 9, 6, 0})
		}
		if !ok {
			fmt.Printf("Warning: no location available for %s castle.\n", domain.ColorMaskToString(color))
			continue
		}

		castle := &domain.Castle{
			Name:      castleNames[idx%len(castleNames)],
			Color:     color,
			RogueName: castleRogues[color],
			MapTile:   loc,
		}
		tile := l.Tile(loc)
		tile.suppressTerrainDecorations()
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

func castleCandidatesForTerrain(l *Level, candidates []image.Point, terrain int) []image.Point {
	matching := make([]image.Point, 0, len(candidates))
	for _, candidate := range candidates {
		if tile := l.Tile(candidate); tile != nil && tile.TerrainType == terrain {
			matching = append(matching, candidate)
		}
	}
	return matching
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
			if t.IsCity() || t.IsRoad() || t.IsCastle {
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
