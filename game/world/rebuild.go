package world

import (
	"fmt"
	"image"

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
	cstline1    [][]*ebiten.Image
	cstline2    [][]*ebiten.Image
	dungeons    [][]*ebiten.Image
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
	cstline1, err := imageutil.LoadSpriteSheet(4, 21, assets.Cstline_png)
	if err != nil {
		return nil, fmt.Errorf("failed to load Cstline1: %w", err)
	}
	cstline2, err := imageutil.LoadSpriteSheet(4, 14, assets.Cstline2_png)
	if err != nil {
		return nil, fmt.Errorf("failed to load Cstline2: %w", err)
	}
	dungeons, err := imageutil.LoadSpriteSheet(6, 4, assets.Dungeons_png)
	if err != nil {
		return nil, fmt.Errorf("failed to load dungeon sprites: %w", err)
	}
	return &spriteAssets{
		ss: ss, foliage: foliage, sfoliage: sfoliage,
		foliage2: foliage2, sfoliage2: sfoliage2,
		citySprites: citySprites, castles1: castles1, castles2: castles2,
		cstline1: cstline1, cstline2: cstline2, dungeons: dungeons,
	}, nil
}

func rebuildTileSprites(tile *Tile, position image.Point, seed int64, sa *spriteAssets) {
	tile.sprites = nil
	tile.transitionSprites = nil
	tile.terrainSprites = nil
	tile.positionedSprites = nil
	tile.roadSprites = nil
	tile.encounterSprites = nil

	addTerrainBase(tile, sa.ss)
	if !tile.IsCity() && !tile.IsCastle && !tile.IsDungeon {
		addTerrainDecorations(tile, position, seed, sa)
	}

	if tile.IsCity() {
		cityIdx := deterministicIndex(seed, tile.City.X, tile.City.Y, 12)
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

	if tile.IsDungeon && tile.Dungeon != nil {
		addDungeonSprites(tile, sa.dungeons, deterministicIndex(seed, position.X, position.Y, 6))
	}
}

func addTerrainBase(tile *Tile, ss *SpriteSheet) {
	switch tile.TerrainBand.properties().base {
	case terrainBaseWater:
		tile.AddSprite(ss.Water)
	case terrainBaseSand:
		tile.AddSprite(ss.Sand)
	case terrainBaseMarsh:
		tile.AddSprite(ss.Marsh)
	case terrainBaseIce:
		tile.AddSprite(ss.Ice)
	default:
		tile.AddSprite(ss.Plains)
	}
}

func addTerrainDecorations(tile *Tile, position image.Point, seed int64, sa *spriteAssets) {
	for _, decoration := range terrainDecorationPlan(seed, position, tile.TerrainBand, 206) {
		foliage := sa.foliage
		shadows := sa.sfoliage
		if decoration.Secondary {
			foliage = sa.foliage2
			shadows = sa.sfoliage2
		}
		if decoration.Variant >= len(foliage) || decoration.Variant >= len(shadows) ||
			decoration.Column >= len(foliage[decoration.Variant]) || decoration.Column >= len(shadows[decoration.Variant]) {
			continue
		}
		tile.AddTerrainSprite(shadows[decoration.Variant][decoration.Column], decoration.OffsetX, decoration.OffsetY)
		tile.AddTerrainSprite(foliage[decoration.Variant][decoration.Column], decoration.OffsetX, decoration.OffsetY)
	}
}

func deterministicIndex(seed int64, x, y, size int) int {
	if size <= 0 {
		return 0
	}
	value := uint64(seed) ^ uint64(uint32(x))*0x9e3779b1 ^ uint64(uint32(y))*0x85ebca77
	value ^= value >> 30
	value *= 0xbf58476d1ce4e5b9
	value ^= value >> 27
	return int(value % uint64(size))
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
				rebuildTileSprites(tile, image.Point{X: x, Y: y}, l.GenerationSeed, sa)
			}
		}
	}

	l.rebuildRoads()
	if err := l.rebuildTerrainTransitions(sa); err != nil {
		return err
	}

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
