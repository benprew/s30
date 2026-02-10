package core

import (
	"testing"
)

func TestResolveCombatDamage_TrampleDealExcessToPlayer(t *testing.T) {
	game, player, opponent := setupCombatTest()

	attacker := addCreatureToBattlefield(player, "War Mammoth", true, false)
	if attacker == nil {
		t.Skip("War Mammoth not found")
	}
	attacker.Power = 5
	attacker.Toughness = 5

	blocker := addCreatureToBattlefield(opponent, "Llanowar Elves", true, false)
	if blocker == nil {
		t.Skip("Llanowar Elves not found")
	}
	blocker.Power = 1
	blocker.Toughness = 1

	game.ActivePlayer = 0
	game.DeclareAttacker(attacker)
	game.Attackers = append(game.Attackers, attacker)
	game.BlockerMap[attacker] = []*Card{blocker}

	initialLife := opponent.LifeTotal
	game.ResolveCombatDamage()

	if blocker.DamageTaken != 1 {
		t.Errorf("Blocker should have taken %d damage (lethal), got %d", 1, blocker.DamageTaken)
	}

	expectedLifeLoss := 4
	if opponent.LifeTotal != initialLife-expectedLifeLoss {
		t.Errorf("Expected opponent life %d (trample excess), got %d", initialLife-expectedLifeLoss, opponent.LifeTotal)
	}
}

func TestResolveCombatDamage_TrampleMultipleBlockers(t *testing.T) {
	game, player, opponent := setupCombatTest()

	attacker := addCreatureToBattlefield(player, "War Mammoth", true, false)
	if attacker == nil {
		t.Skip("War Mammoth not found")
	}
	attacker.Power = 7
	attacker.Toughness = 7

	blocker1 := addCreatureToBattlefield(opponent, "Llanowar Elves", true, false)
	if blocker1 == nil {
		t.Skip("Llanowar Elves not found")
	}
	blocker1.Power = 1
	blocker1.Toughness = 2

	blocker2 := addCreatureToBattlefield(opponent, "Llanowar Elves", true, false)
	if blocker2 == nil {
		t.Skip("Llanowar Elves not found for blocker2")
	}
	blocker2.Power = 1
	blocker2.Toughness = 3

	game.ActivePlayer = 0
	game.DeclareAttacker(attacker)
	game.Attackers = append(game.Attackers, attacker)
	game.BlockerMap[attacker] = []*Card{blocker1, blocker2}

	initialLife := opponent.LifeTotal
	game.ResolveCombatDamage()

	if blocker1.DamageTaken != 2 {
		t.Errorf("Blocker1 should have taken %d damage (lethal), got %d", 2, blocker1.DamageTaken)
	}
	if blocker2.DamageTaken != 3 {
		t.Errorf("Blocker2 should have taken %d damage (lethal), got %d", 3, blocker2.DamageTaken)
	}

	expectedLifeLoss := 2
	if opponent.LifeTotal != initialLife-expectedLifeLoss {
		t.Errorf("Expected opponent life %d (trample excess from multiple blockers), got %d", initialLife-expectedLifeLoss, opponent.LifeTotal)
	}
}

func TestResolveCombatDamage_TrampleNoExcessNoDamageToPlayer(t *testing.T) {
	game, player, opponent := setupCombatTest()

	attacker := addCreatureToBattlefield(player, "War Mammoth", true, false)
	if attacker == nil {
		t.Skip("War Mammoth not found")
	}
	attacker.Power = 3
	attacker.Toughness = 3

	blocker := addCreatureToBattlefield(opponent, "Llanowar Elves", true, false)
	if blocker == nil {
		t.Skip("Llanowar Elves not found")
	}
	blocker.Power = 2
	blocker.Toughness = 4

	game.ActivePlayer = 0
	game.DeclareAttacker(attacker)
	game.Attackers = append(game.Attackers, attacker)
	game.BlockerMap[attacker] = []*Card{blocker}

	initialLife := opponent.LifeTotal
	game.ResolveCombatDamage()

	if blocker.DamageTaken != 3 {
		t.Errorf("Blocker should have taken %d damage, got %d", 3, blocker.DamageTaken)
	}

	if opponent.LifeTotal != initialLife {
		t.Errorf("No excess damage, opponent life should be %d, got %d", initialLife, opponent.LifeTotal)
	}
}
