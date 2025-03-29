package game

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
}

// AddSprite adds a sprite to the Tile.
func (t *Tile) AddSprite(s *ebiten.Image) {
	t.sprites = append(t.sprites, s)
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
	// Draw regular sprites
	for _, s := range t.sprites {
		screen.DrawImage(s, options)
	}

	// Draw positioned sprites with their offsets
	for _, ps := range t.positionedSprites {
		// Create a copy of the options to avoid modifying the original
		posOptions := &ebiten.DrawImageOptions{}
		posOptions.GeoM.Concat(options.GeoM)

		// Apply the offset
		posOptions.GeoM.Translate(ps.OffsetX, ps.OffsetY)

		screen.DrawImage(ps.Image, posOptions)
	}
}
