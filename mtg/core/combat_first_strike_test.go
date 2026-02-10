package core

import (
	"testing"
)

func TestFirstStrike_AttackerKillsBlockerBeforeItDealsDamage(t *testing.T) {
	game, player, opponent := setupCombatTest()

	attacker := addCreatureToBattlefield(player, "Llanowar Elves", true, false)
	if attacker == nil {
		t.Skip("Llanowar Elves not found")
	}
	attacker.Power = 3
	attacker.Toughness = 1
	attacker.Keywords = append(attacker.Keywords, "First Strike")

	blocker := addCreatureToBattlefield(opponent, "Llanowar Elves", true, false)
	if blocker == nil {
		t.Skip("Llanowar Elves not found for blocker")
	}
	blocker.Power = 5
	blocker.Toughness = 2

	game.ActivePlayer = 0
	game.DeclareAttacker(attacker)
	game.Attackers = append(game.Attackers, attacker)
	game.BlockerMap[attacker] = []*Card{blocker}

	game.ResolveFirstStrikeDamage()

	if blocker.DamageTaken != 3 {
		t.Errorf("Blocker should have taken 3 first strike damage, got %d", blocker.DamageTaken)
	}
	if attacker.DamageTaken != 0 {
		t.Errorf("Attacker should not have taken damage during first strike step, got %d", attacker.DamageTaken)
	}
	if !blocker.IsDead() {
		t.Errorf("Blocker should be dead after first strike damage")
	}

	game.CheckStateBasedActions()

	if blocker.CurrentZone != ZoneGraveyard {
		t.Errorf("Blocker should be in graveyard after SBAs, got zone %d", blocker.CurrentZone)
	}

	game.ResolveCombatDamage()

	if attacker.DamageTaken != 0 {
		t.Errorf("Attacker should not take damage from dead blocker, got %d", attacker.DamageTaken)
	}
}

func TestFirstStrike_BlockerWithFirstStrikeKillsAttacker(t *testing.T) {
	game, player, opponent := setupCombatTest()

	attacker := addCreatureToBattlefield(player, "Llanowar Elves", true, false)
	if attacker == nil {
		t.Skip("Llanowar Elves not found")
	}
	attacker.Power = 5
	attacker.Toughness = 2

	blocker := addCreatureToBattlefield(opponent, "Llanowar Elves", true, false)
	if blocker == nil {
		t.Skip("Llanowar Elves not found for blocker")
	}
	blocker.Power = 3
	blocker.Toughness = 1
	blocker.Keywords = append(blocker.Keywords, "First Strike")

	game.ActivePlayer = 0
	game.DeclareAttacker(attacker)
	game.Attackers = append(game.Attackers, attacker)
	game.BlockerMap[attacker] = []*Card{blocker}

	game.ResolveFirstStrikeDamage()

	if attacker.DamageTaken != 3 {
		t.Errorf("Attacker should have taken 3 first strike damage, got %d", attacker.DamageTaken)
	}
	if !attacker.IsDead() {
		t.Errorf("Attacker should be dead after first strike damage")
	}

	game.CheckStateBasedActions()

	if attacker.CurrentZone != ZoneGraveyard {
		t.Errorf("Attacker should be in graveyard after SBAs, got zone %d", attacker.CurrentZone)
	}

	game.ResolveCombatDamage()

	if blocker.DamageTaken != 0 {
		t.Errorf("Blocker should not take damage from dead attacker, got %d", blocker.DamageTaken)
	}
}

func TestFirstStrike_UnblockedAttackerDealsDamageInFirstStrikeStep(t *testing.T) {
	game, player, opponent := setupCombatTest()

	attacker := addCreatureToBattlefield(player, "Llanowar Elves", true, false)
	if attacker == nil {
		t.Skip("Llanowar Elves not found")
	}
	attacker.Power = 3
	attacker.Toughness = 2
	attacker.Keywords = append(attacker.Keywords, "First Strike")

	game.ActivePlayer = 0
	game.DeclareAttacker(attacker)
	game.Attackers = append(game.Attackers, attacker)

	initialLife := opponent.LifeTotal
	game.ResolveFirstStrikeDamage()

	if opponent.LifeTotal != initialLife-3 {
		t.Errorf("Expected opponent life %d after first strike, got %d", initialLife-3, opponent.LifeTotal)
	}

	game.CheckStateBasedActions()
	game.ResolveCombatDamage()

	if opponent.LifeTotal != initialLife-3 {
		t.Errorf("First strike attacker should not deal damage again in normal step, expected %d got %d", initialLife-3, opponent.LifeTotal)
	}
}

func TestFirstStrike_BothCreaturesHaveFirstStrike(t *testing.T) {
	game, player, opponent := setupCombatTest()

	attacker := addCreatureToBattlefield(player, "Llanowar Elves", true, false)
	if attacker == nil {
		t.Skip("Llanowar Elves not found")
	}
	attacker.Power = 2
	attacker.Toughness = 3
	attacker.Keywords = append(attacker.Keywords, "First Strike")

	blocker := addCreatureToBattlefield(opponent, "Llanowar Elves", true, false)
	if blocker == nil {
		t.Skip("Llanowar Elves not found for blocker")
	}
	blocker.Power = 2
	blocker.Toughness = 3
	blocker.Keywords = append(blocker.Keywords, "First Strike")

	game.ActivePlayer = 0
	game.DeclareAttacker(attacker)
	game.Attackers = append(game.Attackers, attacker)
	game.BlockerMap[attacker] = []*Card{blocker}

	game.ResolveFirstStrikeDamage()

	if attacker.DamageTaken != 2 {
		t.Errorf("Attacker should have taken 2 first strike damage, got %d", attacker.DamageTaken)
	}
	if blocker.DamageTaken != 2 {
		t.Errorf("Blocker should have taken 2 first strike damage, got %d", blocker.DamageTaken)
	}

	game.CheckStateBasedActions()
	game.ResolveCombatDamage()

	if attacker.DamageTaken != 2 {
		t.Errorf("Attacker should not take additional damage in normal step, got %d", attacker.DamageTaken)
	}
	if blocker.DamageTaken != 2 {
		t.Errorf("Blocker should not take additional damage in normal step, got %d", blocker.DamageTaken)
	}
}

func TestFirstStrike_NormalCombatUnchangedWithoutFirstStrike(t *testing.T) {
	game, player, opponent := setupCombatTest()

	attacker := addCreatureToBattlefield(player, "Llanowar Elves", true, false)
	if attacker == nil {
		t.Skip("Llanowar Elves not found")
	}
	attacker.Power = 2
	attacker.Toughness = 2

	blocker := addCreatureToBattlefield(opponent, "Llanowar Elves", true, false)
	if blocker == nil {
		t.Skip("Llanowar Elves not found for blocker")
	}
	blocker.Power = 1
	blocker.Toughness = 1

	game.ActivePlayer = 0
	game.DeclareAttacker(attacker)
	game.Attackers = append(game.Attackers, attacker)
	game.BlockerMap[attacker] = []*Card{blocker}

	initialLife := opponent.LifeTotal
	game.ResolveCombatDamage()

	if attacker.DamageTaken != blocker.Power {
		t.Errorf("Attacker should have taken %d damage, got %d", blocker.Power, attacker.DamageTaken)
	}
	if blocker.DamageTaken != attacker.Power {
		t.Errorf("Blocker should have taken %d damage, got %d", attacker.Power, blocker.DamageTaken)
	}
	if opponent.LifeTotal != initialLife {
		t.Errorf("Blocked attacker should not deal damage to player")
	}
}

func TestFirstStrike_AttackerDoesNotKillBlocker_BlockerDealsDamageBack(t *testing.T) {
	game, player, opponent := setupCombatTest()

	attacker := addCreatureToBattlefield(player, "Llanowar Elves", true, false)
	if attacker == nil {
		t.Skip("Llanowar Elves not found")
	}
	attacker.Power = 1
	attacker.Toughness = 1
	attacker.Keywords = append(attacker.Keywords, "First Strike")

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

	game.ResolveFirstStrikeDamage()

	if blocker.DamageTaken != 1 {
		t.Errorf("Blocker should have taken 1 first strike damage, got %d", blocker.DamageTaken)
	}
	if blocker.IsDead() {
		t.Errorf("Blocker should survive first strike damage")
	}

	game.CheckStateBasedActions()

	if blocker.CurrentZone != ZoneBattlefield {
		t.Errorf("Surviving blocker should still be on battlefield, got zone %d", blocker.CurrentZone)
	}

	game.ResolveCombatDamage()

	if attacker.DamageTaken != 5 {
		t.Errorf("Attacker should take 5 damage from surviving blocker, got %d", attacker.DamageTaken)
	}
}

func TestCombatHasFirstStrike_DetectsFirstStrikeAttacker(t *testing.T) {
	game, player, _ := setupCombatTest()

	attacker := addCreatureToBattlefield(player, "Llanowar Elves", true, false)
	if attacker == nil {
		t.Skip("Llanowar Elves not found")
	}
	attacker.Keywords = append(attacker.Keywords, "First Strike")

	game.Attackers = append(game.Attackers, attacker)

	if !game.combatHasFirstStrike() {
		t.Errorf("combatHasFirstStrike should return true when attacker has first strike")
	}
}

func TestCombatHasFirstStrike_DetectsFirstStrikeBlocker(t *testing.T) {
	game, player, opponent := setupCombatTest()

	attacker := addCreatureToBattlefield(player, "Llanowar Elves", true, false)
	if attacker == nil {
		t.Skip("Llanowar Elves not found")
	}
	blocker := addCreatureToBattlefield(opponent, "Llanowar Elves", true, false)
	if blocker == nil {
		t.Skip("Llanowar Elves not found for blocker")
	}
	blocker.Keywords = append(blocker.Keywords, "First Strike")

	game.Attackers = append(game.Attackers, attacker)
	game.BlockerMap[attacker] = []*Card{blocker}

	if !game.combatHasFirstStrike() {
		t.Errorf("combatHasFirstStrike should return true when blocker has first strike")
	}
}

func TestCombatHasFirstStrike_FalseWithNoFirstStrike(t *testing.T) {
	game, player, opponent := setupCombatTest()

	attacker := addCreatureToBattlefield(player, "Llanowar Elves", true, false)
	if attacker == nil {
		t.Skip("Llanowar Elves not found")
	}
	blocker := addCreatureToBattlefield(opponent, "Llanowar Elves", true, false)
	if blocker == nil {
		t.Skip("Llanowar Elves not found for blocker")
	}

	game.Attackers = append(game.Attackers, attacker)
	game.BlockerMap[attacker] = []*Card{blocker}

	if game.combatHasFirstStrike() {
		t.Errorf("combatHasFirstStrike should return false when no creature has first strike")
	}
}
