package core

import (
	"testing"

	"github.com/benprew/s30/game/domain"
)

func TestGiantGrowthTargetsCreaturesOnly(t *testing.T) {
	domainCard := domain.FindCardByName("Giant Growth")
	if domainCard == nil {
		t.Fatal("Giant Growth card not found")
	}

	card := NewCardFromDomain(domainCard, 1, nil)
	if !card.TargetsCreaturesOnly() {
		t.Errorf("Giant Growth should return true for TargetsCreaturesOnly()")
	}
}

func TestAddCounters(t *testing.T) {
	c := makeCreature(1, "Test", 2, 3, nil)
	c.AddCounters(CounterPlusOnePlusOne, 2)
	if c.EffectivePower() != 4 {
		t.Errorf("expected power 4, got %d", c.EffectivePower())
	}
	if c.EffectiveToughness() != 5 {
		t.Errorf("expected toughness 5, got %d", c.EffectiveToughness())
	}
	if c.CounterCount(CounterPlusOnePlusOne) != 2 {
		t.Errorf("expected 2 counters, got %d", c.CounterCount(CounterPlusOnePlusOne))
	}
}

func TestCountersCancelOut(t *testing.T) {
	c := makeCreature(1, "Test", 3, 3, nil)
	c.AddCounters(CounterPlusOnePlusOne, 3)
	c.AddCounters(CounterMinusOneMinusOne, 2)
	c.CancelCounters()
	if c.CounterCount(CounterPlusOnePlusOne) != 1 {
		t.Errorf("expected 1 +1/+1 counter, got %d", c.CounterCount(CounterPlusOnePlusOne))
	}
	if c.CounterCount(CounterMinusOneMinusOne) != 0 {
		t.Errorf("expected 0 -1/-1 counters, got %d", c.CounterCount(CounterMinusOneMinusOne))
	}
	if c.EffectivePower() != 4 {
		t.Errorf("expected power 4, got %d", c.EffectivePower())
	}
}

func TestCountersPersistAcrossTurns(t *testing.T) {
	c := makeCreature(1, "Test", 1, 1, nil)
	c.AddCounters(CounterPlusOnePlusOne, 1)
	c.ClearEndOfTurnEffects()
	if c.EffectivePower() != 2 {
		t.Errorf("expected power 2 after end of turn, got %d", c.EffectivePower())
	}
	if c.EffectiveToughness() != 2 {
		t.Errorf("expected toughness 2 after end of turn, got %d", c.EffectiveToughness())
	}
}

func TestCountersClearedOnZoneChange(t *testing.T) {
	p := &Player{}
	domainCard := &domain.Card{
		CardName:  "Test Creature",
		CardType:  domain.CardTypeCreature,
		Power:     2,
		Toughness: 2,
	}
	c := NewCardFromDomain(domainCard, 1, p)
	c.CurrentZone = ZoneBattlefield
	p.Battlefield = []*Card{c}
	c.AddCounters(CounterPlusOnePlusOne, 3)

	err := p.MoveTo(c, ZoneGraveyard)
	if err != nil {
		t.Fatalf("MoveTo failed: %v", err)
	}
	if c.Counters != nil {
		t.Errorf("expected counters to be nil after zone change, got %v", c.Counters)
	}
}

func TestDeepCopyCounters(t *testing.T) {
	c := makeCreature(1, "Test", 2, 2, nil)
	c.AddCounters(CounterPlusOnePlusOne, 2)

	cp := c.DeepCopy()
	if cp.CounterCount(CounterPlusOnePlusOne) != 2 {
		t.Errorf("expected copied card to have 2 counters, got %d", cp.CounterCount(CounterPlusOnePlusOne))
	}

	cp.AddCounters(CounterPlusOnePlusOne, 1)
	if c.CounterCount(CounterPlusOnePlusOne) != 2 {
		t.Errorf("modifying copy should not affect original, got %d", c.CounterCount(CounterPlusOnePlusOne))
	}
}
