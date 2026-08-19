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
func TestGetKeywordIconsDeduplication(t *testing.T) {
	s := &DuelScreen{abilityIcons: make([]*ebiten.Image, 18)}
	for i := range s.abilityIcons {
		s.abilityIcons[i] = &ebiten.Image{}
	}

	perm := interactive.PermanentState{
		Keywords: []string{"Flying", "Flying", "Trample", "First Strike"},
	}

	icons := s.getKeywordIcons(perm)
	if len(icons) != 3 {
		t.Fatalf("expected 3 icons, got %d", len(icons))
	}
	if icons[0] != s.abilityIcons[11] || icons[1] != s.abilityIcons[12] || icons[2] != s.abilityIcons[14] {
		t.Fatal("icons did not match expected order")
	}
}
