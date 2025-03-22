package sprites

import (
	"bytes"
	"image"
	"image/color"
	_ "image/png"

	"github.com/benprew/s30/art"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	down = iota
	downLeft
	left
	leftUp
	up
	upRight
	right
	downRight
)

type PlayerSprite struct {
	Animations [8][5]*ebiten.Image
}

// LoadSpriteSheet loads the embedded SpriteSheet.
func LoadSpriteSheet(tileWidth, tileHeight int) (*PlayerSprite, error) {
	img, _, err := image.Decode(bytes.NewReader(art.Ego_F_png))
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()
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

	s := &PlayerSprite{}
	for y := range 8 {
		for x := range 5 {
			s.Animations[y][x] = spriteAt(x, y)
		}
	}

	return s, nil
}
