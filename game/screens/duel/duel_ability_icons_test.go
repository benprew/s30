package duel

import (
	"image"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestDrawAbilityIconsShownForTappedAndUntappedPermanents(t *testing.T) {
	icon := &ebiten.Image{}
	s := &DuelScreen{abilityIcons: make([]*ebiten.Image, 18)}
	s.abilityIcons[11] = icon
	pos := image.Pt(10, 20)

	for _, tapped := range []bool{false, true} {
		t.Run(map[bool]string{false: "untapped", true: "tapped"}[tapped], func(t *testing.T) {
			perm := interactive.PermanentState{Tapped: tapped, Keywords: []string{"Flying"}}

			got := s.abilityIconPlacements(perm, pos)

			if len(got) != 1 {
				t.Fatalf("ability icon placements = %d, want 1", len(got))
			}
			if got[0].icon != icon {
				t.Fatal("ability icon placement has the wrong image")
			}
			if got[0].pos != image.Pt(pos.X, pos.Y+fieldCardH-22) {
				t.Fatalf("ability icon position = %v, want %v", got[0].pos, image.Pt(pos.X, pos.Y+fieldCardH-22))
			}
		})
	}
}
