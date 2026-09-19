package screens

import (
	"fmt"
	"image"
	"image/color"
	"sync"

	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/ui/elements"
	"github.com/benprew/s30/game/ui/imageutil"
	"github.com/hajimehoshi/ebiten/v2"
)

// dropToSellHint tells the player what the gold plate is for rather than leaving
// it to be discovered by dragging onto it. The wording is the port's own: the
// game's cue-card table covers the filter buttons only, so this sentence has no
// counterpart in the original's files.
const dropToSellHint = "Drop cards here to sell"

// goldLabel titles the player's gold the way the original does — @GOLDTITLE in
// the game's Menus.txt is "GOLD: %d".
func goldLabel(gold int) string {
	return fmt.Sprintf("GOLD: %d", gold)
}

// goldPlateRect is the part of the drop target the marble plate is drawn in. The
// plate's own art is about 4:1, so it is kept at those proportions inside the
// target rather than stretched to fill it; the band left below carries the hint.
func goldPlateRect(bounds image.Rectangle) image.Rectangle {
	height := bounds.Dx() / 4
	if height > bounds.Dy() {
		height = bounds.Dy()
	}
	return image.Rect(bounds.Min.X, bounds.Min.Y, bounds.Max.X, bounds.Min.Y+height)
}

var (
	goldPlateOnce sync.Once
	goldPlateImg  *ebiten.Image
)

// goldPlateImage loads the plate once. It is decoded on first use rather than
// every frame, and a failure returns nil so the caller falls back to a plain
// fill instead of drawing nothing.
func goldPlateImage() *ebiten.Image {
	goldPlateOnce.Do(func() {
		if img, err := imageutil.LoadImage(assets.EditDeckGoldPlate_png); err == nil {
			goldPlateImg = img
		}
	})
	return goldPlateImg
}

// drawDeckSellTarget draws the area that sells a card dropped on it. The original
// shows the player's gold in this corner, so the port puts the gold here and names
// the drop in the band beneath it, instead of a mask that only explains itself to
// someone who already thought to drag onto it.
func drawDeckSellTarget(screen *ebiten.Image, bounds image.Rectangle, gold int) {
	plate := goldPlateRect(bounds)

	if img := goldPlateImage(); img != nil {
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Scale(
			float64(plate.Dx())/float64(img.Bounds().Dx()),
			float64(plate.Dy())/float64(img.Bounds().Dy()),
		)
		opts.GeoM.Translate(float64(plate.Min.X), float64(plate.Min.Y))
		screen.DrawImage(img, opts)
	} else {
		fallback := ebiten.NewImage(plate.Dx(), plate.Dy())
		fallback.Fill(color.RGBA{R: 75, G: 35, B: 30, A: 240})
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(float64(plate.Min.X), float64(plate.Min.Y))
		screen.DrawImage(fallback, opts)
	}

	elements.NewText(18, goldLabel(gold), plate.Min.X+12, plate.Min.Y+8).
		Draw(screen, &ebiten.DrawImageOptions{}, 1)
	elements.NewText(12, dropToSellHint, bounds.Min.X+12, plate.Max.Y+4).
		Draw(screen, &ebiten.DrawImageOptions{}, 1)
}
