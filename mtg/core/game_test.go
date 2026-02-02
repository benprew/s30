package core

import (
	"testing"

	"github.com/benprew/s30/game/domain"
)

func TestNewGame(t *testing.T) {
	// Create a new game
	players := createTestPlayer(2)
	game := NewGame(players)
	game.StartGame()

	// Check player state
	for i, player := range game.Players {
		if len(player.Hand) != 7 {
			t.Errorf("Player %d should have 7 cards in hand, but has %d", i+1, len(player.Hand))
		}
		if len(player.Library) != 1 {
			t.Errorf("Player %d should have 1 card in library, but has %d", i+1, len(player.Library))
		}
		if len(player.Graveyard) != 0 {
			t.Errorf("Player %d should have 0 cards in graveyard, but has %d", i+1, len(player.Library))
		}
	}
}

func TestPlayLand(t *testing.T) {
	// Create a new game
	players := createTestPlayer(2)

	game := NewGame(players)
	game.StartGame()
	var card *Card

	// Play the land
	game.Players[0].Turn.Phase = PhaseMain1
	for i := range game.Players[0].Hand {
		if game.Players[0].Hand[i].CardType == domain.CardTypeLand {
			card = game.Players[0].Hand[i]
			break
		}
	}

	if card == nil {
		t.Errorf("No land card found in hand")
		return
	}

	if err := game.PlayLand(game.Players[0], card); err != nil {
		t.Errorf("Failed to play land: %v", err)
		return
	}

	// Check that the land is on the battlefield
	if len(game.Players[0].Battlefield) != 1 {
		t.Errorf("Player 0 should have 1 card on the battlefield, but has %d", len(game.Players[0].Battlefield))
	}

	// Check that the land is no longer in the hand
	if len(game.Players[0].Hand) != 6 {
		t.Errorf("Player 0 should have 6 cards in hand, but has %d", len(game.Players[0].Hand))
	}

	// Check that the player cannot play any more lands
	for _, card := range game.Players[0].Hand {
		if game.CanPlayLand(game.Players[0], card) {
			t.Errorf("Player 0 should not be able to play any more lands")
		}
	}

	// Check that the player still has 0 cards in hand
	if len(game.Players[0].Hand) != 6 {
		t.Errorf("Player 0 should still have 6 cards in hand, but has %d", len(game.Players[0].Hand))
	}
}

func TestCastLlanowarElves(t *testing.T) {
	// Create a new game
	players := createTestPlayer(2)

	game := NewGame(players)
	game.StartGame()

	player := game.Players[0]

	// Play land
	game.Players[0].Turn.Phase = PhaseMain1
	var landCard *Card
	for _, card := range player.Hand {
		if card.CardType == domain.CardTypeLand {
			landCard = card
			if err := game.PlayLand(player, card); err != nil {
				t.Errorf("Failed to play land: %v", err)
				return
			}
			break
		}
	}

	if landCard == nil {
		t.Errorf("No land card found in hand")
		return
	}

	// find elves
	var card *Card
	for _, c := range player.Hand {
		if c.Name() == "Llanowar Elves" {
			card = c
			break
		}
	}

	if card == nil {
		t.Errorf("No Llanowar Elves found in hand")
		return
	}

	// Tap the land for mana first
	if err := game.ActivateManaAbility(player, landCard); err != nil {
		t.Errorf("Failed to tap land for mana: %v", err)
		return
	}

	// Check if we can cast after tapping for mana
	if !game.CanCast(player, card) {
		t.Errorf("Player should be able to cast Llanowar Elves after tapping land, but can't: %+v", player.ManaPool)
		return
	}

	// Cast the card
	if err := game.CastSpell(player, card, nil); err != nil {
		t.Errorf("Failed to cast Llanowar Elves: %v", err)
	}
}

func TestDrawCard(t *testing.T) {
	players := createTestPlayer(1)
	player := players[0]

	initialLibrarySize := len(player.Library)
	initialHandSize := len(player.Hand)

	err := player.DrawCard()
	if err != nil {
		t.Errorf("DrawCard() returned an error: %v", err)
	}

	if len(player.Library) != initialLibrarySize-1 {
		t.Errorf("Library size should be %d, but is %d", initialLibrarySize-1, len(player.Library))
	}

	if len(player.Hand) != initialHandSize+1 {
		t.Errorf("Hand size should be %d, but is %d", initialHandSize+1, len(player.Hand))
	}

	// Draw cards until the library is empty
	for range initialLibrarySize - 1 {
		err = player.DrawCard()
		if err != nil {
			t.Errorf("DrawCard() returned an error: %v", err)
		}
	}

	// Try to draw a card from an empty library
	err = player.DrawCard()
	if err == nil {
		t.Errorf("DrawCard() should have returned an error, but didn't")
	}
}

func TestCastLightningBolt(t *testing.T) {
	// Create a new game
	players := createTestPlayer(2)

	game := NewGame(players)
	game.StartGame()

	player := players[0]
	mtn := game.FindCard(FindArgs{Name: "Mountain"}, player.Hand)

	game.PlayLand(player, mtn)
	elf := game.FindCard(FindArgs{Name: "Llanowar Elves"}, player.Hand)
	if err := player.MoveTo(elf, ZoneBattlefield); err != nil {
		t.Errorf("Unable to move Elves: %v", err)
	}

	bolt := game.FindCard(FindArgs{Name: "Lightning Bolt"}, player.Hand)
	if bolt == nil {
		t.Errorf("Failed to find card")
	}

	if elf == nil {
		t.Errorf("Failed to find Llanowar Elves")
		return
	}

	if err := game.ActivateManaAbility(player, mtn); err != nil {
		panic(err)
	}

	err := game.CastSpell(player, bolt, elf)
	if err != nil {
		t.Errorf("Failed to cast spell: %v", err)
	}
	if err := game.Resolve(game.Stack.Pop()); err != nil {
		t.Errorf("Failed to do resolve: %v", err)
	}
	game.CheckStateBasedActions()

	if !mtn.Tapped {
		t.Errorf("Mountain should be tapped after casting Lightning Bolt")
	}

	if game.FindCard(FindArgs{ID: elf.ID}, player.Graveyard) == nil {
		t.Errorf("Elves not in Graveyard")
	}
}

func TestCastArtifact(t *testing.T) {
	// Create a new game
	players := createTestPlayer(2)

	game := NewGame(players)
	game.StartGame()

	player := players[0]

	// Play land to get mana
	game.Players[0].Turn.Phase = PhaseMain1
	landCard := game.FindCard(FindArgs{Name: "Mountain"}, player.Hand) // Assuming Mountain exists
	if landCard == nil {
		landCard = game.FindCard(FindArgs{Name: "Forest"}, player.Hand) // Or Forest
	}
	if landCard == nil {
		t.Skip("Skipping TestCastArtifact: No basic land found in hand")
		return
	}
	if err := game.PlayLand(player, landCard); err != nil {
		t.Errorf("Failed to play land: %v", err)
		return
	}

	// Tap the land for mana
	if err := game.ActivateManaAbility(player, landCard); err != nil {
		t.Errorf("Failed to tap land for mana: %v", err)
		return
	}

	// Find the artifact card (assuming "Sol Ring" exists)
	artifactCard := game.FindCard(FindArgs{Name: "Sol Ring"}, player.Hand)
	if artifactCard == nil {
		t.Errorf("Skipping TestCastArtifact: 'Sol Ring' not found in hand. Add 'Sol Ring' to the test player's library.")
		return
	}

	initialHandSize := len(player.Hand)
	initialBattlefieldSize := len(player.Battlefield)

	if !game.CanCast(player, artifactCard) {
		t.Errorf("Player should be able to cast %s after tapping land, but cannot. Mana Pool: %+v", artifactCard.Name(), player.ManaPool)
		return
	}

	if err := game.CastSpell(player, artifactCard, nil); err != nil {
		t.Errorf("Failed to cast %s: %v", artifactCard.Name(), err)
		return
	}

	if err := game.Resolve(game.Stack.Pop()); err != nil {
		t.Errorf("Failed to do resolve: %v", err)
	}

	// Check that the artifact is on the battlefield using FindCard
	foundOnBattlefield := game.FindCard(FindArgs{ID: artifactCard.ID}, player.Battlefield)
	if foundOnBattlefield == nil {
		t.Errorf("%s not found on the battlefield after casting", artifactCard.Name())
	}
	if len(player.Battlefield) != initialBattlefieldSize+1 {
		t.Errorf("Player battlefield size should be %d, but is %d", initialBattlefieldSize+1, len(player.Battlefield))
	}

	// Check that the artifact is no longer in the hand
	foundInHand := game.FindCard(FindArgs{ID: artifactCard.ID}, player.Hand)
	if foundInHand != nil {
		t.Errorf("%s still found in hand after casting", artifactCard.Name())
	}
	if len(player.Hand) != initialHandSize-1 {
		t.Errorf("Player hand size should be %d, but is %d", initialHandSize-1, len(player.Hand))
	}
}

func TestCheckWinConditions(t *testing.T) {
	players := createTestPlayer(2)
	game := NewGame(players)

	game.CheckWinConditions()
	for _, p := range players {
		if p.HasLost {
			t.Errorf("Player should not have lost at 20 life")
		}
	}

	players[0].LifeTotal = 0
	game.CheckWinConditions()
	if !players[0].HasLost {
		t.Errorf("Player 0 should have lost at 0 life")
	}
	if players[1].HasLost {
		t.Errorf("Player 1 should not have lost")
	}
}

func TestCheckWinConditionsNegativeLife(t *testing.T) {
	players := createTestPlayer(2)
	game := NewGame(players)

	players[0].LifeTotal = -10
	game.CheckWinConditions()
	if !players[0].HasLost {
		t.Errorf("Player should have lost at negative life")
	}
}

func TestCheckWinConditionsMultipleLosers(t *testing.T) {
	players := createTestPlayer(2)
	game := NewGame(players)

	players[0].LifeTotal = 0
	players[1].LifeTotal = -5
	game.CheckWinConditions()

	if !players[0].HasLost {
		t.Errorf("Player 0 should have lost")
	}
	if !players[1].HasLost {
		t.Errorf("Player 1 should have lost")
	}
}

func TestNextPhase(t *testing.T) {
	turn := &Turn{Phase: PhaseUntap}

	expectedOrder := []Phase{
		PhaseUpkeep,
		PhaseDraw,
		PhaseMain1,
		PhaseCombat,
		PhaseMain2,
		PhaseEnd,
		PhaseCleanup,
		PhaseUntap,
	}

	for _, expected := range expectedOrder {
		turn.NextPhase()
		if turn.Phase != expected {
			t.Errorf("Expected phase %s, got %s", expected, turn.Phase)
		}
	}
}

func TestTurnBasedPlayerSwitching(t *testing.T) {
	players := createTestPlayer(2)
	game := NewGame(players)

	if game.CurrentPlayer != 0 {
		t.Errorf("Expected CurrentPlayer=0, got %d", game.CurrentPlayer)
	}

	players[0].Turn.Phase = PhaseEnd
	game.CurrentPlayer = (game.CurrentPlayer + 1) % len(game.Players)

	if game.CurrentPlayer != 1 {
		t.Errorf("Expected CurrentPlayer=1 after switch, got %d", game.CurrentPlayer)
	}

	game.CurrentPlayer = (game.CurrentPlayer + 1) % len(game.Players)
	if game.CurrentPlayer != 0 {
		t.Errorf("Expected CurrentPlayer=0 after wraparound, got %d", game.CurrentPlayer)
	}
}

func TestDrawFromEmptyLibraryLoses(t *testing.T) {
	players := createTestPlayer(1)
	player := players[0]
	game := NewGame(players)

	player.Library = []*Card{}
	player.Turn.Phase = PhaseDraw

	game.DrawPhase(player)

	if !player.HasLost {
		t.Errorf("Player should have lost after drawing from empty library")
	}
}

func TestPlayLandWhenLandPlayed(t *testing.T) {
	players := createTestPlayer(1)
	player := players[0]
	game := NewGame(players)
	game.StartGame()

	player.Turn.Phase = PhaseMain1
	player.Turn.LandPlayed = true

	for _, card := range player.Hand {
		if card.CardType == domain.CardTypeLand {
			if game.CanPlayLand(player, card) {
				t.Errorf("Should not be able to play land when LandPlayed=true")
			}
			break
		}
	}
}

func TestPlayLandWrongPhase(t *testing.T) {
	players := createTestPlayer(1)
	player := players[0]
	game := NewGame(players)
	game.StartGame()

	player.Turn.Phase = PhaseCombat
	player.Turn.LandPlayed = false

	for _, card := range player.Hand {
		if card.CardType == domain.CardTypeLand {
			if game.CanPlayLand(player, card) {
				t.Errorf("Should not be able to play land during combat phase")
			}
			break
		}
	}
}

func TestAvailableActionsPlayLandMainPhase(t *testing.T) {
	players := createTestPlayer(1)
	player := players[0]
	game := NewGame(players)
	game.StartGame()

	player.Turn.Phase = PhaseMain1
	player.Turn.LandPlayed = false

	actions := game.AvailableActions(player)

	landCount := 0
	for _, card := range player.Hand {
		if card.CardType == domain.CardTypeLand {
			landCount++
		}
	}

	playLandCount := 0
	for _, action := range actions {
		if action.Type == ActionPlayLand {
			playLandCount++
		}
	}

	if playLandCount != landCount {
		t.Errorf("Expected %d PlayLand actions, got %d", landCount, playLandCount)
	}
}

func TestAvailableActionsNoPlayLandAfterLandPlayed(t *testing.T) {
	players := createTestPlayer(1)
	player := players[0]
	game := NewGame(players)
	game.StartGame()

	player.Turn.Phase = PhaseMain1
	player.Turn.LandPlayed = true

	actions := game.AvailableActions(player)

	for _, action := range actions {
		if action.Type == ActionPlayLand {
			t.Errorf("Should not have PlayLand action when land already played")
		}
	}
}

func TestAvailableActionsNoPlayLandWrongPhase(t *testing.T) {
	players := createTestPlayer(1)
	player := players[0]
	game := NewGame(players)
	game.StartGame()

	player.Turn.Phase = PhaseUntap
	player.Turn.LandPlayed = false

	actions := game.AvailableActions(player)

	for _, action := range actions {
		if action.Type == ActionPlayLand {
			t.Errorf("Should not have PlayLand action during untap phase")
		}
	}
}

func TestAvailableActionsCastSpellWithMana(t *testing.T) {
	players := createTestPlayer(1)
	player := players[0]
	game := NewGame(players)
	game.StartGame()

	player.Turn.Phase = PhaseMain1

	landCard := game.FindCard(FindArgs{Name: "Forest"}, player.Hand)
	if landCard == nil {
		landCard = game.FindCard(FindArgs{Name: "Mountain"}, player.Hand)
	}
	if landCard == nil {
		t.Skip("No land card found in hand")
	}

	game.PlayLand(player, landCard)
	if err := game.ActivateManaAbility(player, landCard); err != nil {
		panic(err)
	}

	actions := game.AvailableActions(player)

	hasCastSpell := false
	for _, action := range actions {
		if action.Type == ActionCastSpell {
			hasCastSpell = true
			break
		}
	}

	if !hasCastSpell {
		t.Errorf("Expected CastSpell action when player has mana")
	}
}

func TestAvailableActionsNoCastSpellWithoutMana(t *testing.T) {
	players := createTestPlayer(1)
	player := players[0]
	game := NewGame(players)
	game.StartGame()

	player.Turn.Phase = PhaseMain1
	player.ManaPool = ManaPool{}

	actions := game.AvailableActions(player)

	for _, action := range actions {
		if action.Type == ActionCastSpell && len(action.Card.ManaCost) > 0 {
			t.Errorf("Should not have CastSpell action for non-zero cost card %s without mana", action.Card.Name())
		}
	}
}

func TestAvailableActionsPassPriorityAlways(t *testing.T) {
	players := createTestPlayer(1)
	player := players[0]
	game := NewGame(players)
	game.StartGame()

	phases := []Phase{PhaseUntap, PhaseUpkeep, PhaseDraw, PhaseMain1, PhaseCombat, PhaseMain2, PhaseEnd}

	for _, phase := range phases {
		player.Turn.Phase = phase
		actions := game.AvailableActions(player)

		hasPassPriority := false
		for _, action := range actions {
			if action.Type == ActionPassPriority {
				hasPassPriority = true
				break
			}
		}

		if !hasPassPriority {
			t.Errorf("PassPriority should always be available, missing in phase %s", phase)
		}
	}
}

func TestAvailableTargetsLightningBolt(t *testing.T) {
	players := createTestPlayer(2)
	game := NewGame(players)
	game.StartGame()

	bolt := game.FindCard(FindArgs{Name: "Lightning Bolt"}, players[0].Hand)
	if bolt == nil {
		t.Skip("No Lightning Bolt in hand")
	}

	targets := game.AvailableTargets(bolt)
	if len(targets) != 2 {
		t.Errorf("Expected 2 targets (both players), got %d", len(targets))
	}

	playerCount := 0
	for _, target := range targets {
		if target.TargetType() == TargetTypePlayer {
			playerCount++
		}
	}
	if playerCount != 2 {
		t.Errorf("Expected 2 player targets, got %d", playerCount)
	}
}

func TestAvailableTargetsWithCreature(t *testing.T) {
	players := createTestPlayer(2)
	game := NewGame(players)
	game.StartGame()

	elf := game.FindCard(FindArgs{Name: "Llanowar Elves"}, players[0].Hand)
	if elf == nil {
		t.Skip("No Llanowar Elves in hand")
	}
	players[0].MoveTo(elf, ZoneBattlefield)

	bolt := game.FindCard(FindArgs{Name: "Lightning Bolt"}, players[0].Hand)
	if bolt == nil {
		t.Skip("No Lightning Bolt in hand")
	}

	targets := game.AvailableTargets(bolt)
	if len(targets) != 3 {
		t.Errorf("Expected 3 targets (2 players + 1 creature), got %d", len(targets))
	}

	creatureCount := 0
	for _, target := range targets {
		if target.TargetType() == TargetTypeCard {
			creatureCount++
		}
	}
	if creatureCount != 1 {
		t.Errorf("Expected 1 creature target, got %d", creatureCount)
	}
}

func TestCombatPhaseCastSpellAction(t *testing.T) {
	players := createTestPlayer(2)
	game := NewGame(players)
	game.StartGame()

	player := players[0]
	opponent := players[1]

	elf := game.FindCard(FindArgs{Name: "Llanowar Elves"}, player.Hand)
	if elf == nil {
		t.Skip("No Llanowar Elves in hand")
	}
	player.MoveTo(elf, ZoneBattlefield)
	elf.Active = true

	forest := game.FindCard(FindArgs{Name: "Forest"}, player.Hand)
	if forest == nil {
		t.Skip("No Forest in hand")
	}
	player.MoveTo(forest, ZoneBattlefield)

	giantGrowth := game.FindCard(FindArgs{Name: "Giant Growth"}, player.Library)
	if giantGrowth == nil {
		t.Skip("No Giant Growth in library")
	}
	player.DrawCard()
	for game.FindCard(FindArgs{Name: "Giant Growth"}, player.Hand) == nil && len(player.Library) > 0 {
		player.DrawCard()
	}
	giantGrowth = game.FindCard(FindArgs{Name: "Giant Growth"}, player.Hand)
	if giantGrowth == nil {
		t.Skip("Could not draw Giant Growth")
	}

	player.Turn.Phase = PhaseCombat
	player.Turn.CombatStep = CombatStepDeclareAttackers
	game.ActivePlayer = 0
	game.CurrentPlayer = 0
	game.Attackers = []*Card{elf}

	initialHandSize := len(player.Hand)

	go func() {
		player.InputChan <- PlayerAction{
			Type:   ActionCastSpell,
			Card:   giantGrowth,
			Target: elf,
		}
		player.InputChan <- PlayerAction{Type: ActionPassPriority}
		opponent.InputChan <- PlayerAction{Type: ActionPassPriority}
	}()

	game.RunStack()

	if len(player.Hand) != initialHandSize-1 {
		t.Errorf("Giant Growth was NOT cast during combat. Expected hand size %d, got %d", initialHandSize-1, len(player.Hand))
	}

	if elf.EffectivePower() != 4 {
		t.Errorf("Expected Llanowar Elves to have 4 power after Giant Growth, got %d", elf.EffectivePower())
	}
}
