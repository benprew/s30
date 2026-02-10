package core

import (
	"testing"

	"github.com/benprew/s30/game/domain"
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

func TestCardIsActive_HasteBypassesSummoningSickness(t *testing.T) {
	card := &Card{}
	card.Keywords = []string{"Haste"}

	card.Tapped = false
	card.Active = false
	if !card.IsActive() {
		t.Errorf("Untapped creature with haste should be active even when summoning sick")
	}

	card.Tapped = true
	card.Active = false
	if card.IsActive() {
		t.Errorf("Tapped creature with haste should not be active")
	}
}

func TestCardActionsFromParsedAbilities(t *testing.T) {
	card := &Card{}
	card.ParsedAbilities = []domain.ParsedAbility{
		{
			Type: "Spell",
			Effect: &domain.ParsedEffect{
				Amount: 3,
			},
		},
	}

	actions := card.CardActions()
	if len(actions) != 1 {
		t.Errorf("Expected 1 action, got %d", len(actions))
	}
}

func TestActivatedAbilitiesNotTriggeredOnCast(t *testing.T) {
	card := &Card{}
	card.ParsedAbilities = []domain.ParsedAbility{
		{
			Type: "Activated",
			Effect: &domain.ParsedEffect{
				PowerBoost: 1,
			},
		},
	}

	actions := card.CardActions()
	if len(actions) != 0 {
		t.Errorf("Activated abilities should not trigger on cast, got %d actions", len(actions))
	}
}

func TestCardActionsStatBoost(t *testing.T) {
	card := &Card{}
	card.ParsedAbilities = []domain.ParsedAbility{
		{
			Type: "Static",
			Effect: &domain.ParsedEffect{
				PowerBoost:     2,
				ToughnessBoost: 2,
			},
		},
	}

	actions := card.CardActions()
	if len(actions) != 1 {
		t.Errorf("Expected 1 action, got %d", len(actions))
	}
}

func TestCardActionsEmpty(t *testing.T) {
	card := &Card{}
	card.ParsedAbilities = nil

	actions := card.CardActions()
	if len(actions) != 0 {
		t.Errorf("Expected 0 actions, got %d", len(actions))
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
