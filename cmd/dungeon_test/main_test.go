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

func TestDungeonOptionsWizardModeUsesCastleBossAndNoRestrictedCards(t *testing.T) {
	for _, color := range []domain.ColorMask{
		domain.ColorWhite, domain.ColorBlue, domain.ColorBlack, domain.ColorRed, domain.ColorGreen,
	} {
		restricted := []*domain.Card{{CardName: "Black Lotus", VintageRestricted: true}}
		opts, err := dungeonOptions(color, domain.DungeonDifficultyEasy, true, restricted, nil, 1)
		if err != nil {
			t.Fatalf("%s: %v", domain.ColorMaskToString(color), err)
		}
		if opts.FinalEnemy == nil {
			t.Errorf("%s: missing wizard", domain.ColorMaskToString(color))
		}
		if opts.Difficulty != domain.DungeonDifficultyHard {
			t.Errorf("%s: difficulty = %d, want hard", domain.ColorMaskToString(color), opts.Difficulty)
		}
		if len(opts.RestrictedCards) != 0 {
			t.Errorf("%s: wizard dungeon has restricted cards", domain.ColorMaskToString(color))
		}
		if opts.NumGoldChests != 2 {
			t.Errorf("%s: wizard dungeon does not use castle contents", domain.ColorMaskToString(color))
		}
	}
}

func TestDungeonOptionsNormalModePreservesRestrictedCards(t *testing.T) {
	restricted := []*domain.Card{{CardName: "Black Lotus", VintageRestricted: true}}
	opts, err := dungeonOptions(domain.ColorRed, domain.DungeonDifficultyMedium, false, restricted, nil, 1)
	if err != nil {
		t.Fatal(err)
	}
	if opts.FinalEnemy != nil || len(opts.RestrictedCards) != 1 || opts.Difficulty != domain.DungeonDifficultyMedium {
		t.Fatal("normal dungeon options changed")
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
