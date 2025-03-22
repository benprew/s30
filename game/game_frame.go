package game

import (
	"bytes"
	"image"

	"github.com/benprew/s30/art"
	"github.com/hajimehoshi/ebiten/v2"
)

func LoadWorldFrame() (*ebiten.Image, error) {
	img, _, err := image.Decode(bytes.NewReader(art.WorldFrame_png))
	if err != nil {
		return nil, err
	}

	return ebiten.NewImageFromImage(img), nil
}
