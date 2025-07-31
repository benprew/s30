package world

import (
	"fmt"
	"image"
	"math"
	"math/rand"
	"time"

	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/entities"
	"github.com/benprew/s30/game/sprites"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Level represents a Game level.
type Level struct {
	w, h       int
	Tiles      [][]*Tile // (Y,X) array of tiles
	tileWidth  int
	tileHeight int

	roadSprites    [][]*ebiten.Image // Sprites for roads
	roadSpriteInfo [][]string        // Maps sprite index to direction string (e.g., "N", "NE")

	Player  *entities.Player
	enemies []entities.Enemy
	Frame   *ebiten.Image
}

// NewLevel returns a new randomly generated Level.
func NewLevel() (*Level, error) {
	startTime := time.Now()
	fmt.Println("NewLevel start")

	c, err := entities.NewPlayer(entities.EgoFemale)
	if err != nil {
		return nil, fmt.Errorf("failed to load player sprite: %s", err)
	}

	frame, err := LoadWorldFrame()
	if err != nil {
		return nil, fmt.Errorf("failed to load world frame: %s", err)
	}

	l := &Level{
		w:          47,
		h:          63,
		tileWidth:  206,
		tileHeight: 102,
		enemies:    make([]entities.Enemy, 0),
		Player:     c,
		Frame:      frame,
	}

	// Load embedded SpriteSheet.
	ss, err := LoadWorldTileSheet(l.tileWidth, l.tileHeight)
	if err != nil {
		return nil, fmt.Errorf("failed to load embedded spritesheet: %s", err)
	}

	// widths are the 5 terrain types:
	// marsh, desert, forest, mountain, plains
	// foliage is 206x134
	// land tile is 206x102
	foliage, err := sprites.LoadSpriteSheet(5, 11, assets.Land_png)
	if err != nil {
		return nil, fmt.Errorf("failed to load embedded spritesheet: %s", err)
	}
	// shadows for lands
	Sfoliage, err := sprites.LoadSpriteSheet(5, 11, assets.Sland_png)
	if err != nil {
		return nil, fmt.Errorf("failed to load embedded spritesheet: %s", err)
	}

	foliage2, err := sprites.LoadSpriteSheet(5, 11, assets.Land2_png)
	if err != nil {
		return nil, fmt.Errorf("failed to load embedded spritesheet: %s", err)
	}
	Sfoliage2, err := sprites.LoadSpriteSheet(5, 11, assets.Sland2_png)
	if err != nil {
		return nil, fmt.Errorf("failed to load embedded spritesheet: %s", err)
	}

	Cstline2, err := sprites.LoadSpriteSheet(4, 14, assets.Cstline2_png)
	if err != nil {
		return nil, fmt.Errorf("failed to load embedded spritesheet: %s", err)
	}

	citySprites, err := sprites.LoadSpriteSheet(6, 4, assets.Cities1_png)
	if err != nil {
		return nil, fmt.Errorf("failed to load city spritesheet Castles1.spr.png: %w", err)
	}

	roads, err := sprites.LoadSpriteSheet(6, 2, assets.Roads_png)
	if err != nil {
		return nil, fmt.Errorf("failed to load city spritesheet Roads_png: %w", err)
	}
	// Store roads and info in the level struct
	l.roadSprites = roads
	// Define the mapping from sprite sheet index to compass direction exit point.
	// Based on Roads.spr.png layout (6 columns, 2 rows)
	l.roadSpriteInfo = [][]string{
		{"", "NE", "E", "SE", "N", "SW"},
		{"W", "NW", "S", "", "", ""},
	}
	noise := generateTerrain(l.w, l.h)
	// mapTerrainTypes now returns valid city locations and sets Tile.TerrainType
	validCityLocations := l.mapTerrainTypes(noise, ss, foliage, Sfoliage, foliage2, Sfoliage2, Cstline2, citySprites)

	l.placeCities(validCityLocations, citySprites, 35, 6)

	// Set initial player position at center of map
	loc := image.Point{X: l.LevelW() / 2, Y: l.LevelH() / 2}
	l.Player.SetLoc(loc)
	fmt.Printf("Starting player at position: %d, %d\n", loc.X, loc.Y)

	// Spawn initial enemies
	if err := l.spawnEnemies(3); err != nil {
		return nil, fmt.Errorf("failed to spawn enemies: %s", err)
	}

	fmt.Printf("NewLevel execution time: %s\n", time.Since(startTime))
	return l, nil
}

func (l *Level) Draw(screen *ebiten.Image, screenW, screenH int, scale float64) {
	padding := 400

	pLoc := l.Player.Loc()

	l.RenderZigzag(screen, pLoc.X, pLoc.Y, (screenW/2)+padding, (screenH/2)+padding, scale)

	// Draw enemies
	for _, e := range l.enemies {
		eLoc := e.Loc()
		eDim := e.Dims()
		if !l.isVisible(eLoc.X, eLoc.Y, eDim.Width, eDim.Height, screenW, screenH) {
			continue // Skip if not visible
		}

		screenX, screenY := l.screenOffset(eLoc.X, eLoc.Y, screenW, screenH)
		// Draw enemy
		enemyOp := &ebiten.DrawImageOptions{}
		enemyOp.GeoM.Scale(scale, scale)
		enemyOp.GeoM.Translate(-float64(entities.SpriteWidth/2)*scale, -float64(entities.SpriteHeight/2)*scale)
		enemyOp.GeoM.Translate(float64(screenX), float64(screenY))
		e.Draw(screen, enemyOp)
	}

	// Draw player
	options := &ebiten.DrawImageOptions{}
	options.GeoM.Scale(scale, scale)
	options.GeoM.Translate(float64(screenW)/2, float64(screenH)/2)
	options.GeoM.Translate(-float64(entities.SpriteWidth/2)*scale, -float64(entities.SpriteHeight/2)*scale)
	l.Player.Draw(screen, options)

	// Draw the worldFrame over everything
	frameOpts := &ebiten.DrawImageOptions{}
	frameOpts.GeoM.Scale(scale, scale)
	screen.DrawImage(l.Frame, frameOpts)
}

// Update reads current user input and updates the Game state.
func (l *Level) Update(screenW, screenH int) (screenui.ScreenName, error) {
	// Store current player tile before moving
	prevTile := l.CharacterTile()

	// Move player and update direction via keyboard using bit flags
	if err := l.Player.Update(screenW, screenH, l.LevelW(), l.LevelH()); err != nil {
		return screenui.WorldScr, err
	}

	// Get player's new tile
	currentTile := l.CharacterTile()

	if currentTile != (TilePoint{-1, -1}) { // Ensure player is on a valid tile
		tile := l.Tile(currentTile)
		if tile != nil {
			if tile.IsCity && prevTile != currentTile {
				return screenui.CityScr, nil
			}
		}
	}

	// Update enemies to move towards character
	for _, e := range l.enemies {
		e.Update(l.Player.Loc())
	}

	// Add more enemies with the 'N' key
	if inpututil.IsKeyJustPressed(ebiten.KeyN) {
		if err := l.spawnEnemies(5); err != nil {
			return screenui.WorldScr, fmt.Errorf("failed to spawn additional enemies: %s", err)
		}
	}

	return screenui.WorldScr, nil
}

// x,y is the position of the tile, width and height are the dimensions of the tile
// Check if the tile is visible on the screen
// return the position of the tile in the screen
func (l *Level) isVisible(x, y, width, height, screenW, screenH int) bool {
	// convert screenW and screenH based on the player position
	pLoc := l.Player.Loc()
	screenX := pLoc.X - (screenW / 2)
	screenY := pLoc.Y - (screenH / 2)

	// Check if the object is within the screen bounds
	if x+width < screenX || x > screenX+screenW || y+height < screenY || y > screenY+screenH {
		return false
	}

	return true
}

func (l *Level) screenOffset(x, y, screenW, screenH int) (int, int) {
	pLoc := l.Player.Loc()
	// Calculate screen position based on player position
	screenX := pLoc.X - screenW/2
	screenY := pLoc.Y - screenH/2

	// Calculate the position relative to the screen
	return x - screenX, y - screenY
}

// spawnEnemies creates a specified number of enemies at random positions
func (l *Level) spawnEnemies(count int) error {
	// Enemy character types to choose from
	enemyTypes := entities.Enemies

	pLoc := l.Player.Loc()

	for i := 0; i < count; i++ {
		// Choose a random enemy type
		enemyType := enemyTypes[rand.Intn(len(enemyTypes))]

		// Load the enemy sprite
		enemy, err := entities.NewEnemy(enemyType)
		if err != nil {
			return fmt.Errorf("failed to create enemy %s: %w", enemyType, err)
		}
		var x, y int

		// Set random position (avoiding player's immediate area)
		minDistance := 5000.0 // Minimum distance from player
		for {
			// Random position within world bounds
			x = rand.Intn(l.LevelW())
			y = rand.Intn(l.LevelH())

			// Check distance from player
			dx := x - pLoc.X
			dy := y - pLoc.Y
			distance := math.Sqrt(float64(dx*dx + dy*dy))

			if distance < minDistance {
				break
			}
		}

		enemy.SetLoc(image.Point{X: x, Y: y})
		enemy.Speed(5 + rand.Intn(7))

		l.enemies = append(l.enemies, enemy)
	}

	return nil
}

func (l *Level) LevelW() int {
	return l.tileWidth * l.w
}

func (l *Level) LevelH() int {
	return l.tileHeight / 2 * l.h
}

// Tile returns the tile at the provided coordinates, or nil.
func (l *Level) Tile(p TilePoint) *Tile {
	if p.X >= 0 && p.Y >= 0 && p.X < l.w && p.Y < l.h {
		return l.Tiles[p.Y][p.X]
	}
	return nil
}

// Size returns the size of the Level.
func (l *Level) Size() (width, height int) {
	return l.w, l.h
}

func (l *Level) RenderZigzag(screen *ebiten.Image, pX, pY, padX, padY int, scale float64) {
	tileWidth := l.tileWidth
	tileHeight := l.tileHeight

	op := &ebiten.DrawImageOptions{}

	// the visible drawable area
	visibleXOrigin := pX - padX
	visibleYOrigin := pY - padY
	visibleXOpposite := pX + padX
	visibleYOpposite := pY + padY

	for y := 0; y < l.h; y++ {
		for x := 0; x < l.w; x++ {
			tile := l.Tile(TilePoint{x, y})
			if tile == nil {
				continue
			}

			// Calculate screen position
			pixelX := x * tileWidth
			pixelY := y * tileHeight / 2

			// Offset every other row to create the zigzag pattern
			if y%2 != 0 {
				pixelX += tileWidth / 2
			}

			if pixelX < visibleXOrigin || pixelX > visibleXOpposite {
				continue // Skip rendering if outside visible area
			}

			if pixelY < visibleYOrigin || pixelY > visibleYOpposite {
				continue // Skip rendering if outside visible area
			}
			screenX := pixelX - (pX - 1024/2)
			screenY := pixelY - (pY - 768/2)

			op.GeoM.Reset()
			op.GeoM.Scale(scale, scale)
			op.GeoM.Translate(float64(screenX), float64(screenY))
			tile.Draw(screen, op)
		}
	}
}

func (l *Level) CharacterPos() image.Point {
	return l.Player.Loc()
}

func (l *Level) CharacterTile() TilePoint {
	pLoc := l.Player.Loc()
	pixelX := pLoc.X
	pixelY := pLoc.Y

	tileWidth := l.tileWidth
	tileHeight := l.tileHeight

	// Calculate the approximate row and column
	tileY := pixelY / (tileHeight / 2)
	tileX := (pixelX - (tileY%2)*(tileWidth/2)) / tileWidth

	// Ensure the tile coordinates are within bounds
	if pixelX < 0 || tileX >= l.w || pixelY < 0 || tileY >= l.h {
		return TilePoint{-1, -1} // Return invalid coordinates if out of bounds
	}

	return TilePoint{tileX, tileY}
}

func PrintLevel(l *Level) {
	for i, row := range l.Tiles {
		for _, col := range row {
			t := "T"
			if col.IsRoad() {
				t = "R"
			}
			if col.IsCity {
				t = "C"
			}
			if col.TerrainType == TerrainWater {
				t = "W"
			}
			if i%2 == 1 {
				fmt.Print("-", t)
			} else {
				fmt.Print(t, "-")
			}
		}
		fmt.Println()
	}
}
