package game

import (
	"fmt"
	"image"
	"math/rand"

	"github.com/aquilax/go-perlin"
	"github.com/benprew/s30/art"
	"github.com/benprew/s30/game/sprites"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	Seed = 12345 // Fixed seed for consistency

	// Terrain thresholds
	Water     = 0.35 // Deeper water
	Sand      = 0.43 // Beach/Sandy areas
	Marsh     = 0.45 // Swampy areas
	Plains    = 0.60 // Grasslands
	Forest    = 0.75 // Dense forest
	Mountains = 0.95 // Mountain peaks
	Snow      = 1.0  // Snow/Ice tile
)

// Terrain type constants
const (
	TerrainUndefined = iota
	TerrainWater
	TerrainSand
	TerrainMarsh
	TerrainPlains
	TerrainForest
	TerrainMountains
	TerrainSnow
)

type TilePoint image.Point

// different directions based on which y row you're on, because it's a zigzag pattern
var Directions = [2][8]TilePoint{
	{{0, 2}, {0, -2}, {1, 0}, {-1, 0}, {0, -1}, {-1, -1}, {0, 1}, {-1, 1}},
	{{0, 2}, {0, -2}, {1, 0}, {-1, 0}, {1, -1}, {0, -1}, {1, 1}, {0, 1}},
}
var DirNames = []string{"N", "S", "E", "W", "NE", "NW", "SE", "SW"}

// Level represents a Game level.
type Level struct {
	w, h       int
	tiles      [][]*Tile // (Y,X) array of tiles
	tileWidth  int
	tileHeight int

	roadSprites    [][]*ebiten.Image // Sprites for roads
	roadSpriteInfo [][]string        // Maps sprite index to direction string (e.g., "N", "NE")
}

// Tile returns the tile at the provided coordinates, or nil.
func (l *Level) Tile(p TilePoint) *Tile {
	if p.X >= 0 && p.Y >= 0 && p.X < l.w && p.Y < l.h {
		return l.tiles[p.Y][p.X]
	}
	return nil
}

// Size returns the size of the Level.
func (l *Level) Size() (width, height int) {
	return l.w, l.h
}

// NewLevel returns a new randomly generated Level.
func NewLevel() (*Level, error) {
	l := &Level{
		w:          56,
		h:          38,
		tileWidth:  206,
		tileHeight: 102,
	}

	// Load embedded SpriteSheet.
	ss, err := LoadSpriteSheet(l.tileWidth, l.tileHeight)
	if err != nil {
		return nil, fmt.Errorf("failed to load embedded spritesheet: %s", err)
	}

	// widths are the 5 terrain types:
	// marsh, desert, forest, mountain, plains
	// foliage is 206x134
	// land tile is 206x102
	foliage, err := sprites.LoadSpriteSheet(5, 11, art.Land_png)
	if err != nil {
		return nil, fmt.Errorf("failed to load embedded spritesheet: %s", err)
	}
	// shadows for lands
	Sfoliage, err := sprites.LoadSpriteSheet(5, 11, art.Sland_png)
	if err != nil {
		return nil, fmt.Errorf("failed to load embedded spritesheet: %s", err)
	}

	foliage2, err := sprites.LoadSpriteSheet(5, 11, art.Land2_png)
	if err != nil {
		return nil, fmt.Errorf("failed to load embedded spritesheet: %s", err)
	}
	Sfoliage2, err := sprites.LoadSpriteSheet(5, 11, art.Sland2_png)
	if err != nil {
		return nil, fmt.Errorf("failed to load embedded spritesheet: %s", err)
	}

	Cstline2, err := sprites.LoadSpriteSheet(4, 14, art.Cstline2_png)
	if err != nil {
		return nil, fmt.Errorf("failed to load embedded spritesheet: %s", err)
	}

	citySprites, err := sprites.LoadSpriteSheet(6, 4, art.Cities1_png)
	if err != nil {
		return nil, fmt.Errorf("failed to load city spritesheet Castles1.spr.png: %w", err)
	}

	roads, err := sprites.LoadSpriteSheet(6, 2, art.Roads_png)
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

	// placeCities now returns the list of placed cities
	_ = l.placeCities(validCityLocations, citySprites, 35, 6)

	return l, nil
}

func generateTerrain(w, h int) [][]float64 {
	// Multiple Perlin noise generators with different frequencies
	baseNoise := perlin.NewPerlin(2, 2, 3, Seed)
	riverNoise := perlin.NewPerlin(2, 2, 4, Seed+1)
	forestNoise := perlin.NewPerlin(3, 2, 3, Seed+2)
	desertNoise := perlin.NewPerlin(2, 2, 3, Seed+3)

	terrain := make([][]float64, h)
	rivers := make([][]float64, h)
	forests := make([][]float64, h)

	// Generate base terrain
	for y := 0; y < h; y++ {
		terrain[y] = make([]float64, w)
		rivers[y] = make([]float64, w)
		forests[y] = make([]float64, w)

		for x := 0; x < w; x++ {
			nx := float64(x) / float64(w)
			ny := float64(y) / float64(h)

			// Base terrain (mountains, plains)
			baseValue := (baseNoise.Noise2D(nx*8, ny*8) + 1) / 2

			// River channels
			riverValue := (riverNoise.Noise2D(nx*4, ny*4) + 1) / 2

			// Forest distribution
			forestValue := (forestNoise.Noise2D(nx*12, ny*12) + 1) / 2

			// Combine layers
			terrain[y][x] = baseValue
			rivers[y][x] = riverValue
			forests[y][x] = forestValue

			// Create rivers where river noise is within certain range
			if riverValue > 0.48 && riverValue < 0.52 {
				terrain[y][x] *= 0.3 // Make it water

				// Add sandy shores near water
				if riverValue > 0.49 && riverValue < 0.51 {
					terrain[y][x] = 0.42 // Beach/Sand threshold
				}
			}

			// Add desert regions
			desertValue := (desertNoise.Noise2D(nx*6, ny*6) + 1) / 2
			if desertValue > 0.7 && baseValue > 0.3 && baseValue < 0.8 {
				terrain[y][x] = 0.65 // Desert threshold
			}

			// Add forest clusters
			if forestValue > 0.6 && baseValue > 0.4 && baseValue < 0.8 && desertValue < 0.6 {
				terrain[y][x] = 0.75 // Forest threshold
			}
		}
	}

	// Smooth transitions
	smoothTerrain := make([][]float64, h)
	for y := 0; y < h; y++ {
		smoothTerrain[y] = make([]float64, w)
		for x := 0; x < w; x++ {
			sum := 0.0
			count := 0.0

			// Average with neighbors
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					newY, newX := y+dy, x+dx
					if newY >= 0 && newY < h && newX >= 0 && newX < w {
						sum += terrain[newY][newX]
						count++
					}
				}
			}
			smoothTerrain[y][x] = sum / count
		}
	}

	return smoothTerrain
}

// mapTerrainTypes assigns terrain based on noise values and returns potential city locations.
func (l *Level) mapTerrainTypes(terrain [][]float64, ss *SpriteSheet, foliage, Sfoliage, foliage2, Sfoliage2, Cstline2, citySprites [][]*ebiten.Image) []TilePoint {
	// Fill each tile with one or more sprites randomly.
	l.tiles = make([][]*Tile, l.h)
	validCityLocations := []TilePoint{} // Store potential city coordinates

	for y := range l.h {
		l.tiles[y] = make([]*Tile, l.w)
		for x := range l.w {
			t := &Tile{}
			isBorderSpace := x < 4 || y < 8 || x > l.w-4 || y > l.h-8
			val := terrain[y][x]
			folIdx := rand.Intn(11)
			isWater := false // Track if the tile is water
			var terrainType int
			switch {
			case isBorderSpace:
				t.AddSprite(ss.Water)
				isWater = true // Border is treated as water for city placement
				terrainType = TerrainWater
			case val < Water:
				t.AddSprite(ss.Water)
				isWater = true
				terrainType = TerrainWater
				if rand.Float64() < 0.1 {
					t.AddFoliageSprite(Sfoliage2[folIdx][0])
					t.AddFoliageSprite(foliage2[folIdx][0])
				}
			case val < Sand:
				t.AddSprite(ss.Sand) // Use desert sprite for sandy shores
				terrainType = TerrainSand
				t.AddFoliageSprite(Sfoliage[folIdx][1])
				t.AddFoliageSprite(foliage[folIdx][1])
			case val < Marsh:
				t.AddSprite(ss.Marsh)
				terrainType = TerrainMarsh
				t.AddFoliageSprite(Sfoliage[folIdx][0])
				t.AddFoliageSprite(foliage[folIdx][0])
			case val < Plains:
				t.AddSprite(ss.Plains)
				terrainType = TerrainPlains
				t.AddFoliageSprite(Sfoliage[folIdx][4])
				t.AddFoliageSprite(foliage[folIdx][4])
			case val < Forest:
				t.AddSprite(ss.Forest)
				terrainType = TerrainForest
				t.AddFoliageSprite(Sfoliage[folIdx][2])
				t.AddFoliageSprite(foliage[folIdx][2])
			default: // Assuming this is Mountains+
				t.AddSprite(ss.Plains)         // Base is plains
				terrainType = TerrainMountains // Higher ground
				// Add mountain foliage/sprites if available, similar to hills
				t.AddFoliageSprite(Sfoliage[folIdx][3])
				t.AddFoliageSprite(foliage[folIdx][3])
			}
			t.TerrainType = terrainType // Set the type on the tile
			l.tiles[y][x] = t

			// If not water and not border, add to potential city locations
			if !isWater && !isBorderSpace {
				validCityLocations = append(validCityLocations, TilePoint{x, y})
			}
		}
	}
	return validCityLocations // Return potential locations
}

// placeCities places cities and returns their locations.
func (l *Level) placeCities(validLocations []TilePoint, citySprites [][]*ebiten.Image, numCities, minDistance int) []TilePoint {
	if len(validLocations) == 0 || numCities <= 0 {
		fmt.Println("Warning: No valid locations provided or numCities <= 0.")
		return nil
	}

	placedCities := []TilePoint{}

	// Shuffle valid locations for random placement
	rand.Shuffle(len(validLocations), func(i, j int) {
		validLocations[i], validLocations[j] = validLocations[j], validLocations[i]
	})

	for _, loc := range validLocations {
		if len(placedCities) >= numCities {
			break // Reached the desired number of cities
		}

		// Check distance from already placed cities
		isValidPlacement := true
		for _, city := range placedCities {
			dist := absInt(loc.X-city.X) + absInt(loc.Y-city.Y)
			if dist <= minDistance {
				isValidPlacement = false
				break
			}
		}

		if !isValidPlacement {
			continue
		}

		// Place the city
		tile := l.Tile(loc)
		if tile != nil { // Should always be non-nil based on how validLocations is generated
			cityIdx := rand.Intn(12)
			cityX := cityIdx % 6
			cityY := 0
			if cityIdx > 5 {
				cityY = 2
			}

			tile.IsCity = true
			tile.AddCitySprite(citySprites[cityY][cityX])
			tile.AddCitySprite(citySprites[cityY+1][cityX])
			placedCities = append(placedCities, loc)
			// fmt.Printf("Placed city at (%d, %d)\n", loc.x, loc.y) // Optional: for debugging
		}
		// connect the city to the nearest road/city
		if len(placedCities) > 1 {
			path := l.connectCityBFS(loc)
			// fmt.Println(path)
			l.drawRoadAlongPath(path)
		}
	}

	if len(placedCities) < numCities {
		fmt.Printf("Warning: Could only place %d out of %d requested cities with min distance %d.\n", len(placedCities), numCities, minDistance)
	}
	return placedCities // Return the list of placed cities
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// connectCityBFS finds the shortest path from a new city to the nearest existing
// road tile or another city using Breadth-First Search.
func (l *Level) connectCityBFS(start TilePoint) []TilePoint {
	queue := []TilePoint{start}
	visited := make(map[TilePoint]TilePoint) // Stores visited node -> parent node for path reconstruction
	visited[start] = start                   // Mark start as visited, parent is itself

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		// fmt.Println(current, start)

		// Check if the current tile is a target (existing road or another city)
		// Don't target the start city itself if it's the first city placed and roadTiles is empty.
		tile := l.Tile(current)
		if current != start && (tile.IsCity || tile.IsRoad()) {
			// Target found, reconstruct path
			path := []TilePoint{}
			temp := current
			for temp != start { // Backtrack until we reach the start node
				path = append(path, temp)
				// Check if parent exists to prevent infinite loop if start node wasn't set correctly
				parent, ok := visited[temp]
				if !ok || parent == temp {
					fmt.Printf("Error reconstructing path: Parent not found or is self for %v\n", temp)
					return nil // Error in path reconstruction
				}
				temp = parent
			}
			path = append(path, start) // Add the start node itself
			// fmt.Printf("BFS Path found from %v to %v: %v\n", start, current, path)
			// fmt.Printf("Visited: %+v\n", visited)
			return path
		}

		// Explore neighbors (including diagonals)
		// because we render in a zigzag pattern, the neighbors are more than the range -1, +1
		dirs := Directions[current.Y%2]
		// fmt.Println("current:", current, "dirs:", dirs)
		for _, n := range dirs {
			neighborPos := TilePoint{X: current.X + n.X, Y: current.Y + n.Y}
			// fmt.Println("current:", current, "neighborPos:", neighborPos)

			// Check bounds
			if neighborPos.X < 0 || neighborPos.X >= l.w || neighborPos.Y < 0 || neighborPos.Y >= l.h {
				continue
			}

			tile := l.Tile(neighborPos)

			if tile.TerrainType == TerrainWater {
				continue
			}

			// Check if visited
			if _, seen := visited[neighborPos]; !seen {
				visited[neighborPos] = current // Mark visited and store parent
				queue = append(queue, neighborPos)
			}
		}
	}

	panic(fmt.Sprintf("Warning: BFS from %v found no target road or city.\n", start))
}

// getDirection determines the compass direction from one tile to an adjacent tile.
// because we render tiles in a zigzag pattern, it's not a simple X/Y grid
func getDirection(from, to TilePoint) string {
	dx := to.X - from.X
	dy := to.Y - from.Y

	dir := TilePoint{dx, dy}

	dirs := Directions[from.Y%2]
	for i, d := range dirs {
		if d == dir {
			return DirNames[i]
		}
	}
	panic(fmt.Sprintf("unknown dir: %+v, from: %v, to: %v", dir, from, to))
}

// getRoadSprite finds the road sprite corresponding to a specific exit direction.
func (l *Level) getRoadSprite(direction string) *ebiten.Image {
	for r, row := range l.roadSpriteInfo {
		for c, dir := range row {
			if dir == direction {
				// Ensure the sprite exists at this index
				if r < len(l.roadSprites) && c < len(l.roadSprites[r]) && l.roadSprites[r][c] != nil {
					return l.roadSprites[r][c]
				}
				panic(fmt.Sprintf("Warning: Road sprite for direction %s at [%d][%d] not found or is nil.\n", direction, r, c))
			}
		}
	}
	panic(fmt.Sprintf("Warning: No road sprite definition found for direction %s.\n", direction))
}

// drawRoadAlongPath adds road sprites to tiles along a given path.
func (l *Level) drawRoadAlongPath(path []TilePoint) {
	fmt.Println("path:", path)
	if len(path) < 2 {
		return // Need at least two points for a path segment
	}

	for i, currentPos := range path {
		tile := l.Tile(currentPos)
		if tile == nil {
			fmt.Printf("Warning: Tile not found at %v during road drawing.\n", currentPos)
			continue
		}

		// Determine incoming and outgoing directions relative to the current tile
		var incomingDirFromPrev, outgoingDirToNext string

		if i > 0 { // Has a previous node
			// Direction from previous node *to* current node
			incomingDirFromPrev = getDirection(path[i-1], currentPos)
		}
		if i < len(path)-1 { // Has a next node
			// Direction from current node *to* next node
			outgoingDirToNext = getDirection(currentPos, path[i+1])
		}

		// Add road sprites based on directions
		// We need the sprite representing the segment leaving the *current* tile

		// If there's a previous tile, add the road segment pointing back to it.
		if incomingDirFromPrev != "" {
			// The sprite needed is the one exiting the *current* tile towards the *previous* tile.
			exitDirTowardsPrev := getDirection(currentPos, path[i-1])
			sprite := l.getRoadSprite(exitDirTowardsPrev)
			if sprite != nil {
				tile.AddRoadSprite(sprite)
			} else if exitDirTowardsPrev != "" {
				// fmt.Printf("Debug: No sprite found for exit direction %s at %v\n", exitDirTowardsPrev, currentPos)
			}
		}
		// If there's a next tile, add the road segment pointing towards it.
		if outgoingDirToNext != "" {
			// The sprite needed is the one exiting the *current* tile towards the *next* tile.
			sprite := l.getRoadSprite(outgoingDirToNext)
			if sprite != nil {
				tile.AddRoadSprite(sprite)
			} else if outgoingDirToNext != "" {
				// fmt.Printf("Debug: No sprite found for exit direction %s at %v\n", outgoingDirToNext, currentPos)
			}
		}
	}
}

// --- End Road Generation Logic ---

func (l *Level) RenderZigzag(screen *ebiten.Image, pX, pY, padX, padY int) {
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
			op.GeoM.Translate(float64(screenX), float64(screenY))
			tile.Draw(screen, 1.0, op)
		}
	}
}

func (l *Level) PointToTile(pixelX, pixelY int) (tileX, tileY int) {
	tileWidth := l.tileWidth
	tileHeight := l.tileHeight

	// Calculate the approximate row and column
	tileY = pixelY / (tileHeight / 2)
	tileX = (pixelX - (tileY%2)*(tileWidth/2)) / tileWidth

	// Ensure the tile coordinates are within bounds
	if pixelX < 0 || tileX >= l.w || pixelY < 0 || tileY >= l.h {
		return -1, -1 // Return invalid coordinates if out of bounds
	}

	return tileX, tileY
}

func PrintLevel(l *Level) {
	for i, row := range l.tiles {
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
