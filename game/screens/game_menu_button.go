package screens

import (
	"fmt"
	"image"
	"image/color"

	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/ui"
	"github.com/benprew/s30/game/ui/imageutil"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// The game menu button: menu-square icon in the screen's upper right corner.
const (
	gameMenuButtonSize  = 44
	gameMenuButtonTop   = 12
	gameMenuButtonInset = 12
)

// gameMenuButtonBounds is the button's rectangle, in the screen's upper right corner.
func gameMenuButtonBounds(w ...int) image.Rectangle {
	W := 1024
	if len(w) > 0 {
		W = w[0]
	}
	x := W - gameMenuButtonInset - gameMenuButtonSize
	return image.Rect(x, gameMenuButtonTop, x+gameMenuButtonSize, gameMenuButtonTop+gameMenuButtonSize)
}

func worldMenuButtonBounds(w ...int) image.Rectangle {
	return gameMenuButtonBounds(w...)
}

// gameMenuOpens reports whether the menu was asked for, by clicking the button
// or with the key the original used. Kept pure so both paths are testable.
func gameMenuOpens(clicked, escape bool) bool {
	return clicked || escape
}

func worldMenuOpens(clicked, escape bool) bool {
	return gameMenuOpens(clicked, escape)
}

var menuSquareImage *ebiten.Image

func getMenuSquareImage() *ebiten.Image {
	if menuSquareImage == nil {
		img, err := imageutil.LoadImage(assets.MenuSquare_png)
		if err != nil {
			panic(fmt.Sprintf("failed to load menu square image: %v", err))
		}
		menuSquareImage = img
	}
	return menuSquareImage
}

// drawGameMenuButton draws the menu-square button in the upper-right corner.
// It carries its own background so the icon stays readable over any terrain,
// and lightens under the cursor the way the screen's other controls do.
func drawGameMenuButton(screen *ebiten.Image, bounds image.Rectangle, scale float64) {
	f := float32(scale)
	x := float32(bounds.Min.X) * f
	y := float32(bounds.Min.Y) * f
	size := float32(bounds.Dx()) * f

	bg := color.RGBA{R: 24, G: 20, B: 16, A: 200}
	iconColor := color.RGBA{R: 235, G: 231, B: 219, A: 255}
	if ui.Position().In(bounds) {
		bg = color.RGBA{R: 62, G: 54, B: 42, A: 230}
		iconColor = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	}

	corner := float32(6) * f
	vector.FillRect(screen, x+corner, y, size-2*corner, size, bg, false)
	vector.FillRect(screen, x, y+corner, size, size-2*corner, bg, false)
	vector.FillCircle(screen, x+corner, y+corner, corner, bg, false)
	vector.FillCircle(screen, x+size-corner, y+corner, corner, bg, false)
	vector.FillCircle(screen, x+corner, y+size-corner, corner, bg, false)
	vector.FillCircle(screen, x+size-corner, y+size-corner, corner, bg, false)

	icon := getMenuSquareImage()
	iconW := float64(icon.Bounds().Dx())
	iconH := float64(icon.Bounds().Dy())
	ix := float64(bounds.Min.X) + (float64(bounds.Dx())-iconW)/2
	iy := float64(bounds.Min.Y) + (float64(bounds.Dy())-iconH)/2

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Scale(scale, scale)
	opts.GeoM.Translate(ix*scale, iy*scale)
	opts.ColorScale.ScaleWithColor(iconColor)
	screen.DrawImage(icon, opts)
}
