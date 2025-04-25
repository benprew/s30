package sprites

import (
	"bytes"
	"image"

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

	tileWidth := width / sprWidth
	tileHeight := height / sprHeight

	sheet := ebiten.NewImageFromImage(img)

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
