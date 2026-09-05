package world

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/benprew/s30/game/domain"
	"github.com/hajimehoshi/ebiten/v2"
)

// PositionedSprite represents a sprite with positioning information
type PositionedSprite struct {
	Image   *ebiten.Image
	OffsetX float64
	OffsetY float64
}

// Tile represents a space with an x,y coordinate within a Level. Any number of
// sprites may be added to a Tile.
type Tile struct {
	sprites           []*ebiten.Image
	transitionSprites []*PositionedSprite
	terrainSprites    []*PositionedSprite
	positionedSprites []*PositionedSprite
	roadSprites       []*ebiten.Image     // Added for roads
	encounterSprites  []*PositionedSprite // Random encounters
	City              *domain.City        `json:"City,omitempty"`
	IsDungeon         bool                // Indicates if this tile holds a dungeon entrance
	Dungeon           *domain.Dungeon     // Non-nil when IsDungeon is true
	IsCastle          bool                // Indicates if this tile holds a wizard's castle
	Castle            *domain.Castle      // Non-nil when IsCastle is true
	TerrainType       int                 // Added terrain type
	TerrainBand       TerrainBand         // Visual terrain and foliage band
}

func (t *Tile) UnmarshalJSON(data []byte) error {
	type tileJSON struct {
		City        *domain.City    `json:"City,omitempty"`
		LegacyCity  *bool           `json:"IsCity,omitempty"`
		IsDungeon   bool            `json:"IsDungeon"`
		Dungeon     *domain.Dungeon `json:"Dungeon"`
		IsCastle    bool            `json:"IsCastle"`
		Castle      *domain.Castle  `json:"Castle"`
		TerrainType int             `json:"TerrainType"`
		TerrainBand *TerrainBand    `json:"TerrainBand,omitempty"`
	}

	var decoded tileJSON
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	t.City = decoded.City
	if decoded.LegacyCity != nil && !*decoded.LegacyCity {
		t.City = nil
	}
	t.IsDungeon = decoded.IsDungeon
	t.Dungeon = decoded.Dungeon
	t.IsCastle = decoded.IsCastle
	t.Castle = decoded.Castle
	t.TerrainType = decoded.TerrainType
	if decoded.TerrainBand != nil {
		t.TerrainBand = *decoded.TerrainBand
	} else {
		t.TerrainBand = defaultTerrainBand(decoded.TerrainType)
	}
	return nil
}

func defaultTerrainBand(terrain int) TerrainBand {
	switch terrain {
	case TerrainSand:
		return TerrainBandSand
	case TerrainMarsh:
		return TerrainBandMarsh
	case TerrainPlains:
		return TerrainBandPlains
	case TerrainForest:
		return TerrainBandForest
	case TerrainMountains:
		return TerrainBandMountains
	case TerrainSnow:
		return TerrainBandIce
	default:
		return TerrainBandWater
	}
}

// AddSprite adds a sprite to the Tile.
func (t *Tile) AddSprite(s *ebiten.Image) {
	t.sprites = append(t.sprites, s)
}

func (t *Tile) IsRoad() bool {
	return len(t.roadSprites) > 0
}

func (t *Tile) IsCity() bool {
	return t.City != nil
}

// AddFoliageSprite adds a foliage sprite to the Tile with proper positioning.
// Foliage sprites are taller than land tiles, so we need to position them
// so their bottom is centered in the land tile.
func (t *Tile) AddFoliageSprite(s *ebiten.Image) {
	t.AddTerrainSprite(s, 0, -40)
}

// AddTerrainSprite adds one terrain object at an offset from its tile.
func (t *Tile) AddTerrainSprite(s *ebiten.Image, offsetX, offsetY float64) {
	if s == nil {
		return
	}
	t.terrainSprites = append(t.terrainSprites, &PositionedSprite{
		Image:   s,
		OffsetX: offsetX,
		OffsetY: offsetY,
	})
}

// AddTransitionSprite adds a terrain edge below roads and objects.
func (t *Tile) AddTransitionSprite(s *ebiten.Image, offsetX, offsetY float64) {
	if s == nil {
		return
	}
	t.transitionSprites = append(t.transitionSprites, &PositionedSprite{
		Image:   s,
		OffsetX: offsetX,
		OffsetY: offsetY,
	})
}

func (t *Tile) suppressTerrainDecorations() {
	t.terrainSprites = nil
}

// AddCitySprite adds a city sprite to the Tile with proper positioning.
// City sprites are larger and need different offsets than foliage.
func (t *Tile) AddCitySprite(s *ebiten.Image) {
	if s == nil {
		return
	}

	// Determine offsets needed to center the base of the city sprite
	// on the tile, similar to foliage but potentially larger.
	// Base tile is 206x102. Let's assume city sprite base aligns similarly
	// but the sprite is taller. We need the actual city sprite dimensions.
	// If city sprite is ~206x270, the offset might be around -(270 - 102) = -168?
	// This needs tuning based on the actual sprite dimensions and desired look.
	citySpriteHeight := float64(s.Bounds().Dy())
	baseTileHeight := 102.0 // Height of the diamond part of the base tile
	offsetY := -(citySpriteHeight - baseTileHeight)

	// Create a new sprite with positioning information
	cityPosSprite := &PositionedSprite{
		Image: s,
		// OffsetX might be needed if the city sprite isn't perfectly centered horizontally
		OffsetX: 0,
		OffsetY: offsetY, // Adjust Y to align base with tile center
	}

	// Add to positioned sprites (drawn after base tile, potentially overlapping foliage)
	t.positionedSprites = append(t.positionedSprites, cityPosSprite)
}

// AddRoadSprite adds a road sprite to the tile, avoiding duplicates.
func (t *Tile) AddRoadSprite(s *ebiten.Image) {
	if s == nil {
		return
	}
	// Check if this specific sprite is already added to prevent duplicates
	// from path overlaps or multiple connections to the same tile.
	if slices.Contains(t.roadSprites, s) {
		return // Already have this exact sprite instance
	}
	t.roadSprites = append(t.roadSprites, s)
}

func (t *Tile) AddRandomEncounter(s1, s2 *ebiten.Image) {
	// xOffset := 10.0
	yOffset := 2.0
	t.encounterSprites = []*PositionedSprite{
		&PositionedSprite{
			Image:   s1,
			OffsetX: 0,
			OffsetY: -float64(s1.Bounds().Dy()) / yOffset,
		},
		&PositionedSprite{
			Image:   s2,
			OffsetX: 0,
			OffsetY: -float64(s2.Bounds().Dy()) / yOffset,
		},
	}
	fmt.Println("Added random ecounter sprite:")
	fmt.Println(s1.Bounds())
	fmt.Println(t.encounterSprites)
}

func (t *Tile) RemoveRandomEncounter() {
	t.encounterSprites = []*PositionedSprite{}
}

// Draw draws the Tile on the screen using the provided options.
func (t *Tile) Draw(screen *ebiten.Image, options *ebiten.DrawImageOptions) {
	t.drawBase(screen, options)
	t.drawTransitions(screen, options)
	t.drawRoads(screen, options)
	t.drawObjects(screen, options)
}

func (t *Tile) drawBase(screen *ebiten.Image, options *ebiten.DrawImageOptions) {
	for _, s := range t.sprites {
		screen.DrawImage(s, options)
	}
}

func (t *Tile) drawTransitions(screen *ebiten.Image, options *ebiten.DrawImageOptions) {
	drawPositionedSprites(screen, options, t.transitionSprites)
}

func (t *Tile) drawRoads(screen *ebiten.Image, options *ebiten.DrawImageOptions) {
	for _, s := range t.roadSprites {
		if s == nil {
			continue
		}
		screen.DrawImage(s, options)
	}
}

func (t *Tile) drawObjects(screen *ebiten.Image, options *ebiten.DrawImageOptions) {
	drawPositionedSprites(screen, options, t.terrainSprites)
	drawPositionedSprites(screen, options, t.positionedSprites)
	drawPositionedSprites(screen, options, t.encounterSprites)
}

func drawPositionedSprites(screen *ebiten.Image, options *ebiten.DrawImageOptions, sprites []*PositionedSprite) {
	for _, ps := range sprites {
		posOptions := &ebiten.DrawImageOptions{}
		posOptions.GeoM.Translate(ps.OffsetX, ps.OffsetY)
		posOptions.GeoM.Concat(options.GeoM)
		screen.DrawImage(ps.Image, posOptions)
	}
}
