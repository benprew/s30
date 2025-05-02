package rules

import (
	"testing"
)

func createTestPlayer(numPlayers int) []*Player {
	players := []*Player{}

	for range numPlayers {
		library := []*Card{}
		for range 5 {
			cardName := "Forest"
			card, ok := CardDatabase[cardName]
			if !ok {
				panic("Card not found: %s" + cardName)
			}
			library = append(library, card)
		}
		for range 2 {
			cardName := "Llanowar Elves"
			card, ok := CardDatabase[cardName]
			if !ok {
				panic("Card not found: %s" + cardName)
			}
			library = append(library, card)
		}

		player := &Player{
			LifeTotal:   20,
			ManaPool:    ManaPool{},
			Hand:        []*Card{},
			Library:     library,
			Battlefield: []*Card{},
			Graveyard:   []*Card{},
			Exile:       []*Card{},
			Turn:        &Turn{},
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
		if game.Players[0].Hand[i].CardType == "Land" {
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
	if game.CanPlayLand(game.Players[0]) {
		t.Errorf("Player 0 should not be able to play any more lands")
	}
	game.PlayLand(game.Players[0], card)
	if len(game.Players[0].Battlefield) != 1 {
		t.Errorf("Player 0 should still have 1 card on the battlefield, but has %d", len(game.Players[0].Battlefield))
	}

	// Check that the player still has 0 cards in hand
	if len(game.Players[0].Hand) != 6 {
		t.Errorf("Player 0 should still have 6 cards in hand, but has %d", len(game.Players[0].Hand))
	}

	// Check that the player cannot play any more lands
	game.PlayLand(game.Players[0], card)
	if len(game.Players[0].Battlefield) != 1 {
		t.Errorf("Player 0 should still have 1 card on the battlefield, but has %d", len(game.Players[0].Battlefield))
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
	if err := game.CastCard(player, card); err != nil {
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
	for i := 0; i < initialLibrarySize-1; i++ {
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
