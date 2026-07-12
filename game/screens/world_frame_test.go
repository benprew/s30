package screens

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestWorldFrameQuestScrollClickedUsesScaledBounds(t *testing.T) {
	frame := &WorldFrame{questScrollEmpty: ebiten.NewImage(100, 50)}
	want := image.Rect(1630, 1076, 1790, 1156)

	clicked := frame.questScrollClicked(2, func(bounds image.Rectangle) bool {
		if bounds != want {
			t.Fatalf("quest scroll bounds = %v, want %v", bounds, want)
		}
		return true
	})

	if !clicked {
		t.Fatal("quest scroll click was not reported")
	}
}
