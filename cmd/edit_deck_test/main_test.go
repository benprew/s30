package main

import (
	"errors"
	"testing"

	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestNewTestCollectionIncludesDeckAndCollectionCards(t *testing.T) {
	collection, err := newTestCollection([]cardSpec{
		{name: "Lightning Bolt", total: 7, inDeck: 4},
		{name: "Forest", total: 8, inDeck: 8},
	})
	if err != nil {
		t.Fatal(err)
	}

	bolt := domain.FindCardByName("Lightning Bolt")
	if got := collection.GetTotalCount(bolt); got != 7 {
		t.Fatalf("Lightning Bolt total = %d, want 7", got)
	}
	if got := collection.GetDeckCount(bolt, 0); got != 4 {
		t.Fatalf("Lightning Bolt deck count = %d, want 4", got)
	}

	forest := domain.FindCardByName("Forest")
	if got := collection.GetTotalCount(forest); got != 8 {
		t.Fatalf("Forest total = %d, want 8", got)
	}
	if got := collection.GetDeckCount(forest, 0); got != 8 {
		t.Fatalf("Forest deck count = %d, want 8", got)
	}
}

func TestNewTestCollectionRejectsUnknownCard(t *testing.T) {
	_, err := newTestCollection([]cardSpec{{name: "Definitely Not A Card", total: 1}})
	if err == nil {
		t.Fatal("newTestCollection() error = nil, want unknown-card error")
	}
}

type exitScreen struct {
	pointerUpdated *bool
}

func (s *exitScreen) Update(_, _ int, _ float64) (screenui.ScreenName, screenui.Screen, error) {
	if !*s.pointerUpdated {
		return screenui.NoScr, nil, errors.New("screen updated before pointer input")
	}
	return screenui.CityScr, nil, nil
}

func (s *exitScreen) Draw(*ebiten.Image, int, int, float64) {}
func (s *exitScreen) IsFramed() bool                        { return false }
func (s *exitScreen) IsOverlay() bool                       { return false }

func TestGameTerminatesWhenEditDeckReturnsToCity(t *testing.T) {
	pointerUpdated := false
	g := &testGame{
		screen:        &exitScreen{pointerUpdated: &pointerUpdated},
		updatePointer: func() { pointerUpdated = true },
	}

	if err := g.Update(); err != ebiten.Termination {
		t.Fatalf("Update() error = %v, want ebiten.Termination", err)
	}
}
