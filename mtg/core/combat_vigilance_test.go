package core

import (
	"testing"
)

func TestDeclareAttacker_VigilanceDoesNotTap(t *testing.T) {
	game, player, _ := setupCombatTest()

	creature := addCreatureToBattlefield(player, "Llanowar Elves", true, false)
	if creature == nil {
		t.Skip("Llanowar Elves not found")
	}
	creature.Keywords = append(creature.Keywords, "Vigilance")

	err := game.DeclareAttacker(creature)
	if err != nil {
		t.Errorf("DeclareAttacker returned error: %v", err)
	}

	if creature.Tapped {
		t.Errorf("Creature with vigilance should not be tapped after declaring attack")
	}
}

func TestDeclareAttacker_WithoutVigilanceTaps(t *testing.T) {
	game, player, _ := setupCombatTest()

	creature := addCreatureToBattlefield(player, "Llanowar Elves", true, false)
	if creature == nil {
		t.Skip("Llanowar Elves not found")
	}

	err := game.DeclareAttacker(creature)
	if err != nil {
		t.Errorf("DeclareAttacker returned error: %v", err)
	}

	if !creature.Tapped {
		t.Errorf("Creature without vigilance should be tapped after declaring attack")
	}
}

func TestVigilanceCreatureCanBlockAfterAttacking(t *testing.T) {
	game, player, opponent := setupCombatTest()

	creature := addCreatureToBattlefield(player, "Llanowar Elves", true, false)
	if creature == nil {
		t.Skip("Llanowar Elves not found")
	}
	creature.Keywords = append(creature.Keywords, "Vigilance")

	err := game.DeclareAttacker(creature)
	if err != nil {
		t.Errorf("DeclareAttacker returned error: %v", err)
	}

	game.ClearCombatState()

	opponentAttacker := addCreatureToBattlefield(opponent, "Llanowar Elves", true, false)
	if opponentAttacker == nil {
		t.Skip("Llanowar Elves not found for opponent")
	}

	blockers := game.AvailableBlockers(player)
	found := false
	for _, b := range blockers {
		if b == creature {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Creature with vigilance should be available as blocker after attacking (since it didn't tap)")
	}
}
