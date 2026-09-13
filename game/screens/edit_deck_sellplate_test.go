package screens

import (
	"image"
	"testing"
)

func TestGoldLabelMatchesTheOriginalTitle(t *testing.T) {
	cases := []struct {
		gold int
		want string
	}{
		{0, "GOLD: 0"},
		{9, "GOLD: 9"},
		{250, "GOLD: 250"},
	}
	for _, c := range cases {
		if got := goldLabel(c.gold); got != c.want {
			t.Errorf("goldLabel(%d) = %q, want %q", c.gold, got, c.want)
		}
	}
}

// The plate is drawn inside the drop target, never outside it: what the player
// sees has to be what accepts the card.
func TestGoldPlateStaysInsideTheDropTarget(t *testing.T) {
	target := image.Rect(10, 12, 150, 68)
	plate := goldPlateRect(target)
	if !plate.In(target) {
		t.Errorf("plate %v is not inside the target %v", plate, target)
	}
	if plate.Dx() != target.Dx() {
		t.Errorf("plate width = %d, want the target's %d", plate.Dx(), target.Dx())
	}
	if got, want := plate.Dy()*4, plate.Dx(); got != want {
		t.Errorf("plate is %dx%d, which is not the marble art's 4:1", plate.Dx(), plate.Dy())
	}
	if room := target.Dy() - plate.Dy(); room < 16 {
		t.Errorf("only %d px left under the plate for the hint", room)
	}
}

// The plate comes from the embedded asset. A silent failure there would leave the
// corner on the plain fallback fill with nothing saying so, which is exactly the
// kind of thing that survives until someone looks at the screen.
func TestGoldPlateAssetLoads(t *testing.T) {
	img := goldPlateImage()
	if img == nil {
		t.Fatal("the embedded gold plate did not load")
	}
	if b := img.Bounds(); b.Dx() == 0 || b.Dy() == 0 {
		t.Fatalf("plate has no size: %v", b)
	}
}
