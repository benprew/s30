package core

import (
	"testing"

	"github.com/benprew/s30/game/domain"
)

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
