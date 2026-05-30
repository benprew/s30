package duel

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/google/uuid"
)

func TestDuelEscapeWithoutOverlayStaysInDuel(t *testing.T) {
	s := &DuelScreen{}

	s.handleEscape()

	if s.viewingGraveyard != nil {
		t.Fatal("escape should not open graveyard view")
	}
	if s.isChoosingAbility() {
		t.Fatal("escape should not enter ability choosing mode")
	}
	if s.isChoosingX() {
		t.Fatal("escape should not enter X choosing mode")
	}
	if s.targetingCardID != uuid.Nil {
		t.Fatal("escape should not enter targeting mode")
	}
}

func TestDuelEscapeClosesActiveOverlay(t *testing.T) {
	t.Run("graveyard", func(t *testing.T) {
		s := &DuelScreen{viewingGraveyard: &duelPlayer{name: "You"}}
		s.handleEscape()
		if s.viewingGraveyard != nil {
			t.Fatal("escape should close graveyard view")
		}
	})

	t.Run("ability chooser", func(t *testing.T) {
		s := &DuelScreen{abilityChoosingActions: []interactive.ActionOption{{CardName: "Test"}}}
		s.handleEscape()
		if s.isChoosingAbility() {
			t.Fatal("escape should close ability chooser")
		}
	})

	t.Run("x chooser", func(t *testing.T) {
		s := &DuelScreen{xChoosingActions: []interactive.ActionOption{{CardName: "Fireball"}}}
		s.handleEscape()
		if s.isChoosingX() {
			t.Fatal("escape should close X chooser")
		}
	})

	t.Run("selected target", func(t *testing.T) {
		targetingID := uuid.New()
		s := &DuelScreen{
			targetingCardID:  targetingID,
			targetingActions: map[uuid.UUID]interactive.ActionOption{uuid.New(): {}},
			selectedTargetID: uuid.New(),
		}
		s.handleEscape()
		if s.targetingCardID != targetingID {
			t.Fatal("escape should keep targeting mode when only clearing the selected target")
		}
		if s.selectedTargetID != uuid.Nil {
			t.Fatal("escape should clear selected target")
		}
	})

	t.Run("targeting", func(t *testing.T) {
		s := &DuelScreen{
			targetingCardID:  uuid.New(),
			targetingActions: map[uuid.UUID]interactive.ActionOption{uuid.New(): {}},
		}
		s.handleEscape()
		if s.targetingCardID != uuid.Nil {
			t.Fatal("escape should close targeting mode")
		}
		if s.targetingActions != nil {
			t.Fatal("escape should clear targeting actions")
		}
	})
}
