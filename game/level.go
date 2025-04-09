package game

import (
	"fmt"
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

// Level represents a Game level.
type Level struct {
	w, h           int
	tiles          [][]*Tile         // (Y,X) array of tiles
	roadSprites    [][]*ebiten.Image // Store loaded road sprites
	roadSpriteInfo [][]string        // Store road sprite connection info
	tileWidth      int
	tileHeight     int
}

// Tile returns the tile at the provided coordinates, or nil.
func (l *Level) Tile(x, y int) *Tile {
	if x >= 0 && y >= 0 && x < l.w && y < l.h {
		return l.tiles[y][x]
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
	l.roadSpriteInfo = [][]string{
		{"NE,SE,SW,NW", "NE,SW", "E,W", "SE,NW", "N,S", "NE,SW"},
		{"E,W", "SE,NW", "N,S", "NE,SW"},
	}

	noise := generateTerrain(l.w, l.h)
	// mapTerrainTypes now returns valid city locations and sets Tile.TerrainType
	validCityLocations := l.mapTerrainTypes(noise, ss, foliage, Sfoliage, foliage2, Sfoliage2, Cstline2, citySprites)

	// placeCities now returns the list of placed cities
	placedCities := l.placeCities(validCityLocations, citySprites, 35, 6)

	// Connect the placed cities with roads
	l.connectCitiesWithRoads(placedCities) // Pass only cities, roads are in l

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
func (l *Level) mapTerrainTypes(terrain [][]float64, ss *SpriteSheet, foliage, Sfoliage, foliage2, Sfoliage2, Cstline2, citySprites [][]*ebiten.Image) []struct{ x, y int } {
	// Fill each tile with one or more sprites randomly.
	l.tiles = make([][]*Tile, l.h)
	validCityLocations := []struct{ x, y int }{} // Store potential city coordinates

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
				validCityLocations = append(validCityLocations, struct{ x, y int }{x, y})
			}
		}
	}
	return validCityLocations // Return potential locations
}

// placeCities places cities and returns their locations.
func (l *Level) placeCities(validLocations []struct{ x, y int }, citySprites [][]*ebiten.Image, numCities, minDistance int) []struct{ x, y int } {
	if len(validLocations) == 0 || numCities <= 0 {
		fmt.Println("Warning: No valid locations provided or numCities <= 0.")
		return nil
	}

	placedCities := []struct{ x, y int }{}

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
			dist := absInt(loc.x-city.x) + absInt(loc.y-city.y)
			if dist <= minDistance {
				isValidPlacement = false
				break
			}
		}

		if !isValidPlacement {
			continue
		}

		// Place the city
		tile := l.Tile(loc.x, loc.y)
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
	}

	if len(placedCities) < numCities {
		fmt.Printf("Warning: Could only place %d out of %d requested cities with min distance %d.\n", len(placedCities), numCities, minDistance)
	}
	return placedCities // Return the list of placed cities
}

// --- Road Generation Logic ---

// Coordinate struct for pathfinding nodes
type bfsNode struct {
	x, y int
}

// Checks if a tile is suitable for placing a road segment.
func (l *Level) isTileTraversableForRoads(x, y int) bool {
	tile := l.Tile(x, y)
	// Cannot path through non-existent tiles, cities, water, mountains, or border.
	if tile == nil || tile.IsCity {
		return false
	}
	switch tile.TerrainType {
	case TerrainWater:
		return false
	default: // Allow roads on Sand, Marsh, Plains, Forest, Hills
		return true
	}
}

// findPathBFS finds a path between start and end coordinates using Breadth-First Search.
// It avoids non-traversable tiles.
func (l *Level) findPathBFS(start, end struct{ x, y int }) []struct{ x, y int } {
	queue := []bfsNode{{start.x, start.y}}
	visited := make(map[bfsNode]bool)
	parent := make(map[bfsNode]bfsNode)
	visited[bfsNode{start.x, start.y}] = true

	found := false
	endNode := bfsNode{end.x, end.y}

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		if curr == endNode {
			found = true
			break
		}

		// Explore neighbors (N, E, S, W) - Order can influence path shape slightly
		for _, offset := range []struct{ dx, dy int }{{0, -1}, {1, 0}, {0, 1}, {-1, 0}} {
			nx, ny := curr.x+offset.dx, curr.y+offset.dy
			neighborNode := bfsNode{nx, ny}

			// Check bounds
			if nx < 0 || nx >= l.w || ny < 0 || ny >= l.h {
				continue
			}

			// Check visited
			if visited[neighborNode] {
				continue
			}

			// Check traversability (allow pathing *into* the end city tile)
			isEndTile := (nx == end.x && ny == end.y)
			targetTile := l.Tile(nx, ny) // Tile being considered for traversal
			if targetTile != nil && (l.isTileTraversableForRoads(nx, ny) || isEndTile) {
				visited[neighborNode] = true
				parent[neighborNode] = curr
				queue = append(queue, neighborNode)
			}
		}
	}

	if !found {
		// fmt.Printf("BFS: No path found from (%d,%d) to (%d,%d)\n", start.x, start.y, end.x, end.y)
		return nil
	}

	// Reconstruct path from end to start
	path := []struct{ x, y int }{}
	curr := endNode
	for curr != (bfsNode{start.x, start.y}) {
		path = append(path, struct{ x, y int }{curr.x, curr.y})
		p, ok := parent[curr]
		if !ok {
			fmt.Printf("BFS Path Reconstruction Error: No parent for %+v\n", curr)
			return nil // Error in path reconstruction
		}
		curr = p
	}
	path = append(path, struct{ x, y int }{start.x, start.y}) // Add start node

	// Reverse path to be start -> end
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path
}

// getRoadSpritesForTile determines the correct road sprite based on path direction.
// Currently only supports straight N-S and E-W segments based on roadsInfo.
func (l *Level) getRoadSpritesForTile(prev, curr, next struct{ x, y int }) []*ebiten.Image {
	dxIn := curr.x - prev.x
	dyIn := curr.y - prev.y
	dxOut := next.x - curr.x
	dyOut := next.y - curr.y

	sprites := make([]*ebiten.Image, 1)

	var requiredPattern string
	// Check for vertical movement (allow for slight diagonal if needed, but focus on cardinal)
	if dyIn != 0 && dyOut != 0 && dxIn == 0 && dxOut == 0 { // Moving N->S or S->N
		requiredPattern = "N,S"
	} else if dxIn != 0 && dxOut != 0 && dyIn == 0 && dyOut == 0 { // Moving W->E or E->W
		requiredPattern = "E,W"
	} else {
		// Turn detected or diagonal - currently unsupported based on simple roadsInfo interpretation.
		// A more complex version could try to find corner sprites like "NE,SW" if needed.
		// For now, let's try to find *any* matching sprite based on the simple patterns.
		// If moving vertically then horizontally, maybe use "N,S" then "E,W"?
		// Or just return nil for turns. Let's return nil for now.
		// fmt.Printf("Turn/Diagonal detected at (%d,%d): In(%d,%d) Out(%d,%d) - No simple sprite\n", curr.x, curr.y, dxIn, dyIn, dxOut, dyOut)
		return nil
	}

	// Find the sprite matching the pattern in the stored roadSpriteInfo
	for r, rowInfo := range l.roadSpriteInfo {
		for c, pattern := range rowInfo {
			if pattern == requiredPattern {
				// Ensure indices are valid for roadSprites
				if r < len(l.roadSprites) && c < len(l.roadSprites[r]) {
					// fmt.Printf("Found sprite for %s at [%d][%d] for tile (%d,%d)\n", requiredPattern, r, c, curr.x, curr.y)
					sprites = append(sprites, l.roadSprites[r][c])
				}
			}
		}
	}
	return sprites
}

// placeRoadSpritesOnPath adds road sprites to tiles along a given path.
func (l *Level) placeRoadSpritesOnPath(path []struct{ x, y int }) {
	if len(path) < 2 {
		return // Need at least start and end
	}

	for i := 1; i < len(path)-1; i++ { // Iterate through intermediate path tiles (skip start/end cities)
		prev := path[i-1]
		curr := path[i]
		next := path[i+1]

		tile := l.Tile(curr.x, curr.y)
		// Double check tile exists and is not a city (should be guaranteed by BFS traversability)
		if tile == nil || tile.IsCity {
			fmt.Printf("Warning: Trying to place road on nil or city tile at (%d,%d)\n", curr.x, curr.y)
			continue
		}

		roadSprites := l.getRoadSpritesForTile(prev, curr, next)
		if len(roadSprites) != 0 {
			// If a road already exists, we might need logic for intersections later.
			// For now, just overwrite. If getRoadSpriteForTile returns nil (e.g., for a turn),
			// no road sprite is placed for that segment.
			tile.roadSprites = roadSprites
			fmt.Printf("Placed road sprite at (%d, %d)\n", curr.x, curr.y)
		}
	}
}

// connectCitiesWithRoads finds paths between consecutive cities and places roads.
func (l *Level) connectCitiesWithRoads(cities []struct{ x, y int }) {
	if len(cities) < 2 {
		fmt.Println("Not enough cities to connect.")
		return // Need at least two cities to connect
	}

	fmt.Printf("Connecting %d cities with roads...\n", len(cities))
	// Connect cities sequentially (city 0 -> 1, 1 -> 2, etc.)
	for i := 0; i < len(cities)-1; i++ {
		startCity := cities[i]
		endCity := cities[i+1]

		// fmt.Printf("Finding path between city %d (%d,%d) and city %d (%d,%d)\n", i, startCity.x, startCity.y, i+1, endCity.x, endCity.y)
		path := l.findPathBFS(startCity, endCity)

		if path != nil {
			// fmt.Printf("Path found (len %d), placing sprites...\n", len(path))
			l.placeRoadSpritesOnPath(path)
		} else {
			fmt.Printf("Failed to find path between city %d (%d,%d) and city %d (%d,%d)\n", i, startCity.x, startCity.y, i+1, endCity.x, endCity.y)
		}
	}
	fmt.Println("Finished connecting cities.")
}

// --- End Road Generation Logic ---

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

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
			tile := l.Tile(x, y)
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
	tileY = pixelY / tileHeight
	tileX = (pixelX - (tileY%2)*(tileWidth/2)) / tileWidth

	// Ensure the tile coordinates are within bounds
	if pixelX < 0 || tileX >= l.w || pixelY < 0 || tileY >= l.h {
		return -1, -1 // Return invalid coordinates if out of bounds
	}

	return tileX, tileY
}
