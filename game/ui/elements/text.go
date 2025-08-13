package elements

import (
	"image/color"

	"github.com/benprew/s30/game/ui/fonts"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

type Text struct {
	text  string
	font  text.Face
	color color.Color
	X     int
	Y     int
}

// For drawing text, similar to buttons
func NewText(size float64, txt string, x, y int) *Text {
	fontFace := &text.GoTextFace{
		Source: fonts.MtgFont,
		Size:   size,
	}

	return &Text{text: txt, font: fontFace, color: color.White, X: x, Y: y}
}

func (t *Text) Draw(screen *ebiten.Image, opts *ebiten.DrawImageOptions) {
	R, G, B, A := t.color.RGBA()
	options := &ebiten.DrawImageOptions{}
	options.GeoM.Concat(opts.GeoM)
	options.GeoM.Translate(float64(t.X), float64(t.Y))
	options.ColorScale.Scale(float32(R), float32(G), float32(B), float32(A))
	textOpts := text.DrawOptions{DrawImageOptions: *options}

	text.Draw(screen, t.text, t.font, &textOpts)
}
