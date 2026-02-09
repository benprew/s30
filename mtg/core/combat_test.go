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
