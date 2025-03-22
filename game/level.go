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
	Water  = 0.3
	Marsh  = 0.4
	Plains = 0.6
	Desert = 0.7
	Forest = 0.8
	Ice    = 0.9
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
		w:          100,
		h:          100,
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

	noise := generateTerrain(l.w, l.h)
	l.mapTerrainTypes(noise, ss, foliage)
	return l, nil
}

func generateTerrain(w, h int) [][]float64 {
	p := perlin.NewPerlin(2, 2, 3, Seed) // Perlin noise generator
	terrain := make([][]float64, h)

	for y := 0; y < h; y++ {
		terrain[y] = make([]float64, w)
		for x := 0; x < w; x++ {
			nx := float64(x) / float64(w) // Normalize coordinates
			ny := float64(y) / float64(h)
			noiseValue := p.Noise2D(nx*10, ny*10) // Adjust scale for variation
			terrain[y][x] = (noiseValue + 1) / 2  // Normalize to 0-1 range
		}
	}
	return terrain
}

func (l *Level) mapTerrainTypes(terrain [][]float64, ss *SpriteSheet, foliage [][]*ebiten.Image) {
	// Fill each tile with one or more sprites randomly.
	l.tiles = make([][]*Tile, l.h)
	for y := 0; y < l.h; y++ {
		l.tiles[y] = make([]*Tile, l.w)
		for x := 0; x < l.w; x++ {
			t := &Tile{}
			isBorderSpace := x == 0 || y == 0 || x == l.w-1 || y == l.h-1
			val := terrain[y][x]
			switch {
			case isBorderSpace:
				t.AddSprite(ss.Ice)
			case val < Water:
				t.AddSprite(ss.Water)
			case val < Marsh:
				t.AddSprite(ss.Marsh)
				if rand.Float64() < 0.3 {
					t.foliage = foliage[rand.Intn(11)][0]
				}
			case val < Plains:
				t.AddSprite(ss.Plains)
				if rand.Float64() < 0.2 {
					t.foliage = foliage[rand.Intn(11)][4]
				}
			case val < Desert:
				t.AddSprite(ss.Desert)
				if rand.Float64() < 0.15 {
					t.foliage = foliage[rand.Intn(11)][1]
				}
			case val < Forest:
				t.AddSprite(ss.Forest)
				if rand.Float64() < 0.8 {
					t.foliage = foliage[rand.Intn(11)][2]
				}
			default:
				t.AddSprite(ss.Plains)
			}
			l.tiles[y][x] = t
		}
	}
}
