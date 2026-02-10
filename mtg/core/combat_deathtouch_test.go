package core

import (
	"testing"
)

func TestDeathtouch_SmallCreatureKillsLargeCreature(t *testing.T) {
	game, player, opponent := setupCombatTest()

	attacker := addCreatureToBattlefield(player, "Llanowar Elves", true, false)
	if attacker == nil {
		t.Skip("Llanowar Elves not found")
	}
	attacker.Power = 1
	attacker.Toughness = 1
	attacker.Keywords = append(attacker.Keywords, "Deathtouch")

	blocker := addCreatureToBattlefield(opponent, "Llanowar Elves", true, false)
	if blocker == nil {
		t.Skip("Llanowar Elves not found for blocker")
	}
	blocker.Power = 5
	blocker.Toughness = 5

	game.ActivePlayer = 0
	game.DeclareAttacker(attacker)
	game.Attackers = append(game.Attackers, attacker)
	game.BlockerMap[attacker] = []*Card{blocker}

	game.ResolveCombatDamage()
	game.CheckStateBasedActions()

	if !blocker.IsDead() {
		t.Errorf("Creature dealt damage by deathtouch should be dead even with 1 damage on a 5-toughness creature")
	}

	blockerInGraveyard := false
	for _, c := range opponent.Graveyard {
		if c == blocker {
			blockerInGraveyard = true
			break
		}
	}
	if !blockerInGraveyard {
		t.Errorf("Creature killed by deathtouch should be in graveyard")
	}
}

func TestDeathtouch_BlockerWithDeathtouchKillsAttacker(t *testing.T) {
	game, player, opponent := setupCombatTest()

	attacker := addCreatureToBattlefield(player, "Llanowar Elves", true, false)
	if attacker == nil {
		t.Skip("Llanowar Elves not found")
	}
	attacker.Power = 5
	attacker.Toughness = 5

	blocker := addCreatureToBattlefield(opponent, "Llanowar Elves", true, false)
	if blocker == nil {
		t.Skip("Llanowar Elves not found for blocker")
	}
	blocker.Power = 1
	blocker.Toughness = 1
	blocker.Keywords = append(blocker.Keywords, "Deathtouch")

	game.ActivePlayer = 0
	game.DeclareAttacker(attacker)
	game.Attackers = append(game.Attackers, attacker)
	game.BlockerMap[attacker] = []*Card{blocker}

	game.ResolveCombatDamage()
	game.CheckStateBasedActions()

	if !attacker.IsDead() {
		t.Errorf("Attacker dealt damage by deathtouch blocker should be dead")
	}

	attackerInGraveyard := false
	for _, c := range player.Graveyard {
		if c == attacker {
			attackerInGraveyard = true
			break
		}
	}
	if !attackerInGraveyard {
		t.Errorf("Attacker killed by deathtouch blocker should be in graveyard")
	}
}

func TestDeathtouch_BothCreaturesDieWithDeathtouch(t *testing.T) {
	game, player, opponent := setupCombatTest()

	attacker := addCreatureToBattlefield(player, "Llanowar Elves", true, false)
	if attacker == nil {
		t.Skip("Llanowar Elves not found")
	}
	attacker.Power = 1
	attacker.Toughness = 1
	attacker.Keywords = append(attacker.Keywords, "Deathtouch")

	blocker := addCreatureToBattlefield(opponent, "Llanowar Elves", true, false)
	if blocker == nil {
		t.Skip("Llanowar Elves not found for blocker")
	}
	blocker.Power = 1
	blocker.Toughness = 1
	blocker.Keywords = append(blocker.Keywords, "Deathtouch")

	game.ActivePlayer = 0
	game.DeclareAttacker(attacker)
	game.Attackers = append(game.Attackers, attacker)
	game.BlockerMap[attacker] = []*Card{blocker}

	game.ResolveCombatDamage()
	game.CheckStateBasedActions()

	if !attacker.IsDead() {
		t.Errorf("Both creatures with deathtouch should die - attacker survived")
	}
	if !blocker.IsDead() {
		t.Errorf("Both creatures with deathtouch should die - blocker survived")
	}
}

func TestDeathtouch_NoDeathtouchDoesNotKillLargeCreature(t *testing.T) {
	game, player, opponent := setupCombatTest()

	attacker := addCreatureToBattlefield(player, "Llanowar Elves", true, false)
	if attacker == nil {
		t.Skip("Llanowar Elves not found")
	}
	attacker.Power = 1
	attacker.Toughness = 1

	blocker := addCreatureToBattlefield(opponent, "Llanowar Elves", true, false)
	if blocker == nil {
		t.Skip("Llanowar Elves not found for blocker")
	}
	blocker.Power = 5
	blocker.Toughness = 5

	game.ActivePlayer = 0
	game.DeclareAttacker(attacker)
	game.Attackers = append(game.Attackers, attacker)
	game.BlockerMap[attacker] = []*Card{blocker}

	game.ResolveCombatDamage()
	game.CheckStateBasedActions()

	if blocker.IsDead() {
		t.Errorf("Without deathtouch, 1 damage should not kill a 5-toughness creature")
	}
}

func TestDeathtouch_UnblockedDealsDamageToPlayer(t *testing.T) {
	game, player, opponent := setupCombatTest()

	attacker := addCreatureToBattlefield(player, "Llanowar Elves", true, false)
	if attacker == nil {
		t.Skip("Llanowar Elves not found")
	}
	attacker.Power = 1
	attacker.Toughness = 1
	attacker.Keywords = append(attacker.Keywords, "Deathtouch")

	game.ActivePlayer = 0
	game.DeclareAttacker(attacker)
	game.Attackers = append(game.Attackers, attacker)

	initialLife := opponent.LifeTotal
	game.ResolveCombatDamage()

	if opponent.LifeTotal != initialLife-1 {
		t.Errorf("Deathtouch creature unblocked should deal normal damage to player, expected %d got %d",
			initialLife-1, opponent.LifeTotal)
	}
}

func TestDeathtouch_ZeroPowerDoesNotKill(t *testing.T) {
	game, player, opponent := setupCombatTest()

	attacker := addCreatureToBattlefield(player, "Llanowar Elves", true, false)
	if attacker == nil {
		t.Skip("Llanowar Elves not found")
	}
	attacker.Power = 0
	attacker.Toughness = 1
	attacker.Keywords = append(attacker.Keywords, "Deathtouch")

	blocker := addCreatureToBattlefield(opponent, "Llanowar Elves", true, false)
	if blocker == nil {
		t.Skip("Llanowar Elves not found for blocker")
	}
	blocker.Power = 5
	blocker.Toughness = 5

	game.ActivePlayer = 0
	game.DeclareAttacker(attacker)
	game.Attackers = append(game.Attackers, attacker)
	game.BlockerMap[attacker] = []*Card{blocker}

	game.ResolveCombatDamage()
	game.CheckStateBasedActions()

	if blocker.IsDead() {
		t.Errorf("Deathtouch creature with 0 power deals no damage, should not kill blocker")
	}
}

func TestResolveCombatDamage_DeathtouchTrampleAssignsOneDamagePerBlocker(t *testing.T) {
	game, player, opponent := setupCombatTest()

	attacker := addCreatureToBattlefield(player, "War Mammoth", true, false)
	if attacker == nil {
		t.Skip("War Mammoth not found")
	}
	attacker.Power = 6
	attacker.Toughness = 6
	attacker.Keywords = append(attacker.Keywords, "Trample", "Deathtouch")

	blocker := addCreatureToBattlefield(opponent, "Llanowar Elves", true, false)
	if blocker == nil {
		t.Skip("Llanowar Elves not found")
	}
	blocker.Power = 1
	blocker.Toughness = 5

	game.ActivePlayer = 0
	game.DeclareAttacker(attacker)
	game.Attackers = append(game.Attackers, attacker)
	game.BlockerMap[attacker] = []*Card{blocker}

	initialLife := opponent.LifeTotal
	game.ResolveCombatDamage()

	if blocker.DamageTaken != 1 {
		t.Errorf("With deathtouch+trample, blocker should take 1 damage (lethal with deathtouch), got %d", blocker.DamageTaken)
	}

	if !blocker.DeathtouchDamaged {
		t.Errorf("Blocker should be marked as deathtouch damaged")
	}

	expectedLifeLoss := 5
	if opponent.LifeTotal != initialLife-expectedLifeLoss {
		t.Errorf("Expected opponent life %d (deathtouch+trample excess), got %d", initialLife-expectedLifeLoss, opponent.LifeTotal)
	}

	game.CheckStateBasedActions()
	if !blocker.IsDead() {
		t.Errorf("Blocker should be dead from deathtouch")
	}
}

func TestResolveCombatDamage_DeathtouchTrampleMultipleBlockers(t *testing.T) {
	game, player, opponent := setupCombatTest()

	attacker := addCreatureToBattlefield(player, "War Mammoth", true, false)
	if attacker == nil {
		t.Skip("War Mammoth not found")
	}
	attacker.Power = 7
	attacker.Toughness = 7
	attacker.Keywords = append(attacker.Keywords, "Trample", "Deathtouch")

	blocker1 := addCreatureToBattlefield(opponent, "Llanowar Elves", true, false)
	if blocker1 == nil {
		t.Skip("Llanowar Elves not found")
	}
	blocker1.Power = 1
	blocker1.Toughness = 4

	blocker2 := addCreatureToBattlefield(opponent, "Llanowar Elves", true, false)
	if blocker2 == nil {
		t.Skip("Llanowar Elves not found for blocker2")
	}
	blocker2.Power = 1
	blocker2.Toughness = 5

	game.ActivePlayer = 0
	game.DeclareAttacker(attacker)
	game.Attackers = append(game.Attackers, attacker)
	game.BlockerMap[attacker] = []*Card{blocker1, blocker2}

	initialLife := opponent.LifeTotal
	game.ResolveCombatDamage()

	if blocker1.DamageTaken != 1 {
		t.Errorf("Blocker1 should take 1 damage (deathtouch lethal), got %d", blocker1.DamageTaken)
	}
	if blocker2.DamageTaken != 1 {
		t.Errorf("Blocker2 should take 1 damage (deathtouch lethal), got %d", blocker2.DamageTaken)
	}

	expectedLifeLoss := 5
	if opponent.LifeTotal != initialLife-expectedLifeLoss {
		t.Errorf("Expected opponent life %d (deathtouch+trample excess from 2 blockers), got %d", initialLife-expectedLifeLoss, opponent.LifeTotal)
	}
}
