package core_engine

import (
	"fmt"
	"sync"
	"testing"
)

func createTestPlayer(numPlayers int) []*Player {
	players := []*Player{}

	for i := range numPlayers {
		library := []*Card{}
		for range 5 {
			cardName := "Forest"
			card, ok := CardDatabase[cardName]
			if !ok {
				panic(fmt.Sprintf("Card not found: %s", cardName))
			}
			library = append(library, card)
		}
		for range 2 {
			cardName := "Llanowar Elves"
			card, ok := CardDatabase[cardName]
			if !ok {
				panic(fmt.Sprintf("Card not found: %s", cardName))
			}
			library = append(library, card)
		}

		player := &Player{
			ID:          EntityID(i),
			LifeTotal:   20,
			ManaPool:    ManaPool{},
			Hand:        []*Card{},
			Library:     library,
			Battlefield: []*Card{},
			Graveyard:   []*Card{},
			Exile:       []*Card{},
			Turn:        &Turn{},
			InputChan:   make(chan PlayerAction, 100), // Still need a channel even for AI, as WaitForPlayerInput uses it
			IsAI:        true,                         // Make test players AI so WaitForPlayerInput doesn't block indefinitely
		}
		players = append(players, player)
	}

	return players
}

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
		if len(player.Library) != 0 {
			t.Errorf("Player %d should have 0 cards in library, but has %d", i+1, len(player.Library))
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
		if game.Players[0].Hand[i].CardType == CardTypeLand {
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
	for _, card := range player.Hand {
		if card.CardType == "Land" {
			if err := game.PlayLand(player, card); err != nil {
				t.Errorf("Failed to play land: %v", err)
				return
			}
			break
		}
	}

	var pPool ManaPool
	copy(pPool, game.Players[0].ManaPool)
	fmt.Println(game.AvailableMana(game.Players[0], pPool))

	// find elves
	var card *Card
	for _, c := range player.Hand {
		if c.Name == "Llanowar Elves" {
			card = c
			break
		}
	}

	// need to tap the land first
	if !game.CanCast(player, card) {
		t.Errorf("Player should be able to cast Llanowar Elves, but can't: %+v", player.ManaPool)
	}

	// Cast the card
	if err := game.CastCreature(player, card); err != nil {
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

func TestRunStack(t *testing.T) {
	// Test the next turn functionality with 1 player, make sure the player
	// has the opportunity to respond in each phase
	players := createTestPlayer(2)
	player := players[0]
	player2 := players[1]
	game := NewGame(players)

	// Check that the player had an opportunity to respond in each phase
	expectedPhases := []Phase{
		PhaseUpkeep,
		PhaseDraw,
		PhaseMain1,
		PhaseCombat,
		PhaseMain2,
		PhaseEnd,
	}
	var wg sync.WaitGroup
	wg.Add(1)

	// Start a goroutine to simulate player responses
	go func() {
		defer wg.Done()
		// For each expected phase, send a PassPriority action
		for range expectedPhases {
			fmt.Println("player2 Passing Priority")
			player.InputChan <- PlayerAction{Type: "PassPriority"}
			player2.InputChan <- PlayerAction{Type: "PassPriority"}
		}
	}()

	// Start a turn
	player.Turn.Phase = PhaseUntap
	game.RunStack()
	wg.Wait()
	fmt.Println("waitgroup finished")
}
