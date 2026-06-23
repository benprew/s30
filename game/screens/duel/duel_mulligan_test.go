package duel

import (
	"image"
	"testing"
)

func TestRectIndexAtPoint(t *testing.T) {
	rects := []image.Rectangle{
		image.Rect(10, 20, 30, 50),
		image.Rect(40, 20, 60, 50),
	}

	tests := []struct {
		name string
		x    int
		y    int
		want int
	}{
		{name: "first card", x: 10, y: 20, want: 0},
		{name: "second card", x: 59, y: 49, want: 1},
		{name: "right edge excluded", x: 60, y: 49, want: -1},
		{name: "gap", x: 35, y: 30, want: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := rectIndexAtPoint(rects, tt.x, tt.y); got != tt.want {
				t.Fatalf("rectIndexAtPoint() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestMulliganPreviewPositionUsesOppositeSide(t *testing.T) {
	const screenW, screenH = 1024, 768
	previewBounds := image.Rect(0, 0, 245, 342)

	leftCard := image.Rect(50, 300, 170, 468)
	if got := mulliganPreviewPosition(screenW, screenH, leftCard, previewBounds); got.X != 759 {
		t.Fatalf("preview for left card X = %d, want 759", got.X)
	}

	rightCard := image.Rect(850, 300, 970, 468)
	if got := mulliganPreviewPosition(screenW, screenH, rightCard, previewBounds); got.X != 20 {
		t.Fatalf("preview for right card X = %d, want 20", got.X)
	}
}

func TestMulliganPreviewPositionStaysOnScreen(t *testing.T) {
	card := image.Rect(5, 40, 25, 68)
	previewBounds := image.Rect(0, 0, 245, 342)

	got := mulliganPreviewPosition(200, 180, card, previewBounds)
	if got.X != 0 || got.Y != 0 {
		t.Fatalf("preview position = %v, want (0,0)", got)
	}
}
