package elements

import (
	"image/color"

	"github.com/benprew/s30/game/ui/fonts"
	"github.com/benprew/s30/game/ui/layout"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// TODO: TextBox should be a Rectangle and text should
// flow into textbox size
type Text struct {
	text  string
	font  text.Face
	Color color.Color
	color color.Color

	// Legacy positioning (deprecated, use Position instead)
	X int
	Y int

	// New anchor-based positioning
	Position *layout.Position // If nil, falls back to X, Y
}

// For drawing text, similar to buttons
func NewText(size float64, txt string, x, y int) *Text {
	fontFace := &text.GoTextFace{
		Source: fonts.MtgFont,
		Size:   size,
	}

	return &Text{text: txt, font: fontFace, Color: color.White, X: x, Y: y}
}

func (t *Text) Draw(screen *ebiten.Image, opts *ebiten.DrawImageOptions, scale float64) {
	R, G, B, A := t.Color.RGBA()
	// Calculate position (anchor-based or legacy X,Y)
	x, y := t.getPosition(screen, scale)
	options := &text.DrawOptions{}

	options.GeoM.Concat(opts.GeoM)
	options.GeoM.Translate(float64(x), float64(y))
	options.ColorScale.Scale(float32(R), float32(G), float32(B), float32(A))
	options.LineSpacing = 32.0

	text.Draw(screen, t.text, t.font, options)
}

// getPosition returns the text's X, Y position based on anchor or legacy positioning
func (t *Text) getPosition(screen *ebiten.Image, scale float64) (int, int) {
	if t.Position != nil {
		w, h := layout.GetBounds(screen)
		return t.Position.Resolve(int(float64(w)/scale), int(float64(h)/scale))
	}
	// Fallback to legacy X, Y positioning
	return t.X, t.Y
}
