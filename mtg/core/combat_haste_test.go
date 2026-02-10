package core

import (
	"testing"
)

func TestHaste_CanAttackImmediately(t *testing.T) {
	game, player, _ := setupCombatTest()

	creature := addCreatureToBattlefield(player, "Llanowar Elves", false, false)
	if creature == nil {
		t.Skip("Llanowar Elves not found")
	}
	creature.Keywords = append(creature.Keywords, "Haste")

	attackers := game.AvailableAttackers(player)

	found := false
	for _, a := range attackers {
		if a == creature {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Creature with haste should be available as attacker even when summoning sick")
	}
}

func TestHaste_DeclareAttackerSucceeds(t *testing.T) {
	game, player, _ := setupCombatTest()

	creature := addCreatureToBattlefield(player, "Llanowar Elves", false, false)
	if creature == nil {
		t.Skip("Llanowar Elves not found")
	}
	creature.Keywords = append(creature.Keywords, "Haste")

	err := game.DeclareAttacker(creature)
	if err != nil {
		t.Errorf("Creature with haste should be able to attack: %v", err)
	}

	if !creature.Tapped {
		t.Errorf("Creature should be tapped after attacking")
	}
}

func TestHaste_WithoutKeywordCannotAttackWhenSummoningSick(t *testing.T) {
	game, player, _ := setupCombatTest()

	creature := addCreatureToBattlefield(player, "Llanowar Elves", false, false)
	if creature == nil {
		t.Skip("Llanowar Elves not found")
	}

	attackers := game.AvailableAttackers(player)

	for _, a := range attackers {
		if a == creature {
			t.Errorf("Creature without haste should not be available as attacker when summoning sick")
		}
	}

	err := game.DeclareAttacker(creature)
	if err == nil {
		t.Errorf("Creature without haste should not be able to attack when summoning sick")
	}
}

func TestHaste_AvailableActionsIncludesAttack(t *testing.T) {
	game, player, _ := setupCombatTest()

	creature := addCreatureToBattlefield(player, "Llanowar Elves", false, false)
	if creature == nil {
		t.Skip("Llanowar Elves not found")
	}
	creature.Keywords = append(creature.Keywords, "Haste")

	player.Turn.Phase = PhaseCombat
	player.Turn.CombatStep = CombatStepDeclareAttackers
	game.ActivePlayer = 0

	actions := game.AvailableActions(player)

	hasAttackAction := false
	for _, a := range actions {
		if a.Type == ActionDeclareAttacker && a.Card == creature {
			hasAttackAction = true
			break
		}
	}

	if !hasAttackAction {
		t.Errorf("Creature with haste should have attack action in AvailableActions when summoning sick")
	}
}
