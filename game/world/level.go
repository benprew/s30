package world

import (
	"fmt"
	"image"
	"math"
	"math/rand"
	"time"

	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/sprites"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// TODO this file should be separated into screen and world logic

// Level represents a Game level.
type Level struct {
	w, h       int
	Tiles      [][]*Tile // (Y,X) array of tiles
	tileWidth  int
	tileHeight int

	roadSprites    [][]*ebiten.Image // Sprites for roads
	roadSpriteInfo [][]string        // Maps sprite index to direction string (e.g., "N", "NE")

	Player  *domain.Player
	enemies []domain.Enemy
	// encounterIndex holds the index of an enemy that triggered an encounter
	// encounterPending indicates whether an encounter is waiting to be consumed
	encounterIndex   int
	encounterPending bool
}

// NewLevel returns a new randomly generated Level.
func NewLevel(c *domain.Player) (*Level, error) {
	startTime := time.Now()
	fmt.Println("NewLevel start")

	l := &Level{
		w:              47,
		h:              63,
		tileWidth:      206,
		tileHeight:     102,
		enemies:        make([]domain.Enemy, 0),
		Player:         c,
		encounterIndex: -1,
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

func (l *Level) IsFramed() bool {
	return true
}

func (l *Level) Draw(screen *ebiten.Image, screenW, screenH int, scale float64) {
	padding := 400

	pLoc := l.Player.Loc()

	l.RenderZigzag(screen, pLoc.X, pLoc.Y, (screenW/2)+padding, (screenH/2)+padding, scale)

	// Draw enemies
	for _, e := range l.enemies {
		eLoc := e.Loc()
		eDim := e.Dims()
		if !l.isVisible(eLoc.X, eLoc.Y, eDim.Dx(), eDim.Dy(), screenW, screenH) {
			continue // Skip if not visible
		}

		screenX, screenY := l.screenOffset(eLoc.X, eLoc.Y, screenW, screenH)
		// Draw enemy
		enemyOp := &ebiten.DrawImageOptions{}
		enemyOp.GeoM.Scale(scale, scale)
		enemyOp.GeoM.Translate(-float64(domain.SpriteWidth/2)*scale, -float64(domain.SpriteHeight/2)*scale)
		enemyOp.GeoM.Translate(float64(screenX), float64(screenY))
		e.Draw(screen, enemyOp)
	}

	// Draw player
	options := &ebiten.DrawImageOptions{}
	options.GeoM.Scale(scale, scale)
	options.GeoM.Translate(float64(screenW)/2, float64(screenH)/2)
	options.GeoM.Translate(-float64(domain.SpriteWidth/2)*scale, -float64(domain.SpriteHeight/2)*scale)
	l.Player.Draw(screen, options)
}

// Update reads current user input and updates the Game state.
func (l *Level) Update(screenW, screenH int, scale float64) (screenui.ScreenName, error) {
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

	// Update enemies to move towards character (operate on slice elements so updates persist)
	for i := range l.enemies {
		// update the enemy in place
		_ = l.enemies[i].Update(l.Player.Loc())
	}

	// Check for proximity between player and any enemy (center-to-center)
	pLoc := l.Player.Loc()
	for i, e := range l.enemies {
		// skip already engaged enemies
		if e.Engaged {
			continue
		}
		eLoc := e.Loc()

		// compute center points
		pCenterX := float64(pLoc.X)
		pCenterY := float64(pLoc.Y)
		eCenterX := float64(eLoc.X)
		eCenterY := float64(eLoc.Y)

		dx := pCenterX - eCenterX
		dy := pCenterY - eCenterY
		dist := math.Hypot(dx, dy)

		if dist <= 20.0 {
			// record which enemy triggered encounter and mark pending
			l.encounterIndex = i
			l.encounterPending = true
			return screenui.DuelAnteScr, nil
		}
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
	enemyTypes := domain.Rogues

	// Get the keys (rogue names) from the map
	var rogueNames []string
	for name := range enemyTypes {
		rogueNames = append(rogueNames, name)
	}

	pLoc := l.Player.Loc()

	for i := 0; i < count; i++ {
		var enemy domain.Enemy
		var err error

		// Try to find an enemy with a valid walking sprite
		maxAttempts := len(rogueNames)
		for attempt := 0; attempt < maxAttempts; attempt++ {
			// Choose a random enemy type
			enemyType := rogueNames[rand.Intn(len(rogueNames))]

			// Load the enemy sprite
			enemy, err = domain.NewEnemy(enemyType)
			if err != nil {
				continue // Try another enemy type
			}

			// Check if the enemy has a valid walking sprite
			if enemy.Character.WalkingSprite != nil {
				break // Found a valid enemy
			}
		}

		// If we couldn't find any enemy with a walking sprite after trying all types
		if enemy.Character.WalkingSprite == nil {
			return fmt.Errorf("no enemies with valid walking sprites available")
		}
		var x, y int

		// Set random position (avoiding player's immediate area)
		minDistance := 500.0 // Minimum distance from player
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
		enemy.MoveSpeed = 5 + rand.Intn(7)

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

			// we don't scale the world view up
			op.GeoM.Reset()
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

	// Calculate the approximate row and column
	tileY := pixelY / (l.tileHeight / 2)
	tileX := (pixelX - (tileY%2)*(l.tileWidth/2)) / l.tileWidth

	// Ensure the tile coordinates are within bounds
	if pixelX < 0 || tileX >= l.w || pixelY < 0 || tileY >= l.h {
		return TilePoint{-1, -1} // Return invalid coordinates if out of bounds
	}

	return TilePoint{tileX, tileY}
}

// EncounterPending returns true if an encounter was recorded and not yet taken.
func (l *Level) EncounterPending() bool {
	return l.encounterPending && l.encounterIndex >= 0 && l.encounterIndex < len(l.enemies)
}

// TakeEncounter returns the enemy that triggered the encounter and clears the pending flag.
// The second return value is false if no pending encounter exists.
// TakeEncounter returns the enemy that triggered the encounter and its index, and clears the pending flag.
// The second return value is false if no pending encounter exists.
func (l *Level) TakeEncounter() (domain.Enemy, int, bool) {
	if !l.EncounterPending() {
		return domain.Enemy{}, -1, false
	}
	idx := l.encounterIndex
	e := l.enemies[idx]
	l.encounterPending = false
	l.encounterIndex = -1
	return e, idx, true
}

// RemoveEnemyAt removes an enemy by index.
func (l *Level) RemoveEnemyAt(idx int) {
	if idx < 0 || idx >= len(l.enemies) {
		return
	}
	l.enemies = append(l.enemies[:idx], l.enemies[idx+1:]...)
}

// SetEnemyEngaged marks the enemy at index as engaged or not.
func (l *Level) SetEnemyEngaged(idx int, v bool) {
	if idx < 0 || idx >= len(l.enemies) {
		return
	}
	// modify the enemy in place
	e := l.enemies[idx]
	e.SetEngaged(v)
	l.enemies[idx] = e
}

// FindClosestCity returns the tile coordinates and distance of the closest city to the player
func (l *Level) FindClosestCity() (TilePoint, float64) {
	pLoc := l.Player.Loc()
	closestTile := TilePoint{-1, -1}
	minDistance := math.MaxFloat64

	for y := 0; y < l.h; y++ {
		for x := 0; x < l.w; x++ {
			tile := l.Tile(TilePoint{x, y})
			if tile != nil && tile.IsCity {
				// Calculate pixel position of this tile
				pixelX := x * l.tileWidth
				pixelY := y * l.tileHeight / 2
				if y%2 != 0 {
					pixelX += l.tileWidth / 2
				}

				// Calculate distance from player
				dx := float64(pixelX - pLoc.X)
				dy := float64(pixelY - pLoc.Y)
				distance := math.Sqrt(dx*dx + dy*dy)

				// Apply camera scale to the distance
				scaledDistance := distance

				if scaledDistance < minDistance {
					minDistance = scaledDistance
					closestTile = TilePoint{x, y}
				}
			}
		}
	}

	return closestTile, minDistance
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
