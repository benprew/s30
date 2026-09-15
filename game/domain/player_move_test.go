package domain

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestMoveDirectionBitsSupportsArrowsAndWASD(t *testing.T) {
	cases := []struct {
		name     string
		pressed  map[ebiten.Key]bool
		wantBits int
	}{
		{
			name: "arrows",
			pressed: map[ebiten.Key]bool{
				ebiten.KeyLeft:  true,
				ebiten.KeyUp:    true,
				ebiten.KeyRight: true,
				ebiten.KeyDown:  true,
			},
			wantBits: DirLeft | DirUp | DirRight | DirDown,
		},
		{
			name: "wasd",
			pressed: map[ebiten.Key]bool{
				ebiten.KeyA: true,
				ebiten.KeyW: true,
				ebiten.KeyD: true,
				ebiten.KeyS: true,
			},
			wantBits: DirLeft | DirUp | DirRight | DirDown,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := moveDirectionBits(func(key ebiten.Key) bool {
				return tc.pressed[key]
			})
			if got != tc.wantBits {
				t.Fatalf("moveDirectionBits() = %d, want %d", got, tc.wantBits)
			}
		})
	}
}
