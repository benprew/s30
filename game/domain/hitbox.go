package domain

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type Hitbox struct {
	OffsetX int
	OffsetY int
	Width   int
	Height  int
}

func (h Hitbox) Rect(posX, posY int) image.Rectangle {
	return image.Rect(
		posX+h.OffsetX,
		posY+h.OffsetY,
		posX+h.OffsetX+h.Width,
		posY+h.OffsetY+h.Height,
	)
}

func (h Hitbox) Draw(screen *ebiten.Image, posX, posY int) {
	vector.StrokeRect(
		screen,
		float32(posX+h.OffsetX),
		float32(posY+h.OffsetY),
		float32(h.Width),
		float32(h.Height),
		3,
		color.White,
		false,
	)
}
