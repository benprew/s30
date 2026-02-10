package core

import (
	"testing"
)

func TestDeclareBlocker_FlyingCannotBeBlockedByGround(t *testing.T) {
	game, player, opponent := setupCombatTest()

	attacker := addCreatureToBattlefield(player, "Air Elemental", true, false)
	if attacker == nil {
		t.Skip("Air Elemental not found")
	}

	blocker := addCreatureToBattlefield(opponent, "Llanowar Elves", true, false)
	if blocker == nil {
		t.Skip("Llanowar Elves not found")
	}

	err := game.DeclareBlocker(blocker, attacker)
	if err == nil {
		t.Errorf("Ground creature should not be able to block flying creature")
	}
}

func TestDeclareBlocker_FlyingCanBeBlockedByFlying(t *testing.T) {
	game, player, opponent := setupCombatTest()

	attacker := addCreatureToBattlefield(player, "Air Elemental", true, false)
	if attacker == nil {
		t.Skip("Air Elemental not found")
	}

	blocker := addCreatureToBattlefield(opponent, "Air Elemental", true, false)
	if blocker == nil {
		t.Skip("Air Elemental not found for blocker")
	}

	err := game.DeclareBlocker(blocker, attacker)
	if err != nil {
		t.Errorf("Flying creature should be able to block flying creature: %v", err)
	}
}

func TestAvailableActions_GroundCannotBlockFlying(t *testing.T) {
	game, player, opponent := setupCombatTest()

	attacker := addCreatureToBattlefield(player, "Air Elemental", true, false)
	if attacker == nil {
		t.Skip("Air Elemental not found")
	}
	blocker := addCreatureToBattlefield(opponent, "Llanowar Elves", true, false)
	if blocker == nil {
		t.Skip("Llanowar Elves not found")
	}

	game.ActivePlayer = 0
	player.Turn.Phase = PhaseCombat
	player.Turn.CombatStep = CombatStepDeclareBlockers
	game.DeclareAttacker(attacker)
	game.Attackers = append(game.Attackers, attacker)

	actions := game.AvailableActions(opponent)

	for _, a := range actions {
		if a.Type == ActionDeclareBlocker && a.Card == blocker && a.Target == attacker {
			t.Errorf("Ground creature should not be offered as blocker against flying attacker")
		}
	}
}

func TestAvailableActions_FlyingCanBlockFlying(t *testing.T) {
	game, player, opponent := setupCombatTest()

	attacker := addCreatureToBattlefield(player, "Air Elemental", true, false)
	if attacker == nil {
		t.Skip("Air Elemental not found")
	}
	blocker := addCreatureToBattlefield(opponent, "Air Elemental", true, false)
	if blocker == nil {
		t.Skip("Air Elemental not found for blocker")
	}

	game.ActivePlayer = 0
	player.Turn.Phase = PhaseCombat
	player.Turn.CombatStep = CombatStepDeclareBlockers
	game.DeclareAttacker(attacker)
	game.Attackers = append(game.Attackers, attacker)

	actions := game.AvailableActions(opponent)

	found := false
	for _, a := range actions {
		if a.Type == ActionDeclareBlocker && a.Card == blocker && a.Target == attacker {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Flying creature should be offered as blocker against flying attacker")
	}
}

func TestAvailableActions_FlyingBlockerFilteredInActions(t *testing.T) {
	game, player, opponent := setupCombatTest()

	attacker := addCreatureToBattlefield(player, "Air Elemental", true, false)
	if attacker == nil {
		t.Skip("Air Elemental not found")
	}

	groundBlocker := addCreatureToBattlefield(opponent, "Llanowar Elves", true, false)
	if groundBlocker == nil {
		t.Skip("Llanowar Elves not found")
	}

	game.ActivePlayer = 0
	player.Turn.Phase = PhaseCombat
	player.Turn.CombatStep = CombatStepDeclareBlockers
	game.DeclareAttacker(attacker)
	game.Attackers = append(game.Attackers, attacker)

	actions := game.AvailableActions(opponent)

	for _, a := range actions {
		if a.Type == ActionDeclareBlocker && a.Card == groundBlocker && a.Target == attacker {
			t.Errorf("Ground creature should not be offered as a blocker for a flying attacker in AvailableActions")
		}
	}
}

func TestDeclareBlocker_GroundCanBeBlockedByGround(t *testing.T) {
	game, player, opponent := setupCombatTest()

	attacker := addCreatureToBattlefield(player, "Llanowar Elves", true, false)
	if attacker == nil {
		t.Skip("Llanowar Elves not found")
	}

	blocker := addCreatureToBattlefield(opponent, "Llanowar Elves", true, false)
	if blocker == nil {
		t.Skip("Llanowar Elves not found for blocker")
	}

	err := game.DeclareBlocker(blocker, attacker)
	if err != nil {
		t.Errorf("Ground creature should be able to block ground creature: %v", err)
	}
}
