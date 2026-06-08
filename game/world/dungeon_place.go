package world

import (
	"image"
	"math/rand"

	"github.com/benprew/s30/game/domain"
	"github.com/hajimehoshi/ebiten/v2"
)

// dungeonColors is the cycle order used when assigning a color to each
// generated dungeon. We round-robin through the five MTG colors so a small
// number of dungeons still spans the color wheel.
var dungeonColors = []domain.ColorMask{
	domain.ColorWhite,
	domain.ColorBlue,
	domain.ColorBlack,
	domain.ColorRed,
	domain.ColorGreen,
}

// placeDungeons selects up to numDungeons land tiles, generates a Dungeon for
// each, and attaches it to both the world tile and the Level's Dungeons slice.
// Dungeon tiles are kept at minDistance from each other and from city tiles.
// dungeonSprites may be nil (e.g. in tests) — when supplied, a sprite is added
// to each placed tile so the entrance is visible on the world map.
func (l *Level) placeDungeons(numDungeons, minDistance int, seed int64, dungeonSprites [][]*ebiten.Image) {
	if numDungeons <= 0 {
		return
	}

	candidates := l.dungeonCandidateTiles()
	if len(candidates) == 0 {
		return
	}

	rng := rand.New(rand.NewSource(seed))
	rng.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})

	cityLocs := l.cityTileLocations()
	castleLocs := l.castleTileLocations()
	placed := []image.Point{}

	// Dice grant a card from the player's own deck (lands excluded), so the
	// pool is shared across every dungeon placed in this world.
	var diceCardPool []*domain.Card
	if l.Player != nil {
		diceCardPool = l.Player.GetDuelDeck().NonLandCards()
	}

	for _, loc := range candidates {
		if len(placed) >= numDungeons {
			break
		}
		if !farFrom(loc, cityLocs, minDistance) || !farFrom(loc, placed, minDistance) {
			continue
		}
		if !farFrom(loc, castleLocs, minDistance) {
			continue
		}

		idx := len(placed)
		color := dungeonColors[idx%len(dungeonColors)]
		dungeon := domain.GenerateDungeon(domain.DungeonGenOptions{
			Name:          dungeonName(idx),
			Level:         1,
			Color:         color,
			CreatureSize:  domain.CreatureSizeSmall,
			GridSize:      11,
			NumEnemies:    3,
			NumDice:       2,
			NumScrolls:    1,
			NumGoldChests: 2,
			EnemyPool:     domain.DungeonEnemyPool(color),
			DiceCardPool:  diceCardPool,
			Seed:          seed + int64(idx),
		})
		dungeon.MapTile = loc

		tile := l.Tile(loc)
		tile.IsDungeon = true
		tile.Dungeon = dungeon
		if dungeonSprites != nil {
			addDungeonSprites(tile, dungeonSprites, idx)
		}

		l.Dungeons = append(l.Dungeons, dungeon)
		placed = append(placed, loc)
	}
}

// addDungeonSprites adds a dungeon entrance sprite (and its shadow) to the
// tile, mirroring the city sprite layout. The sprite sheet is 6 columns × 4
// rows: rows 0-1 hold dungeon variants, rows 2-3 are the matching shadows.
func addDungeonSprites(tile *Tile, sheet [][]*ebiten.Image, idx int) {
	col := idx % 6
	tile.AddCitySprite(sheet[2][col])
	tile.AddCitySprite(sheet[0][col])
}

func (l *Level) dungeonCandidateTiles() []image.Point {
	var out []image.Point
	for y := 0; y < l.H; y++ {
		for x := 0; x < l.W; x++ {
			t := l.Tiles[y][x]
			if t == nil {
				continue
			}
			if t.IsCity || t.IsRoad() {
				continue
			}
			switch t.TerrainType {
			case TerrainPlains, TerrainForest, TerrainMountains:
				out = append(out, image.Point{X: x, Y: y})
			}
		}
	}
	return out
}

func (l *Level) cityTileLocations() []image.Point {
	var out []image.Point
	for y := 0; y < l.H; y++ {
		for x := 0; x < l.W; x++ {
			if l.Tiles[y][x] != nil && l.Tiles[y][x].IsCity {
				out = append(out, image.Point{X: x, Y: y})
			}
		}
	}
	return out
}

func farFrom(p image.Point, others []image.Point, minDistance int) bool {
	for _, o := range others {
		if absInt(p.X-o.X)+absInt(p.Y-o.Y) <= minDistance {
			return false
		}
	}
	return true
}

func dungeonName(idx int) string {
	names := []string{
		"Whispering Crypt",
		"Blackthorn Hold",
		"Sunken Vault",
		"Ember Spire",
		"Frostbite Warren",
		"Hollow Reliquary",
		"Gloom Bastion",
		"Verdant Sanctum",
	}
	return names[idx%len(names)]
}
