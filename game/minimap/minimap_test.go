package minimap

import (
	"image"
	"testing"

	"github.com/benprew/s30/game/ui/elements"
)

func TestMiniMapButtonsArePreScaled(t *testing.T) {
	miniMap := NewMiniMap(nil)

	for _, button := range miniMap.buttons {
		if got, want := button.Bounds.Dx(), 148; got != want {
			t.Fatalf("button width = %d; want %d", got, want)
		}
		if got, want := button.Bounds.Dy(), 43; got != want {
			t.Fatalf("button height = %d; want %d", got, want)
		}
	}
}

func TestMiniMapArtworkIsPreScaled(t *testing.T) {
	miniMap := NewMiniMap(nil)

	if got, want := miniMap.frame.Bounds().Size(), image.Pt(1024, 768); got != want {
		t.Fatalf("frame size = %v; want %v", got, want)
	}
	if got, want := miniMap.terrainSprite[0][0].Bounds().Size(), image.Pt(20, 20); got != want {
		t.Fatalf("terrain sprite size = %v; want %v", got, want)
	}
}

func TestMiniMapTextIsPreScaledAndCentered(t *testing.T) {
	miniMap := NewMiniMap(nil)

	if got, want := miniMap.fontFace.Size, 14*SCALE; got != want {
		t.Fatalf("font size = %v; want %v", got, want)
	}
	for _, button := range miniMap.buttons {
		if got, want := button.ButtonText.VAlign, elements.AlignMiddle; got != want {
			t.Fatalf("button vertical alignment = %v; want %v", got, want)
		}
	}
}
