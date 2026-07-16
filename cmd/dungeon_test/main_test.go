package main

import (
	"math/rand"
	"testing"

	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestSelectRestrictedCardsChoosesOneToFourUniqueRestrictedCards(t *testing.T) {
	cards := []*domain.Card{
		{CardName: "Black Lotus", VintageRestricted: true},
		{CardName: "Black Lotus", VintageRestricted: true},
		{CardName: "Lightning Bolt"},
		{CardName: "Mox Emerald", VintageRestricted: true},
		{CardName: "Mox Jet", VintageRestricted: true},
		{CardName: "Mox Pearl", VintageRestricted: true},
		{CardName: "Mox Ruby", VintageRestricted: true},
		{CardName: "Mox Sapphire", VintageRestricted: true},
	}

	selected := selectRestrictedCards(cards, rand.New(rand.NewSource(1)))
	if len(selected) < 1 || len(selected) > 4 {
		t.Fatalf("selected %d cards, want 1-4", len(selected))
	}

	seen := make(map[string]bool)
	for _, card := range selected {
		if !card.VintageRestricted {
			t.Fatalf("selected unrestricted card %q", card.CardName)
		}
		if seen[card.CardName] {
			t.Fatalf("selected duplicate card %q", card.CardName)
		}
		seen[card.CardName] = true
	}
}

func TestSelectRestrictedCardsReturnsEmptyWhenPoolHasNoRestrictedCards(t *testing.T) {
	selected := selectRestrictedCards([]*domain.Card{
		{CardName: "Lightning Bolt"},
		{CardName: "Serra Angel"},
	}, rand.New(rand.NewSource(1)))

	if len(selected) != 0 {
		t.Fatalf("selected %d cards, want none", len(selected))
	}
}

type pointerAwareScreen struct {
	t              *testing.T
	pointerUpdated *bool
}

func (s *pointerAwareScreen) Update(_, _ int, _ float64) (screenui.ScreenName, screenui.Screen, error) {
	s.t.Helper()
	if !*s.pointerUpdated {
		s.t.Fatal("screen updated before pointer input was sampled")
	}
	return screenui.DungeonScr, nil, nil
}

func (s *pointerAwareScreen) Draw(*ebiten.Image, int, int, float64) {}
func (s *pointerAwareScreen) IsFramed() bool                        { return false }
func (s *pointerAwareScreen) IsOverlay() bool                       { return false }

func TestGameUpdatesPointerBeforeCurrentScreen(t *testing.T) {
	pointerUpdated := false
	g := &testGame{
		screens: map[screenui.ScreenName]screenui.Screen{
			screenui.DungeonScr: &pointerAwareScreen{t: t, pointerUpdated: &pointerUpdated},
		},
		current:       screenui.DungeonScr,
		updatePointer: func() { pointerUpdated = true },
	}

	if err := g.Update(); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
}
