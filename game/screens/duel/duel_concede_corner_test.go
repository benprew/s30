package duel

import (
	"image"
	"testing"

	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/ui/elements"
	"github.com/benprew/s30/game/ui/imageutil"
)

// The game menu's square sits in the screen's corner on every screen, and the menu
// is drawn after the screen, so whatever it covers is lost. The duel's Concede
// button shares that corner and has to clear the square.
func TestConcedeButtonClearsTheGameMenuSquare(t *testing.T) {
	const W = 1024

	sheet, err := imageutil.LoadSpriteSheet(3, 1, assets.Tradbut1_png)
	if err != nil {
		t.Fatalf("sprite sheet: %v", err)
	}
	btn := elements.NewButton(sheet[0][0], sheet[0][1], sheet[0][2], 0, 0, 1.0)
	s := &DuelScreen{}
	s.positionTopRightButton(btn, W)

	menuSquare := image.Rect(W-12-44, 12, W-12, 56)
	if btn.Bounds.Overlaps(menuSquare) {
		t.Errorf("Concede button %v overlaps the game menu square %v", btn.Bounds, menuSquare)
	}
	if btn.Bounds.Max.X > W || btn.Bounds.Min.X < 0 {
		t.Errorf("Concede button %v is not on the screen", btn.Bounds)
	}
}
