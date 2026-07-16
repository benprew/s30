package screens

import (
	"image"
	"testing"
)

func TestDungeonPointerDirection(t *testing.T) {
	origin := image.Pt(400, 300)
	tests := []struct {
		name    string
		point   image.Point
		clicked bool
		dx, dy  int
	}{
		{name: "not clicked", point: image.Pt(500, 300)},
		{name: "dead zone", point: image.Pt(410, 300), clicked: true},
		{name: "up left", point: image.Pt(368, 284), clicked: true, dx: -1},
		{name: "up right", point: image.Pt(432, 284), clicked: true, dy: -1},
		{name: "down right", point: image.Pt(432, 316), clicked: true, dx: 1},
		{name: "down left", point: image.Pt(368, 316), clicked: true, dy: 1},
		{name: "far right resolves toward down right", point: image.Pt(500, 300), clicked: true, dx: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dx, dy := dungeonPointerDirection(test.point, origin, test.clicked)
			if dx != test.dx || dy != test.dy {
				t.Fatalf("direction = (%d, %d), want (%d, %d)", dx, dy, test.dx, test.dy)
			}
		})
	}
}
