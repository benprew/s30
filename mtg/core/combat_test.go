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

func TestCombatPhase_NoAttackersSkipsBlockersAndDamage(t *testing.T) {
	game, player, opponent := setupCombatTest()
	game.ActivePlayer = 0
	game.CurrentPlayer = 0

	addCreatureToBattlefield(player, "Llanowar Elves", true, false)
	addCreatureToBattlefield(opponent, "Llanowar Elves", true, false)

	initialLife := opponent.LifeTotal

	// Feed pass actions for each WaitForPlayerInput call:
	// BeginningOfCombat RunStack: player pass, opponent pass
	// DeclareAttackers loop: player pass (no attackers declared)
	// DeclareAttackers RunStack: player pass, opponent pass
	// EndOfCombat RunStack: player pass, opponent pass
	// If blockers/damage were NOT skipped, the test would deadlock
	// waiting for additional input.
	passAll := func(p *Player, n int) <-chan struct{} {
		done := make(chan struct{})
		go func() {
			defer close(done)
			for range n {
				<-p.WaitingChan
				p.InputChan <- PlayerAction{Type: ActionPassPriority}
			}
		}()
		return done
	}
	playerDone := passAll(player, 4)
	opponentDone := passAll(opponent, 3)

	game.CombatPhase(player)

	<-playerDone
	<-opponentDone

	if opponent.LifeTotal != initialLife {
		t.Errorf("Opponent should take no damage when no attackers declared, expected life %d, got %d", initialLife, opponent.LifeTotal)
	}

	if len(game.Attackers) != 0 {
		t.Errorf("Attackers should be cleared after combat, got %d", len(game.Attackers))
	}

	if len(game.BlockerMap) != 0 {
		t.Errorf("BlockerMap should be cleared after combat, got %d entries", len(game.BlockerMap))
	}

	if player.Turn.CombatStep != CombatStepNone {
		t.Errorf("CombatStep should be None after combat, got %s", player.Turn.CombatStep)
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

func TestAvailableAttackers_ExcludesDeclaredVigilanceAttacker(t *testing.T) {
	game, player, _ := setupCombatTest()

	creature := addCreatureToBattlefield(player, "Serra Angel", true, false)
	if creature == nil {
		t.Skip("Serra Angel not found")
	}

	err := game.DeclareAttacker(creature)
	if err != nil {
		t.Fatalf("DeclareAttacker returned error: %v", err)
	}
	game.Attackers = append(game.Attackers, creature)

	if creature.Tapped {
		t.Errorf("Vigilance creature should not be tapped after declaring attack")
	}

	attackers := game.AvailableAttackers(player)
	for _, a := range attackers {
		if a == creature {
			t.Errorf("Declared vigilance attacker should not appear in available attackers")
		}
	}
}
