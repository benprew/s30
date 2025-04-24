package sprites

import (
	"bytes"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// LoadSpriteSheet loads the embedded SpriteSheet.
// sprWidth is the number of images horizontally in the sheet
// sprHeight is the number of images vertically in the sheet
// pixel size of a single sprite iamge is deteremined by the image width divided by sprWidth
func LoadSpriteSheet(sprWidth, sprHeight int, file []byte) ([][]*ebiten.Image, error) {
	img, _, err := image.Decode(bytes.NewReader(file))
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	width := bounds.Max.X - bounds.Min.X
	height := bounds.Max.Y - bounds.Min.Y
	rgba := image.NewRGBA(bounds)

	tileWidth := width / sprWidth
	tileHeight := height / sprHeight

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

	s := make([][]*ebiten.Image, sprHeight)
	for y := 0; y < sprHeight; y++ {
		s[y] = make([]*ebiten.Image, sprWidth)
		for x := 0; x < sprWidth; x++ {
			s[y][x] = spriteAt(x, y)
		}
	}

	return s, nil
}
