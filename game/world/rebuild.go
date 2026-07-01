package world

import (
	"fmt"
	"image"
	"math/rand"

	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/ui/imageutil"
	"github.com/hajimehoshi/ebiten/v2"
)

type spriteAssets struct {
	ss          *SpriteSheet
	foliage     [][]*ebiten.Image
	sfoliage    [][]*ebiten.Image
	foliage2    [][]*ebiten.Image
	sfoliage2   [][]*ebiten.Image
	citySprites [][]*ebiten.Image
	castles1    [][]*ebiten.Image
	castles2    [][]*ebiten.Image
}

func loadSpriteAssets(tileWidth, tileHeight int) (*spriteAssets, error) {
	ss, err := LoadWorldTileSheet(tileWidth, tileHeight)
	if err != nil {
		return nil, fmt.Errorf("failed to load world tile sheet: %w", err)
	}
	foliage, err := imageutil.LoadSpriteSheet(5, 11, assets.Land_png)
	if err != nil {
		return nil, fmt.Errorf("failed to load foliage: %w", err)
	}
	sfoliage, err := imageutil.LoadSpriteSheet(5, 11, assets.Sland_png)
	if err != nil {
		return nil, fmt.Errorf("failed to load foliage shadows: %w", err)
	}
	foliage2, err := imageutil.LoadSpriteSheet(5, 11, assets.Land2_png)
	if err != nil {
		return nil, fmt.Errorf("failed to load foliage2: %w", err)
	}
	sfoliage2, err := imageutil.LoadSpriteSheet(5, 11, assets.Sland2_png)
	if err != nil {
		return nil, fmt.Errorf("failed to load foliage2 shadows: %w", err)
	}
	citySprites, err := imageutil.LoadSpriteSheet(6, 4, assets.Cities1_png)
	if err != nil {
		return nil, fmt.Errorf("failed to load city sprites: %w", err)
	}
	castles1, err := imageutil.LoadSpriteSheet(2, 6, assets.Castles1_png)
	if err != nil {
		return nil, fmt.Errorf("failed to load Castles1 sprites: %w", err)
	}
	castles2, err := imageutil.LoadSpriteSheet(2, 6, assets.Castles2_png)
	if err != nil {
		return nil, fmt.Errorf("failed to load Castles2 sprites: %w", err)
	}
	return &spriteAssets{ss, foliage, sfoliage, foliage2, sfoliage2, citySprites, castles1, castles2}, nil
}

func rebuildTileSprites(tile *Tile, sa *spriteAssets) {
	tile.sprites = nil
	tile.positionedSprites = nil
	tile.roadSprites = nil
	tile.encounterSprites = nil

	folIdx := rand.Intn(11)

	switch tile.TerrainType {
	case TerrainWater:
		tile.AddSprite(sa.ss.Water)
		if rand.Float64() < 0.1 {
			tile.AddFoliageSprite(sa.sfoliage2[folIdx][0])
			tile.AddFoliageSprite(sa.foliage2[folIdx][0])
		}
	case TerrainSand:
		tile.AddSprite(sa.ss.Sand)
		tile.AddFoliageSprite(sa.sfoliage[folIdx][1])
		tile.AddFoliageSprite(sa.foliage[folIdx][1])
	case TerrainMarsh:
		tile.AddSprite(sa.ss.Marsh)
		tile.AddFoliageSprite(sa.sfoliage[folIdx][0])
		tile.AddFoliageSprite(sa.foliage[folIdx][0])
	case TerrainPlains:
		tile.AddSprite(sa.ss.Plains)
		tile.AddFoliageSprite(sa.sfoliage[folIdx][4])
		tile.AddFoliageSprite(sa.foliage[folIdx][4])
	case TerrainForest:
		tile.AddSprite(sa.ss.Forest)
		tile.AddFoliageSprite(sa.sfoliage[folIdx][2])
		tile.AddFoliageSprite(sa.foliage[folIdx][2])
	case TerrainMountains, TerrainSnow:
		tile.AddSprite(sa.ss.Plains)
		tile.AddFoliageSprite(sa.sfoliage[folIdx][3])
		tile.AddFoliageSprite(sa.foliage[folIdx][3])
	}

	if tile.IsCity() {
		cityIdx := rand.Intn(12)
		cityX := cityIdx % 6
		cityY := 0
		if cityIdx > 5 {
			cityY = 2
		}
		tile.AddCitySprite(sa.citySprites[cityY][cityX])
		tile.AddCitySprite(sa.citySprites[cityY+1][cityX])
		tile.City.BackgroundImage = cityBgImage(int(tile.City.Tier))
	}

	if tile.IsCastle && tile.Castle != nil {
		if spec, ok := castleSpecs[tile.Castle.Color]; ok {
			sheet := sa.castles1
			if spec.sheet == 2 {
				sheet = sa.castles2
			}
			addCastleSprites(tile, sheet, spec, tile.Castle.Defeated)
		}
	}
}

// RebuildSprites reloads all image data after deserializing a Level from JSON.
// Sprite pointers don't survive JSON round-trips, so this rebuilds terrain,
// foliage, city, road, and encounter sprites from the saved TerrainType and
// City data on each tile.
func (l *Level) RebuildSprites() error {
	if l.TileWidth == 0 {
		l.TileWidth = 206
	}
	if l.TileHeight == 0 {
		l.TileHeight = 102
	}

	sa, err := loadSpriteAssets(l.TileWidth, l.TileHeight)
	if err != nil {
		return err
	}
	l.castles1Sprites = sa.castles1
	l.castles2Sprites = sa.castles2

	roads, err := imageutil.LoadSpriteSheet(6, 2, assets.Roads_png)
	if err != nil {
		return fmt.Errorf("failed to load road sprites: %w", err)
	}
	l.roadSprites = roads
	l.roadSpriteInfo = [][]string{
		{"", "NE", "E", "SE", "N", "SW"},
		{"W", "NW", "S", "", "", ""},
	}

	// JSON unmarshalling produces a separate *Castle for each tile and for
	// each entry in l.Castles even when the original objects were aliased.
	// Re-link so updates to a castle's Defeated flag are visible from both
	// sides without any divergence.
	for _, c := range l.Castles {
		if c == nil {
			continue
		}
		if tile := l.Tile(c.MapTile); tile != nil && tile.IsCastle {
			tile.Castle = c
		}
	}

	for y := 0; y < l.H; y++ {
		for x := 0; x < l.W; x++ {
			if tile := l.Tiles[y][x]; tile != nil {
				rebuildTileSprites(tile, sa)
			}
		}
	}

	l.rebuildRoads()

	if err := l.rebuildEncounters(); err != nil {
		return err
	}

	if err := l.Player.LoadImages(); err != nil {
		return fmt.Errorf("failed to load player sprites: %w", err)
	}

	for i := range l.Enemies {
		name := l.Enemies[i].Character.Name
		if rogue, ok := domain.Rogues[name]; ok {
			l.Enemies[i].Character = rogue
		}
		if err := l.Enemies[i].Character.LoadImages(); err != nil {
			fmt.Printf("Warning: failed to load sprites for enemy %q: %v\n", name, err)
		}
	}

	l.rebuildDungeonEnemies()

	return nil
}

// rebuildDungeonEnemies re-links each dungeon enemy tile to the shared rogue
// Character from the registry. JSON unmarshalling produces a detached copy per
// tile; re-linking restores the canonical character (with its deck) and lets
// the sprites load lazily when the dungeon is entered.
func (l *Level) rebuildDungeonEnemies() {
	for _, d := range l.Dungeons {
		if d == nil {
			continue
		}
		for y := range d.Grid {
			for x := range d.Grid[y] {
				tile := &d.Grid[y][x]
				if tile.Type != domain.DungeonTileEnemy || tile.Enemy == nil {
					continue
				}
				if rogue, ok := domain.Rogues[tile.Enemy.Name]; ok {
					tile.Enemy = rogue
				}
			}
		}
	}
}

func (l *Level) rebuildRoads() {
	var cityLocations []image.Point
	for y := 0; y < l.H; y++ {
		for x := 0; x < l.W; x++ {
			if tile := l.Tiles[y][x]; tile != nil && tile.IsCity() {
				cityLocations = append(cityLocations, image.Point{x, y})
			}
		}
	}
	for i, loc := range cityLocations {
		if i == 0 {
			continue
		}
		if path := l.connectCityBFS(loc); path != nil {
			l.drawRoadAlongPath(path)
		}
	}
}

func (l *Level) rebuildEncounters() error {
	if err := l.LoadRandomEncounterSprites(); err != nil {
		return fmt.Errorf("failed to load encounter sprites: %w", err)
	}
	for _, re := range l.RandomEncounters {
		if tile := l.Tile(re.Tile); tile != nil {
			tile.AddRandomEncounter(
				l.encounterSprites[1][re.SpriteIndex],
				l.encounterSprites[0][re.SpriteIndex],
			)
		}
	}
	return nil
}
