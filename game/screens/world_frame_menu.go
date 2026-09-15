package screens

import (
	"image"
	"image/color"

	"github.com/benprew/s30/game/ui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// The world frame's menu button: three dots in the screen's upper right corner,
// clear of the frame itself (which starts at x=100, y=75) and of the sidebar
// buttons down the left edge. It is sized for touch, because the menu behind it
// is the one the original only reached with a keyboard.
const (
	worldMenuButtonSize  = 44
	worldMenuButtonY     = 16
	worldMenuButtonInset = 30 // distance from the screen's right edge
	worldMenuDotRadius   = 3.5
	worldMenuDotSpacing  = 12
)

// worldMenuButtonBounds is the button's rectangle for a screen of width W.
func worldMenuButtonBounds(W int) image.Rectangle {
	x := W - worldMenuButtonInset - worldMenuButtonSize
	return image.Rect(x, worldMenuButtonY, x+worldMenuButtonSize, worldMenuButtonY+worldMenuButtonSize)
}

// worldMenuOpens reports whether the menu was asked for, by clicking the button
// or with the key the original used. Kept pure so both paths are testable.
func worldMenuOpens(clicked, escape bool) bool {
	return clicked || escape
}

// drawWorldMenuButton draws the three dots over the world. It carries its own
// background so the dots stay readable over any terrain, and lightens under the
// cursor the way the screen's other controls do.
func drawWorldMenuButton(screen *ebiten.Image, bounds image.Rectangle, scale float64) {
	f := float32(scale)
	x := float32(bounds.Min.X) * f
	y := float32(bounds.Min.Y) * f
	size := float32(bounds.Dx()) * f

	bg := color.RGBA{R: 24, G: 20, B: 16, A: 200}
	if ui.Position().In(bounds) {
		bg = color.RGBA{R: 62, G: 54, B: 42, A: 230}
	}
	vector.FillRect(screen, x, y, size, size, bg, false)

	dot := color.RGBA{R: 235, G: 231, B: 219, A: 255}
	radius := float32(worldMenuDotRadius) * f
	spacing := float32(worldMenuDotSpacing) * f
	cx := x + size/2
	cy := y + size/2
	for i := -1; i <= 1; i++ {
		vector.FillCircle(screen, cx, cy+float32(i)*spacing, radius, dot, false)
	}
}
