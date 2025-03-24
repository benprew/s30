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
	w, h int

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
	// Create a 108x108 Level.
	l := &Level{
		w:          300,
		h:          300,
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
	foliage, err := sprites.LoadSpriteSheet(5, 11, art.Land_png)
	if err != nil {
		return nil, fmt.Errorf("failed to load embedded spritesheet: %s", err)
	}
	// shadows for lands
	Sfoliage, err := sprites.LoadSpriteSheet(5, 11, art.Sland_png)
	if err != nil {
		return nil, fmt.Errorf("failed to load embedded spritesheet: %s", err)
	}

	noise := generateTerrain(l.w, l.h)
	l.mapTerrainTypes(noise, ss, foliage, Sfoliage)
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

func (l *Level) mapTerrainTypes(terrain [][]float64, ss *SpriteSheet, foliage [][]*ebiten.Image, Sfoliage [][]*ebiten.Image) {
	// Fill each tile with one or more sprites randomly.
	l.tiles = make([][]*Tile, l.h)
	for y := 0; y < l.h; y++ {
		l.tiles[y] = make([]*Tile, l.w)
		for x := 0; x < l.w; x++ {
			t := &Tile{}
			isBorderSpace := x == 0 || y == 0 || x == l.w-1 || y == l.h-1
			val := terrain[y][x]
			folIdx := rand.Intn(11)
			switch {
			case isBorderSpace:
				t.AddSprite(ss.Ice)
			case val < Water:
				t.AddSprite(ss.Water)
			case val < Sand:
				t.AddSprite(ss.Sand) // Use desert sprite for sandy shores
				t.AddSprite(Sfoliage[folIdx][1])
				t.AddSprite(foliage[folIdx][1])
			case val < Marsh:
				t.AddSprite(ss.Marsh)
				t.AddSprite(Sfoliage[folIdx][0])
				t.AddSprite(foliage[folIdx][0])
			case val < Plains:
				t.AddSprite(ss.Plains)
				t.AddSprite(Sfoliage[folIdx][4])
				t.AddSprite(foliage[folIdx][4])
			case val < Forest:
				t.AddSprite(ss.Forest)
				t.AddSprite(Sfoliage[folIdx][2])
				t.AddSprite(foliage[folIdx][2])
			case val < Mountains:
				t.AddSprite(ss.Plains)
				t.AddSprite(Sfoliage[folIdx][3])
				t.AddSprite(foliage[folIdx][3])
			default:
				t.AddSprite(ss.Plains)
				if rand.Float64() < 0.7 {
					t.AddSprite(Sfoliage[folIdx][4])
					t.AddSprite(foliage[folIdx][4])
				}
			}
			l.tiles[y][x] = t
		}
	}
}
