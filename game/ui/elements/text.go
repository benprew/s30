package elements

import (
	"image/color"

	"github.com/benprew/s30/game/ui/fonts"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// TODO: TextBox should be a Rectangle and text should
// flow into textbox size
type Text struct {
	text  string
	font  text.Face
	Color color.Color
	X     int
	Y     int
}

// For drawing text, similar to buttons
func NewText(size float64, txt string, x, y int) *Text {
	fontFace := &text.GoTextFace{
		Source: fonts.MtgFont,
		Size:   size,
	}

	return &Text{text: txt, font: fontFace, Color: color.White, X: x, Y: y}
}

func (t *Text) Draw(screen *ebiten.Image, opts *ebiten.DrawImageOptions) {
	R, G, B, A := t.Color.RGBA()
	options := &text.DrawOptions{}
	options.GeoM.Concat(opts.GeoM)
	options.GeoM.Translate(float64(t.X), float64(t.Y))
	options.ColorScale.Scale(float32(R), float32(G), float32(B), float32(A))
	options.LineSpacing = 32.0

	text.Draw(screen, t.text, t.font, options)
}
