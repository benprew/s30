package world

import (
	"fmt"
	"image"
	"math"
	"math/rand"

	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/ui/imageutil"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	// Configuration
	MaxRandomEncounters    = 20    // Increased from 10
	EncounterSpawnRate     = 100   // Decreased from 200 (faster spawning)
	EncounterMinDistPlayer = 300.0 // Pixels
	EncounterMaxDistPlayer = 750.0 // Pixels
	EncounterTriggerDist   = 50.0  // Pixels (radius)

	// Sprite constants
	EncounterSpriteRows = 4
	EncounterSpriteCols = 6
)

type RandomEncounter struct {
	Tile        image.Point
	SpriteIndex int
	TerrainType int
}

func (l *Level) LoadRandomEncounterSprites() error {
	// 5 cols, 2 rows. Row 0 is object, Row 1 is shadow.
	sprites, err := imageutil.LoadSpriteSheet(EncounterSpriteCols, EncounterSpriteRows, assets.RandomEncounters_png)
	if err != nil {
		return fmt.Errorf("failed to load random encounter sprites: %w", err)
	}
	fmt.Println("main encounter spr", sprites[0][0].Bounds().Size())
	fmt.Println("shadow encounter spr", sprites[1][0].Bounds().Size())

	l.encounterSprites = sprites
	return nil
}

func (l *Level) SpawnEncounters(count int) {
	pLoc := l.Player.Loc()

	for i := 0; i < count; i++ {
		var tileX, tileY int // Tile coords

		maxAttempts := 100
		valid := false
		spriteIdx := 0

		for attempt := 0; attempt < maxAttempts; attempt++ {
			// Random tile coordinates
			tileX = rand.Intn(l.w)
			tileY = rand.Intn(l.h)

			// Get Tile
			t := l.Tile(image.Point{tileX, tileY})
			if t == nil {
				continue
			}

			// // Check Terrain
			// if t.TerrainType == TerrainWater {
			// 	continue // Impassable
			// }

			// Avoid Cities
			if t.IsCity {
				continue
			}

			// Calculate pixel position
			tLoc := l.TileToPixel(image.Point{tileX, tileY})

			// Check distance from player
			dx := float64(tLoc.X - pLoc.X)
			dy := float64(tLoc.Y - pLoc.Y)
			distance := math.Sqrt(dx*dx + dy*dy)

			switch t.TerrainType {
			case TerrainWater:
				spriteIdx = 0
			case TerrainPlains:
				spriteIdx = 1
			case TerrainMountains:
				spriteIdx = 2
			case TerrainForest:
				spriteIdx = 3
			case TerrainMarsh:
				spriteIdx = 4
			}

			if distance > EncounterMinDistPlayer && distance < EncounterMaxDistPlayer {
				valid = true
				break
			}
		}

		if !valid {
			fmt.Println("unable to create encounter")
			continue // Could not find a valid spot
		}

		// Choose a random sprite index

		t := l.Tile(image.Point{tileX, tileY})
		t.AddRandomEncounter(
			l.encounterSprites[1][spriteIdx],
			l.encounterSprites[0][spriteIdx],
		)

		pixel := l.TileToPixel(image.Point{tileX, tileY})

		fmt.Println("Added encounter at Tile:", tileX, tileY, "Pixel:", pixel.X, pixel.Y, "type:", spriteIdx)

		re := RandomEncounter{
			Tile:        image.Point{tileX, tileY},
			SpriteIndex: spriteIdx,
			TerrainType: t.TerrainType,
		}
		l.randomEncounters = append(l.randomEncounters, re)
	}
}

func dist(p1, p2 image.Point) float64 {
	dx := float64(p1.X - p2.X)
	dy := float64(p1.Y - p2.Y)
	return math.Sqrt(dx*dx + dy*dy)
}

func (l *Level) UpdateEncounters() {
	for i, re := range l.randomEncounters {
		if dist(l.Player.Loc(), l.TileToPixel(re.Tile)) < EncounterTriggerDist {
			l.randomEncounterPending = true
			l.pendingEncounterSprite = re.SpriteIndex
			l.pendingEncounterTerrain = re.TerrainType
			t := l.Tile(re.Tile)
			t.RemoveRandomEncounter()
			l.randomEncounters = append(l.randomEncounters[:i], l.randomEncounters[i+1:]...)
		}
	}

	if l.totalTicks%EncounterSpawnRate == 0 {
		l.SpawnEncounters(1)
	}
}

func (l *Level) RandomEncounterPending() bool {
	return l.randomEncounterPending
}

func (l *Level) TakeRandomEncounter() (spriteIdx int, terrainType int, ok bool) {
	if !l.randomEncounterPending {
		return -1, 0, false
	}
	l.randomEncounterPending = false
	return l.pendingEncounterSprite, l.pendingEncounterTerrain, true
}

var basicLands = []string{"Plains", "Island", "Swamp", "Mountain", "Forest"}

func TerrainToLandName(terrainType int) string {
	switch terrainType {
	case TerrainPlains:
		return basicLands[0]
	case TerrainForest:
		return basicLands[4]
	case TerrainMountains:
		return basicLands[3]
	case TerrainMarsh:
		return basicLands[2]
	case TerrainWater, TerrainSand:
		return basicLands[1]
	case TerrainSnow:
		return basicLands[rand.Intn(len(basicLands))]
	default:
		return basicLands[0]
	}
}

func (l *Level) GetEncounterSprite(index int) *ebiten.Image {
	if len(l.encounterSprites) > 0 && len(l.encounterSprites[0]) > index {
		return l.encounterSprites[0][index]
	}
	return nil
}
