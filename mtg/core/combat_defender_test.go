package core

import (
	"testing"
)

func TestDeclareAttacker_DefenderCannotAttack(t *testing.T) {
	game, player, _ := setupCombatTest()

	creature := addCreatureToBattlefield(player, "Wall of Bone", true, false)
	if creature == nil {
		t.Skip("Wall of Bone not found")
	}

	err := game.DeclareAttacker(creature)
	if err == nil {
		t.Errorf("DeclareAttacker should return error for creature with defender")
	}
}

func TestAvailableAttackers_ExcludesDefender(t *testing.T) {
	game, player, _ := setupCombatTest()

	defender := addCreatureToBattlefield(player, "Wall of Bone", true, false)
	if defender == nil {
		t.Skip("Wall of Bone not found")
	}

	normal := addCreatureToBattlefield(player, "Llanowar Elves", true, false)
	if normal == nil {
		t.Skip("Llanowar Elves not found")
	}

	attackers := game.AvailableAttackers(player)
	for _, a := range attackers {
		if a == defender {
			t.Errorf("Creature with defender should not be in available attackers")
		}
	}
	found := false
	for _, a := range attackers {
		if a == normal {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Normal creature should be in available attackers")
	}
}

func TestDefenderCanBlock(t *testing.T) {
	game, player, opponent := setupCombatTest()

	defender := addCreatureToBattlefield(player, "Wall of Bone", true, false)
	if defender == nil {
		t.Skip("Wall of Bone not found")
	}

	attacker := addCreatureToBattlefield(opponent, "Llanowar Elves", true, false)
	if attacker == nil {
		t.Skip("Llanowar Elves not found")
	}

	game.Attackers = append(game.Attackers, attacker)

	blockers := game.AvailableBlockers(player)
	found := false
	for _, b := range blockers {
		if b == defender {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Creature with defender should be available as blocker")
	}

	err := game.DeclareBlocker(defender, attacker)
	if err != nil {
		t.Errorf("DeclareBlocker should not return error for defender: %v", err)
	}
}
