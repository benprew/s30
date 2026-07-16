package domain

import (
	"image"
	"testing"
)

func TestPointerMoveDirection(t *testing.T) {
	tests := []struct {
		name string
		pos  image.Point
		want int
	}{
		{name: "center dead zone", pos: image.Pt(512, 384)},
		{name: "right", pos: image.Pt(600, 384), want: DirRight},
		{name: "up left", pos: image.Pt(400, 300), want: DirUp | DirLeft},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := pointerMoveDirection(test.pos, 1024, 768); got != test.want {
				t.Fatalf("pointerMoveDirection() = %d, want %d", got, test.want)
			}
		})
	}
}
