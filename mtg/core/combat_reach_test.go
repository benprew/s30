package core

import (
	"testing"

	"github.com/benprew/s30/game/domain"
)

func TestDeclareBlocker_FlyingCanBeBlockedByReach(t *testing.T) {
	game, player, opponent := setupCombatTest()

	attacker := addCreatureToBattlefield(player, "Air Elemental", true, false)
	if attacker == nil {
		t.Skip("Air Elemental not found")
	}

	blocker := addCreatureToBattlefield(opponent, "Llanowar Elves", true, false)
	if blocker == nil {
		t.Skip("Llanowar Elves not found")
	}
	blocker.Keywords = append(blocker.Keywords, "Reach")

	err := game.DeclareBlocker(blocker, attacker)
	if err != nil {
		t.Errorf("Creature with reach should be able to block flying creature: %v", err)
	}
}

func TestAvailableActions_ReachCanBlockFlying(t *testing.T) {
	game, player, opponent := setupCombatTest()

	attacker := addCreatureToBattlefield(player, "Air Elemental", true, false)
	if attacker == nil {
		t.Skip("Air Elemental not found")
	}
	blocker := addCreatureToBattlefield(opponent, "Llanowar Elves", true, false)
	if blocker == nil {
		t.Skip("Llanowar Elves not found")
	}
	blocker.Keywords = append(blocker.Keywords, "Reach")

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
		t.Errorf("Creature with reach should be offered as blocker against flying attacker")
	}
}

func TestDeclareBlocker_ReachCanBlockGround(t *testing.T) {
	game, player, opponent := setupCombatTest()

	attacker := addCreatureToBattlefield(player, "Llanowar Elves", true, false)
	if attacker == nil {
		t.Skip("Llanowar Elves not found")
	}

	blocker := addCreatureToBattlefield(opponent, "Llanowar Elves", true, false)
	if blocker == nil {
		t.Skip("Llanowar Elves not found")
	}
	blocker.Keywords = append(blocker.Keywords, "Reach")

	err := game.DeclareBlocker(blocker, attacker)
	if err != nil {
		t.Errorf("Creature with reach should be able to block ground creature: %v", err)
	}
}

func TestDeclareBlocker_ReachViaParsedAbilities(t *testing.T) {
	game, player, opponent := setupCombatTest()

	attacker := addCreatureToBattlefield(player, "Air Elemental", true, false)
	if attacker == nil {
		t.Skip("Air Elemental not found")
	}

	blocker := addCreatureToBattlefield(opponent, "Llanowar Elves", true, false)
	if blocker == nil {
		t.Skip("Llanowar Elves not found")
	}
	blocker.ParsedAbilities = append(blocker.ParsedAbilities, domain.ParsedAbility{
		Type:     "Keyword",
		Keywords: []string{"Reach"},
	})

	err := game.DeclareBlocker(blocker, attacker)
	if err != nil {
		t.Errorf("Creature with reach via parsed abilities should block flying: %v", err)
	}
}
