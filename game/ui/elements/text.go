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
	Text  string
	font  text.Face
	Color color.Color

	// Legacy positioning (deprecated, use Position instead)
	X int
	Y int

	// New anchor-based positioning
	Position *layout.Position // If nil, falls back to X, Y

	// Alignment within a bounding rectangle starting at X, Y
	HAlign  HorizontalAlignment
	VAlign  VerticalAlignment
	BoundsW float64
	BoundsH float64

	LineSpacing float64
}

// For drawing text, similar to buttons
func NewText(size float64, txt string, x, y int) *Text {
	fontFace := &text.GoTextFace{
		Source: fonts.MtgFont,
		Size:   size,
	}

	return &Text{Text: txt, font: fontFace, Color: color.White, X: x, Y: y}
}

func (t *Text) Draw(screen *ebiten.Image, opts *ebiten.DrawImageOptions, scale float64) {
	R, G, B, A := t.Color.RGBA()
	x, y := t.getPosition(screen, scale)

	lineSpacing := 32.0
	if t.LineSpacing != 0 {
		lineSpacing = t.LineSpacing
	}

	shadow := &text.DrawOptions{}
	shadow.GeoM.Concat(opts.GeoM)
	shadow.GeoM.Translate(float64(x)+1, float64(y)+2)
	shadow.ColorScale.Scale(0, 0, 0, float32(A)/65535)
	shadow.LineSpacing = lineSpacing
	text.Draw(screen, t.Text, t.font, shadow)

	options := &text.DrawOptions{}
	options.GeoM.Concat(opts.GeoM)
	options.GeoM.Translate(float64(x), float64(y))
	// Normalize 16-bit RGBA to 0.0-1.0 so semi-transparent anti-aliased edge pixels
	// aren't blown out to fully opaque (which makes text look blurry)
	options.ColorScale.Scale(float32(R)/65535, float32(G)/65535, float32(B)/65535, float32(A)/65535)
	options.LineSpacing = lineSpacing
	text.Draw(screen, t.Text, t.font, options)
}

func (t *Text) Measure() (float64, float64) {
	lineSpacing := 32.0
	if t.LineSpacing != 0 {
		lineSpacing = t.LineSpacing
	}
	return text.Measure(t.Text, t.font, lineSpacing)
}

// getPosition returns the text's X, Y position based on anchor or legacy positioning
func (t *Text) getPosition(screen *ebiten.Image, scale float64) (int, int) {
	if t.Position != nil {
		w, h := layout.GetBounds(screen)
		return t.Position.Resolve(int(float64(w)/scale), int(float64(h)/scale))
	}
	x, y := float64(t.X), float64(t.Y)
	if t.BoundsW > 0 || t.BoundsH > 0 {
		txtW, txtH := text.Measure(t.Text, t.font, 0)
		switch t.HAlign {
		case AlignCenter:
			x += (t.BoundsW - txtW) / 2
		case AlignRight:
			x += t.BoundsW - txtW
		}
		switch t.VAlign {
		case AlignMiddle:
			y += (t.BoundsH - txtH) / 2
		case AlignBottom:
			y += t.BoundsH - txtH
		}
	}
	return int(x), int(y)
}
