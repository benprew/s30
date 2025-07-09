package world

import (
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
	positionedSprites []*PositionedSprite
	roadSprites       []*ebiten.Image // Added for roads
	IsCity            bool            // Indicates if this tile represents a city
	City              City
	TerrainType       int // Added terrain type
}

// AddSprite adds a sprite to the Tile.
func (t *Tile) AddSprite(s *ebiten.Image) {
	t.sprites = append(t.sprites, s)
}

func (t *Tile) IsRoad() bool {
	return len(t.roadSprites) > 0
}

// AddFoliageSprite adds a foliage sprite to the Tile with proper positioning.
// Foliage sprites are taller than land tiles, so we need to position them
// so their bottom is centered in the land tile.
func (t *Tile) AddFoliageSprite(s *ebiten.Image) {
	if s == nil {
		return
	}

	// Create a new sprite with positioning information
	foliageSprite := &PositionedSprite{
		Image: s,
		// Foliage is 206x134, land tile is 206x102
		OffsetY: -40,
	}

	t.positionedSprites = append(t.positionedSprites, foliageSprite)
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
	for _, existing := range t.roadSprites {
		if existing == s {
			return // Already have this exact sprite instance
		}
	}
	t.roadSprites = append(t.roadSprites, s)
}

// Transition tile between terrain types (Cstline/Cstline2)
//
// sides:
// 2,2,3,2 L,T,TL,R
// 4,3,4,3 T,TR,T,RB
// 2,3,2,4 B,BR,L,B
// 3,4,3,2 BL,B,BL,L
// 3,2,4,3 BL,T,L,TL
// 4,3,2,3 L,TR,R,TR
// 3,4,3,4 BR,R,BR,R

// Draw draws the Tile on the screen using the provided options.
func (t *Tile) Draw(screen *ebiten.Image, options *ebiten.DrawImageOptions) {
	// Draw regular sprites (base terrain)
	for _, s := range t.sprites {
		screen.DrawImage(s, options)
	}

	// Draw road first if it exists, so it's under other elements
	for _, s := range t.roadSprites {
		if s == nil {
			continue
		}
		screen.DrawImage(s, options)
	}

	// Draw positioned sprites (foliage, cities) with their offsets
	for _, ps := range t.positionedSprites {
		// Create a copy of the options to avoid modifying the original
		posOptions := &ebiten.DrawImageOptions{}
		posOptions.GeoM.Translate(ps.OffsetX, ps.OffsetY)
		posOptions.GeoM.Concat(options.GeoM)

		screen.DrawImage(ps.Image, posOptions)
	}
}
