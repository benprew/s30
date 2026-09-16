package screens

import (
	"image"
	"image/color"

	"github.com/benprew/s30/game/ui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// The world frame's menu button: three dots in a round button, in the frame's
// upper right corner. It is sized for touch, because the menu behind it is the
// one the original only reached with a keyboard.
//
// The corner comes from the frame's own art (Advinter1024.pic.png): its playable
// area starts at x=922 and its top strip ends at y=66. Insetting from the screen
// instead put the button outside the frame, on the border itself.
const (
	worldMenuButtonSize  = 44
	worldMenuButtonRight = 922
	worldMenuButtonTop   = 66
	worldMenuButtonInset = 10
	worldMenuDotRadius   = 3.5
	worldMenuDotSpacing  = 12
)

// worldMenuButtonBounds is the button's rectangle, inside the frame's corner.
func worldMenuButtonBounds() image.Rectangle {
	x := worldMenuButtonRight - worldMenuButtonInset - worldMenuButtonSize
	y := worldMenuButtonTop + worldMenuButtonInset
	return image.Rect(x, y, x+worldMenuButtonSize, y+worldMenuButtonSize)
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
	// Round rather than a box: the button sits on the map's frame, where a square
	// corner reads as something the game drew by accident. Ebiten's vector
	// package has no rounded rectangle, so the shape is a circle.
	vector.FillCircle(screen, x+size/2, y+size/2, size/2, bg, false)

	dot := color.RGBA{R: 235, G: 231, B: 219, A: 255}
	radius := float32(worldMenuDotRadius) * f
	spacing := float32(worldMenuDotSpacing) * f
	cx := x + size/2
	cy := y + size/2
	for i := -1; i <= 1; i++ {
		vector.FillCircle(screen, cx, cy+float32(i)*spacing, radius, dot, false)
	}
}
