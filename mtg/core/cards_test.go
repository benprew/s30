package core

import (
	"testing"

	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/mtg/effects"
)

func TestLoadCardDatabase(t *testing.T) {
	if len(domain.CARDS) == 0 {
		t.Errorf("CardDatabase is empty after loading")
	}

	forestCard := domain.FindCardByName("Forest")
	if forestCard == nil {
		t.Errorf("Card 'Forest' not found in database")
	}

	if forestCard.Name() != "Forest" {
		t.Errorf("Expected Forest name 'Forest', got '%s'", forestCard.Name())
	}

	if forestCard.CardType != domain.CardTypeLand {
		t.Errorf("Expected Forest card type '%s', got '%s'", domain.CardTypeLand, forestCard.CardType)
	}
}

func TestCardIsDead(t *testing.T) {
	card := &Card{}
	card.Toughness = 3

	card.DamageTaken = 0
	if card.IsDead() {
		t.Errorf("Card with 0 damage should not be dead")
	}

	card.DamageTaken = 2
	if card.IsDead() {
		t.Errorf("Card with damage < toughness should not be dead")
	}

	card.DamageTaken = 3
	if !card.IsDead() {
		t.Errorf("Card with damage == toughness should be dead")
	}

	card.DamageTaken = 5
	if !card.IsDead() {
		t.Errorf("Card with damage > toughness should be dead")
	}
}

func TestCardIsActive(t *testing.T) {
	card := &Card{}

	card.Tapped = false
	card.Active = true
	if !card.IsActive() {
		t.Errorf("Untapped and active card should be active")
	}

	card.Tapped = true
	card.Active = true
	if card.IsActive() {
		t.Errorf("Tapped card should not be active")
	}

	card.Tapped = false
	card.Active = false
	if card.IsActive() {
		t.Errorf("Inactive card should not be active")
	}

	card.Tapped = true
	card.Active = false
	if card.IsActive() {
		t.Errorf("Tapped and inactive card should not be active")
	}
}

func TestUnMarshalActionsDirectDamage(t *testing.T) {
	card := &Card{}
	card.Events = [][]string{{"DirectDamage", "3"}}

	card.UnMarshalActions()

	if len(card.Actions) != 1 {
		t.Errorf("Expected 1 action, got %d", len(card.Actions))
	}

	dd, ok := card.Actions[0].(*effects.DirectDamage)
	if !ok {
		t.Errorf("Expected DirectDamage action")
	}
	if dd.Amount != 3 {
		t.Errorf("Expected damage amount 3, got %d", dd.Amount)
	}
}

func TestUnMarshalActionsMultiple(t *testing.T) {
	card := &Card{}
	card.Events = [][]string{
		{"DirectDamage", "2"},
		{"DirectDamage", "5"},
	}

	card.UnMarshalActions()

	if len(card.Actions) != 2 {
		t.Errorf("Expected 2 actions, got %d", len(card.Actions))
	}

	dd1 := card.Actions[0].(*effects.DirectDamage)
	dd2 := card.Actions[1].(*effects.DirectDamage)

	if dd1.Amount != 2 {
		t.Errorf("Expected first damage amount 2, got %d", dd1.Amount)
	}
	if dd2.Amount != 5 {
		t.Errorf("Expected second damage amount 5, got %d", dd2.Amount)
	}
}

func TestUnMarshalActionsEmpty(t *testing.T) {
	card := &Card{}
	card.Events = [][]string{}

	card.UnMarshalActions()

	if len(card.Actions) != 0 {
		t.Errorf("Expected 0 actions, got %d", len(card.Actions))
	}
}

func TestCardReceiveDamage(t *testing.T) {
	card := &Card{}
	card.DamageTaken = 0

	card.ReceiveDamage(3)
	if card.DamageTaken != 3 {
		t.Errorf("Expected damage taken 3, got %d", card.DamageTaken)
	}

	card.ReceiveDamage(2)
	if card.DamageTaken != 5 {
		t.Errorf("Expected damage taken 5, got %d", card.DamageTaken)
	}
}
