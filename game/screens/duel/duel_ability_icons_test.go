package duel

import (
	"image"
	"slices"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
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

// Protection shows as a shield whether it is printed on the card or granted by
// another permanent, and Artifact Ward gets the brown artifact shield.
func TestGetKeywordIconsShowsProtectionShields(t *testing.T) {
	const blackShield, artifactShield = 8, 10
	tests := []struct {
		name     string
		setup    func(g *gametest.TestGame)
		creature string
		want     int
	}{
		{
			name: "printed protection",
			setup: func(g *gametest.TestGame) {
				g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "White Knight")
			},
			creature: "White Knight",
			want:     blackShield,
		},
		{
			name: "protection granted by Black Ward",
			setup: func(g *gametest.TestGame) {
				g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
				g.AddCard(core.ZoneHand, gametest.PlayerA, "Black Ward")
				g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Black Ward", "Grizzly Bears")
			},
			creature: "Grizzly Bears",
			want:     blackShield,
		},
		{
			name: "Artifact Ward",
			setup: func(g *gametest.TestGame) {
				g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
				g.AddCard(core.ZoneHand, gametest.PlayerA, "Artifact Ward")
				g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Artifact Ward", "Grizzly Bears")
			},
			creature: "Grizzly Bears",
			want:     artifactShield,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			tc.setup(g)
			g.StopAt(1, core.BeginCombat)
			g.Execute()

			perm := g.FindPermanentByName(tc.creature, g.GetPlayer(gametest.PlayerA).PlayerID())
			if perm == nil {
				t.Fatalf("%s not found", tc.creature)
			}
			s := &DuelScreen{game: g.Game, abilityIcons: make([]*ebiten.Image, 18)}
			for i := range s.abilityIcons {
				s.abilityIcons[i] = &ebiten.Image{}
			}

			icons := s.getKeywordIcons(interactive.PermanentState{ID: perm.ID()})
			if !slices.Contains(icons, s.abilityIcons[tc.want]) {
				t.Fatalf("%s shows %d icons, missing shield %d", tc.creature, len(icons), tc.want)
			}
		})
	}
}
