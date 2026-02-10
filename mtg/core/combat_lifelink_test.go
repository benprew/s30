package core

import (
	"testing"
)

func TestResolveCombatDamage_LifelinkUnblockedGainsLife(t *testing.T) {
	game, player, opponent := setupCombatTest()

	attacker := addCreatureToBattlefield(player, "Llanowar Elves", true, false)
	if attacker == nil {
		t.Skip("Llanowar Elves not found")
	}
	attacker.Power = 3
	attacker.Keywords = append(attacker.Keywords, "Lifelink")

	game.ActivePlayer = 0
	game.DeclareAttacker(attacker)
	game.Attackers = append(game.Attackers, attacker)

	initialLife := player.LifeTotal
	initialOpponentLife := opponent.LifeTotal
	game.ResolveCombatDamage()

	if opponent.LifeTotal != initialOpponentLife-3 {
		t.Errorf("Expected opponent life %d, got %d", initialOpponentLife-3, opponent.LifeTotal)
	}
	if player.LifeTotal != initialLife+3 {
		t.Errorf("Expected attacker's controller to gain 3 life (from %d to %d), got %d", initialLife, initialLife+3, player.LifeTotal)
	}
}

func TestResolveCombatDamage_LifelinkBlockedGainsLife(t *testing.T) {
	game, player, opponent := setupCombatTest()

	attacker := addCreatureToBattlefield(player, "Llanowar Elves", true, false)
	if attacker == nil {
		t.Skip("Llanowar Elves not found")
	}
	attacker.Power = 4
	attacker.Toughness = 4
	attacker.Keywords = append(attacker.Keywords, "Lifelink")

	blocker := addCreatureToBattlefield(opponent, "Llanowar Elves", true, false)
	if blocker == nil {
		t.Skip("Llanowar Elves not found for blocker")
	}
	blocker.Power = 2
	blocker.Toughness = 2

	game.ActivePlayer = 0
	game.DeclareAttacker(attacker)
	game.Attackers = append(game.Attackers, attacker)
	game.BlockerMap[attacker] = []*Card{blocker}

	initialLife := player.LifeTotal
	initialOpponentLife := opponent.LifeTotal
	game.ResolveCombatDamage()

	if player.LifeTotal != initialLife+4 {
		t.Errorf("Expected attacker's controller to gain 4 life (from %d to %d), got %d", initialLife, initialLife+4, player.LifeTotal)
	}
	if opponent.LifeTotal != initialOpponentLife {
		t.Errorf("Opponent life should not change when attacker is blocked, got %d", opponent.LifeTotal)
	}
}

func TestResolveCombatDamage_BlockerWithLifelinkGainsLife(t *testing.T) {
	game, player, opponent := setupCombatTest()

	attacker := addCreatureToBattlefield(player, "Llanowar Elves", true, false)
	if attacker == nil {
		t.Skip("Llanowar Elves not found")
	}
	attacker.Power = 3
	attacker.Toughness = 3

	blocker := addCreatureToBattlefield(opponent, "Llanowar Elves", true, false)
	if blocker == nil {
		t.Skip("Llanowar Elves not found for blocker")
	}
	blocker.Power = 2
	blocker.Toughness = 5
	blocker.Keywords = append(blocker.Keywords, "Lifelink")

	game.ActivePlayer = 0
	game.DeclareAttacker(attacker)
	game.Attackers = append(game.Attackers, attacker)
	game.BlockerMap[attacker] = []*Card{blocker}

	initialOpponentLife := opponent.LifeTotal
	game.ResolveCombatDamage()

	if opponent.LifeTotal != initialOpponentLife+2 {
		t.Errorf("Expected blocker's controller to gain 2 life (from %d to %d), got %d", initialOpponentLife, initialOpponentLife+2, opponent.LifeTotal)
	}
}

func TestResolveCombatDamage_WithoutLifelinkNoLifeGain(t *testing.T) {
	game, player, _ := setupCombatTest()

	attacker := addCreatureToBattlefield(player, "Llanowar Elves", true, false)
	if attacker == nil {
		t.Skip("Llanowar Elves not found")
	}
	attacker.Power = 3

	game.ActivePlayer = 0
	game.DeclareAttacker(attacker)
	game.Attackers = append(game.Attackers, attacker)

	initialLife := player.LifeTotal
	game.ResolveCombatDamage()

	if player.LifeTotal != initialLife {
		t.Errorf("Without lifelink, controller should not gain life. Expected %d, got %d", initialLife, player.LifeTotal)
	}
}

func TestResolveCombatDamage_MultipleLifelinkAttackersGainLife(t *testing.T) {
	game, player, opponent := setupCombatTest()

	attacker1 := addCreatureToBattlefield(player, "Llanowar Elves", true, false)
	if attacker1 == nil {
		t.Skip("Llanowar Elves not found")
	}
	attacker1.Power = 2
	attacker1.Keywords = append(attacker1.Keywords, "Lifelink")

	attacker2 := addCreatureToBattlefield(player, "Llanowar Elves", true, false)
	if attacker2 == nil {
		t.Skip("Llanowar Elves not found")
	}
	attacker2.Power = 3
	attacker2.Keywords = append(attacker2.Keywords, "Lifelink")

	game.ActivePlayer = 0
	game.DeclareAttacker(attacker1)
	game.Attackers = append(game.Attackers, attacker1)
	game.DeclareAttacker(attacker2)
	game.Attackers = append(game.Attackers, attacker2)

	initialLife := player.LifeTotal
	initialOpponentLife := opponent.LifeTotal
	game.ResolveCombatDamage()

	if opponent.LifeTotal != initialOpponentLife-5 {
		t.Errorf("Expected opponent to take 5 total damage, got %d", initialOpponentLife-opponent.LifeTotal)
	}
	if player.LifeTotal != initialLife+5 {
		t.Errorf("Expected controller to gain 5 total life, got %d gained", player.LifeTotal-initialLife)
	}
}
