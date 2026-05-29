package duel

import (
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"
)

func TestCreatureStatsTextSubtractsMarkedDamage(t *testing.T) {
	perm := interactive.PermanentState{Power: 3, Toughness: 4, Damage: 2}

	got := creatureStatsText(perm)
	if got != "3/2" {
		t.Errorf("want 3/2, got %q", got)
	}
}

func TestCreatureStatsTextDoesNotGoBelowZero(t *testing.T) {
	perm := interactive.PermanentState{Power: 1, Toughness: 1, Damage: 3}

	got := creatureStatsText(perm)
	if got != "1/0" {
		t.Errorf("want 1/0, got %q", got)
	}
}
