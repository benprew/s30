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
	Shallows  = 0.40 // Shallow water
	Sand      = 0.43 // Beach/Sandy areas
	Marsh     = 0.45 // Swampy areas
	Plains    = 0.60 // Grasslands
	Forest    = 0.75 // Dense forest
	Hills     = 0.85 // Rolling hills
	Mountains = 0.95 // Mountain peaks
	Snow      = 1.0  // Snow-capped peaks
)

// Level represents a Game level.
type Level struct {
	w, h       int
	tiles      [][]*Tile // (Y,X) array of tiles
	tileWidth  int
	tileHeight int
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

	noise := generateTerrain(l.w, l.h)
	l.mapTerrainTypes(noise, ss, foliage, Sfoliage, foliage2, Sfoliage2, Cstline2, citySprites)
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

// mapTerrainTypes assigns terrain and places cities based on noise values.
func (l *Level) mapTerrainTypes(terrain [][]float64, ss *SpriteSheet, foliage, Sfoliage, foliage2, Sfoliage2, Cstline2, citySprites [][]*ebiten.Image) {
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
			switch {
			case isBorderSpace:
				t.AddSprite(ss.Water)
				isWater = true // Border is treated as water for city placement
			case val < Water:
				t.AddSprite(ss.Water)
				isWater = true
				if rand.Float64() < 0.1 {
					t.AddFoliageSprite(Sfoliage2[folIdx][0])
					t.AddFoliageSprite(foliage2[folIdx][0])
				}
			case val < Sand:
				t.AddSprite(ss.Sand) // Use desert sprite for sandy shores
				t.AddFoliageSprite(Sfoliage[folIdx][1])
				t.AddFoliageSprite(foliage[folIdx][1])
			case val < Marsh:
				t.AddSprite(ss.Marsh)
				t.AddFoliageSprite(Sfoliage[folIdx][0])
				t.AddFoliageSprite(foliage[folIdx][0])
			case val < Plains:
				t.AddSprite(ss.Plains)
				t.AddFoliageSprite(Sfoliage[folIdx][4])
				t.AddFoliageSprite(foliage[folIdx][4])
			case val < Forest:
				t.AddSprite(ss.Forest)
				t.AddFoliageSprite(Sfoliage[folIdx][2])
				t.AddFoliageSprite(foliage[folIdx][2])
			case val < Mountains:
				t.AddSprite(ss.Plains)
				t.AddFoliageSprite(Sfoliage[folIdx][3])
				t.AddFoliageSprite(foliage[folIdx][3])
			default:
				t.AddSprite(ss.Plains)
				if rand.Float64() < 0.7 {
					t.AddFoliageSprite(Sfoliage[folIdx][4])
					t.AddFoliageSprite(foliage[folIdx][4])
				}
			}
			l.tiles[y][x] = t

			// If not water and not border, add to potential city locations
			if !isWater && !isBorderSpace {
				validCityLocations = append(validCityLocations, struct{ x, y int }{x, y})
			}
		}
	}

	// Place cities after terrain is mapped
	l.placeCities(validCityLocations, citySprites, 35, 6)
}

// placeCities places a specified number of cities on valid land tiles, ensuring
// they are at least minDistance apart.
func (l *Level) placeCities(validLocations []struct{ x, y int }, citySprites [][]*ebiten.Image, numCities, minDistance int) {
	if len(validLocations) == 0 || numCities <= 0 {
		return // No valid locations or no cities to place
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
			fmt.Println("cityIdx", cityIdx, "cityX", cityX)
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
}

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
