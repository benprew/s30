package core_engine

import (
    "fmt"
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

func TestRemoveFrom(t *testing.T) {
    player := createTestPlayer(1)[0]

    for range 7 {
        player.DrawCard()
    }

    card := player.Hand[0]
    player.RemoveFrom(card, player.Hand, "Hand")

    if len(player.Hand) != 6 {
        t.Errorf("Card %v not removed from hand: %v", card, player.Hand)
    }
}

func TestAddTo(t *testing.T) {
    player := createTestPlayer(1)[0]

    card := player.Library[0]
    player.AddTo(card, "Hand")

    if len(player.Hand) != 1 {
        t.Errorf("Card %v not added to hand: %v", card, player.Hand)
    }
}
