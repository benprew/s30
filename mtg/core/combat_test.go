package core

import (
	"testing"

	"github.com/benprew/s30/game/domain"
)

func setupCombatTest() (*GameState, *Player, *Player) {
	players := createTestPlayer(2)
	game := NewGame(players)
	game.StartGame()
	return game, players[0], players[1]
}

func addCreatureToBattlefield(player *Player, name string, active bool, tapped bool) *Card {
	domainCard := domain.FindCardByName(name)
	if domainCard == nil {
		return nil
	}
	card := NewCardFromDomain(domainCard, EntityID(100+len(player.Battlefield)), player)
	card.Active = active
	card.Tapped = tapped
	card.CurrentZone = ZoneBattlefield
	player.Battlefield = append(player.Battlefield, card)
	return card
}

func TestAvailableAttackers_ReturnsUntappedActiveCreatures(t *testing.T) {
	game, player, _ := setupCombatTest()

	creature := addCreatureToBattlefield(player, "Llanowar Elves", true, false)
	if creature == nil {
		t.Skip("Llanowar Elves not found")
	}

	attackers := game.AvailableAttackers(player)

	found := false
	for _, a := range attackers {
		if a == creature {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected untapped active creature to be available as attacker")
	}
}

func TestAvailableAttackers_ExcludesTappedCreatures(t *testing.T) {
	game, player, _ := setupCombatTest()

	creature := addCreatureToBattlefield(player, "Llanowar Elves", true, true)
	if creature == nil {
		t.Skip("Llanowar Elves not found")
	}

	attackers := game.AvailableAttackers(player)

	for _, a := range attackers {
		if a == creature {
			t.Errorf("Tapped creature should not be available as attacker")
		}
	}
}

func TestAvailableAttackers_ExcludesInactiveCreatures(t *testing.T) {
	game, player, _ := setupCombatTest()

	creature := addCreatureToBattlefield(player, "Llanowar Elves", false, false)
	if creature == nil {
		t.Skip("Llanowar Elves not found")
	}

	attackers := game.AvailableAttackers(player)

	for _, a := range attackers {
		if a == creature {
			t.Errorf("Inactive (summoning sick) creature should not be available as attacker")
		}
	}
}

func TestAvailableAttackers_ExcludesNonCreatures(t *testing.T) {
	game, player, _ := setupCombatTest()

	land := player.Hand[0]
	for _, c := range player.Hand {
		if c.CardType == domain.CardTypeLand {
			land = c
			break
		}
	}
	player.Turn.Phase = PhaseMain1
	game.PlayLand(player, land)
	land.Tapped = false
	land.Active = true

	attackers := game.AvailableAttackers(player)

	for _, a := range attackers {
		if a.CardType != domain.CardTypeCreature {
			t.Errorf("Non-creature should not be available as attacker")
		}
	}
}

func TestDeclareAttacker_TapsCreature(t *testing.T) {
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
		t.Errorf("Creature should be tapped after declaring attack")
	}
}

func TestDeclareAttacker_FailsForTappedCreature(t *testing.T) {
	game, player, _ := setupCombatTest()

	creature := addCreatureToBattlefield(player, "Llanowar Elves", true, true)
	if creature == nil {
		t.Skip("Llanowar Elves not found")
	}

	err := game.DeclareAttacker(creature)
	if err == nil {
		t.Errorf("DeclareAttacker should fail for tapped creature")
	}
}

func TestDeclareAttacker_FailsForInactiveCreature(t *testing.T) {
	game, player, _ := setupCombatTest()

	creature := addCreatureToBattlefield(player, "Llanowar Elves", false, false)
	if creature == nil {
		t.Skip("Llanowar Elves not found")
	}

	err := game.DeclareAttacker(creature)
	if err == nil {
		t.Errorf("DeclareAttacker should fail for inactive creature")
	}
}

func TestAvailableBlockers_ReturnsUntappedCreatures(t *testing.T) {
	game, player, _ := setupCombatTest()

	creature := addCreatureToBattlefield(player, "Llanowar Elves", true, false)
	if creature == nil {
		t.Skip("Llanowar Elves not found")
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
		t.Errorf("Expected untapped creature to be available as blocker")
	}
}

func TestAvailableBlockers_ExcludesTappedCreatures(t *testing.T) {
	game, player, _ := setupCombatTest()

	creature := addCreatureToBattlefield(player, "Llanowar Elves", true, true)
	if creature == nil {
		t.Skip("Llanowar Elves not found")
	}

	blockers := game.AvailableBlockers(player)

	for _, b := range blockers {
		if b == creature {
			t.Errorf("Tapped creature should not be available as blocker")
		}
	}
}

func TestAvailableBlockers_AllowsSummoningSick(t *testing.T) {
	game, player, _ := setupCombatTest()

	creature := addCreatureToBattlefield(player, "Llanowar Elves", false, false)
	if creature == nil {
		t.Skip("Llanowar Elves not found")
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
		t.Errorf("Summoning sick creature should be able to block")
	}
}

func TestResolveCombatDamage_UnblockedDamageToPlayer(t *testing.T) {
	game, player, opponent := setupCombatTest()

	creature := addCreatureToBattlefield(player, "Llanowar Elves", true, false)
	if creature == nil {
		t.Skip("Llanowar Elves not found")
	}
	creature.Power = 2

	game.ActivePlayer = 0
	game.DeclareAttacker(creature)
	game.Attackers = append(game.Attackers, creature)

	initialLife := opponent.LifeTotal
	game.ResolveCombatDamage()

	if opponent.LifeTotal != initialLife-creature.Power {
		t.Errorf("Expected opponent life %d, got %d", initialLife-creature.Power, opponent.LifeTotal)
	}
}

func TestResolveCombatDamage_BlockedCreaturesFight(t *testing.T) {
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

func TestResolveCombatDamage_DeadCreaturesGoToGraveyard(t *testing.T) {
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
	blocker.Power = 1
	blocker.Toughness = 1

	game.ActivePlayer = 0
	game.DeclareAttacker(attacker)
	game.Attackers = append(game.Attackers, attacker)
	game.BlockerMap[attacker] = []*Card{blocker}

	game.ResolveCombatDamage()
	game.CheckStateBasedActions()

	blockerInGraveyard := false
	for _, c := range opponent.Graveyard {
		if c == blocker {
			blockerInGraveyard = true
			break
		}
	}

	if !blockerInGraveyard {
		t.Errorf("Dead blocker should be in graveyard")
	}
}

func TestAvailableActions_MainPhaseAllowsLandAndSorcery(t *testing.T) {
	game, player, _ := setupCombatTest()

	player.Turn.Phase = PhaseMain1
	player.Turn.LandPlayed = false

	actions := game.AvailableActions(player)

	hasPlayLand := false
	for _, a := range actions {
		if a.Type == ActionPlayLand {
			hasPlayLand = true
			break
		}
	}

	if !hasPlayLand {
		landInHand := false
		for _, c := range player.Hand {
			if c.CardType == domain.CardTypeLand {
				landInHand = true
				break
			}
		}
		if landInHand {
			t.Errorf("PlayLand action should be available in main phase")
		}
	}
}

func TestAvailableActions_CombatPhaseNoLandActions(t *testing.T) {
	game, player, _ := setupCombatTest()

	player.Turn.Phase = PhaseCombat
	player.Turn.LandPlayed = false

	actions := game.AvailableActions(player)

	for _, a := range actions {
		if a.Type == ActionPlayLand {
			t.Errorf("PlayLand action should not be available during combat phase")
		}
	}
}

func TestAvailableActions_DeclareAttackersInCombat(t *testing.T) {
	game, player, _ := setupCombatTest()

	creature := addCreatureToBattlefield(player, "Llanowar Elves", true, false)
	if creature == nil {
		t.Skip("Llanowar Elves not found")
	}

	player.Turn.Phase = PhaseCombat
	player.Turn.CombatStep = CombatStepDeclareAttackers
	game.ActivePlayer = 0

	actions := game.AvailableActions(player)

	hasDeclareAttacker := false
	for _, a := range actions {
		if a.Type == ActionDeclareAttacker {
			hasDeclareAttacker = true
			break
		}
	}

	if !hasDeclareAttacker {
		t.Errorf("DeclareAttacker action should be available during combat declare attackers step")
	}
}

func TestAvailableActions_DeclareBlockersInCombat(t *testing.T) {
	game, player, opponent := setupCombatTest()

	attacker := addCreatureToBattlefield(player, "Llanowar Elves", true, false)
	if attacker == nil {
		t.Skip("Llanowar Elves not found")
	}
	blocker := addCreatureToBattlefield(opponent, "Llanowar Elves", true, false)
	if blocker == nil {
		t.Skip("Llanowar Elves not found for blocker")
	}

	game.ActivePlayer = 0
	player.Turn.Phase = PhaseCombat
	player.Turn.CombatStep = CombatStepDeclareBlockers
	game.DeclareAttacker(attacker)
	game.Attackers = append(game.Attackers, attacker)

	actions := game.AvailableActions(opponent)

	hasDeclareBlocker := false
	for _, a := range actions {
		if a.Type == ActionDeclareBlocker {
			hasDeclareBlocker = true
			break
		}
	}

	if !hasDeclareBlocker {
		t.Errorf("DeclareBlocker action should be available for defending player during declare blockers step")
	}
}

func TestDeclareBlocker_AddsBlockerToMap(t *testing.T) {
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
		t.Errorf("DeclareBlocker returned error: %v", err)
	}

	blockers := game.BlockerMap[attacker]
	found := false
	for _, b := range blockers {
		if b == blocker {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Blocker should be in BlockerMap for the attacker")
	}
}

func TestDeclareBlocker_FailsForTappedCreature(t *testing.T) {
	game, player, opponent := setupCombatTest()

	attacker := addCreatureToBattlefield(player, "Llanowar Elves", true, false)
	if attacker == nil {
		t.Skip("Llanowar Elves not found")
	}
	blocker := addCreatureToBattlefield(opponent, "Llanowar Elves", true, true)
	if blocker == nil {
		t.Skip("Llanowar Elves not found for blocker")
	}

	err := game.DeclareBlocker(blocker, attacker)
	if err == nil {
		t.Errorf("DeclareBlocker should fail for tapped creature")
	}
}

func TestClearCombatState(t *testing.T) {
	game, player, _ := setupCombatTest()

	creature := addCreatureToBattlefield(player, "Llanowar Elves", true, false)
	if creature == nil {
		t.Skip("Llanowar Elves not found")
	}

	game.Attackers = []*Card{creature}
	game.BlockerMap[creature] = []*Card{creature}

	game.ClearCombatState()

	if len(game.Attackers) != 0 {
		t.Errorf("Attackers should be cleared")
	}

	if len(game.BlockerMap) != 0 {
		t.Errorf("BlockerMap should be cleared")
	}
}

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

func TestResolveCombatDamage_NoTrampleNoDamageToPlayerWhenBlocked(t *testing.T) {
	game, player, opponent := setupCombatTest()

	attacker := addCreatureToBattlefield(player, "Llanowar Elves", true, false)
	if attacker == nil {
		t.Skip("Llanowar Elves not found")
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

	if opponent.LifeTotal != initialLife {
		t.Errorf("Non-trample creature should not deal excess damage to player, life should be %d, got %d", initialLife, opponent.LifeTotal)
	}
}

func TestResolveCombatDamage_DamageAssignmentLethalToBlockers(t *testing.T) {
	game, player, opponent := setupCombatTest()

	attacker := addCreatureToBattlefield(player, "Llanowar Elves", true, false)
	if attacker == nil {
		t.Skip("Llanowar Elves not found")
	}
	attacker.Power = 5
	attacker.Toughness = 5

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

	game.ResolveCombatDamage()

	if blocker1.DamageTaken != 2 {
		t.Errorf("Blocker1 should receive lethal damage (%d), got %d", 2, blocker1.DamageTaken)
	}
	if blocker2.DamageTaken != 3 {
		t.Errorf("Blocker2 should receive lethal damage (%d), got %d", 3, blocker2.DamageTaken)
	}
}

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

func addSwampwalkAbility(card *Card) {
	card.ParsedAbilities = append(card.ParsedAbilities, domain.ParsedAbility{
		Type:     "Keyword",
		Keywords: []string{"Landwalk"},
		Effect:   &domain.ParsedEffect{Modifier: "swampwalk"},
	})
}

func addLandToBattlefield(player *Player, name string) *Card {
	card := &Card{
		Card: domain.Card{
			CardName: name,
			CardType: domain.CardTypeLand,
		},
		ID:          EntityID(200 + len(player.Battlefield)),
		Owner:       player,
		CurrentZone: ZoneBattlefield,
		Active:      true,
	}
	player.Battlefield = append(player.Battlefield, card)
	return card
}

func TestDeclareBlocker_LandwalkCannotBeBlockedWhenDefenderHasLand(t *testing.T) {
	game, player, opponent := setupCombatTest()

	attacker := addCreatureToBattlefield(player, "Llanowar Elves", true, false)
	if attacker == nil {
		t.Skip("Llanowar Elves not found")
	}
	addSwampwalkAbility(attacker)

	blocker := addCreatureToBattlefield(opponent, "Llanowar Elves", true, false)
	if blocker == nil {
		t.Skip("Llanowar Elves not found for blocker")
	}

	addLandToBattlefield(opponent, "Swamp")

	game.ActivePlayer = 0
	err := game.DeclareBlocker(blocker, attacker)
	if err == nil {
		t.Errorf("Creature with swampwalk should not be blockable when defender controls a Swamp")
	}
}

func TestDeclareBlocker_LandwalkCanBeBlockedWhenDefenderLacksLand(t *testing.T) {
	game, player, opponent := setupCombatTest()

	attacker := addCreatureToBattlefield(player, "Llanowar Elves", true, false)
	if attacker == nil {
		t.Skip("Llanowar Elves not found")
	}
	addSwampwalkAbility(attacker)

	blocker := addCreatureToBattlefield(opponent, "Llanowar Elves", true, false)
	if blocker == nil {
		t.Skip("Llanowar Elves not found for blocker")
	}

	game.ActivePlayer = 0
	err := game.DeclareBlocker(blocker, attacker)
	if err != nil {
		t.Errorf("Creature with swampwalk should be blockable when defender has no Swamp: %v", err)
	}
}

func TestDeclareBlocker_LandwalkDifferentLandType(t *testing.T) {
	game, player, opponent := setupCombatTest()

	attacker := addCreatureToBattlefield(player, "Llanowar Elves", true, false)
	if attacker == nil {
		t.Skip("Llanowar Elves not found")
	}
	addSwampwalkAbility(attacker)

	blocker := addCreatureToBattlefield(opponent, "Llanowar Elves", true, false)
	if blocker == nil {
		t.Skip("Llanowar Elves not found for blocker")
	}

	addLandToBattlefield(opponent, "Forest")

	game.ActivePlayer = 0
	err := game.DeclareBlocker(blocker, attacker)
	if err != nil {
		t.Errorf("Creature with swampwalk should be blockable when defender only has Forest: %v", err)
	}
}

func TestAvailableActions_LandwalkFiltersBlockers(t *testing.T) {
	game, player, opponent := setupCombatTest()

	attacker := addCreatureToBattlefield(player, "Llanowar Elves", true, false)
	if attacker == nil {
		t.Skip("Llanowar Elves not found")
	}
	addSwampwalkAbility(attacker)

	blocker := addCreatureToBattlefield(opponent, "Llanowar Elves", true, false)
	if blocker == nil {
		t.Skip("Llanowar Elves not found for blocker")
	}

	addLandToBattlefield(opponent, "Swamp")

	game.ActivePlayer = 0
	player.Turn.Phase = PhaseCombat
	player.Turn.CombatStep = CombatStepDeclareBlockers
	game.DeclareAttacker(attacker)
	game.Attackers = append(game.Attackers, attacker)

	actions := game.AvailableActions(opponent)

	for _, a := range actions {
		if a.Type == ActionDeclareBlocker && a.Target == attacker {
			t.Errorf("Blocker actions should not be available against creature with swampwalk when defender controls a Swamp")
		}
	}
}

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
