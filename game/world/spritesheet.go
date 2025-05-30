package world

import (
	"bytes"
	"fmt"
	"image"
	"image/color"

	"github.com/benprew/s30/art"
	"github.com/hajimehoshi/ebiten/v2"
)

type SpriteSheet struct {
	Plains *ebiten.Image
	Water  *ebiten.Image
	Sand   *ebiten.Image
	Forest *ebiten.Image
	Marsh  *ebiten.Image
	Ice    *ebiten.Image
}

// LoadWorldTileSheet loads the embedded SpriteSheet.
func LoadWorldTileSheet(tileWidth, tileHeight int) (*SpriteSheet, error) {
	img, _, err := image.Decode(bytes.NewReader(art.Landtile_png))
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	width := bounds.Max.X - bounds.Min.X
	height := bounds.Max.Y - bounds.Min.Y
	fmt.Printf("Loaded sprite sheet dimensions: %dx%d\n", width, height)
	rgba := image.NewRGBA(bounds)

	// Convert indexed color to RGBA and set transparency
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := img.(*image.Paletted).ColorIndexAt(x, y)
			if c == 255 { // Assuming index 255 is the transparent color
				rgba.Set(x, y, color.RGBA{0, 0, 0, 0}) // Transparent
			} else {
				rgba.Set(x, y, img.At(x, y))
			}
		}
	}
	sheet := ebiten.NewImageFromImage(rgba)

	// spriteAt returns a sprite at the provided coordinates.
	spriteAt := func(x, y int) *ebiten.Image {
		return sheet.SubImage(image.Rect(x*tileWidth, (y+1)*tileHeight, (x+1)*tileWidth, y*tileHeight)).(*ebiten.Image)
	}

	// Populate SpriteSheet.
	s := &SpriteSheet{}
	s.Plains = spriteAt(0, 0)
	s.Water = spriteAt(1, 0)
	s.Sand = spriteAt(2, 0)
	s.Forest = spriteAt(3, 0)
	s.Marsh = spriteAt(4, 0)
	s.Ice = spriteAt(5, 0)

	return s, nil
}
