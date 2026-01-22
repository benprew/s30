package core_engine

import (
	"fmt"
	"testing"

	"github.com/benprew/s30/game/domain"
)

func createTestPlayer(numPlayers int) []*Player {
	players := []*Player{}
	entityID := EntityID(1)

	for i := range numPlayers {
		player := &Player{
			ID:          EntityID(i),
			LifeTotal:   20,
			ManaPool:    ManaPool{},
			Hand:        []*Card{},
			Library:     []*Card{},
			Battlefield: []*Card{},
			Graveyard:   []*Card{},
			Exile:       []*Card{},
			Turn:        &Turn{},
			InputChan:   make(chan PlayerAction, 100),
			IsAI:        true,
		}

		addCard := func(name string) {
			domainCard := domain.FindCardByName(name)
			if domainCard != nil {
				coreCard := NewCardFromDomain(domainCard, entityID, player)
				player.Library = append(player.Library, coreCard)
				entityID++
			}
		}

		addCard("Forest")
		addCard("Forest")
		addCard("Llanowar Elves")
		addCard("Llanowar Elves")
		addCard("Lightning Bolt")
		addCard("Mountain")
		addCard("Sol Ring")

		fmt.Println("Library")
		for _, c := range player.Library {
			fmt.Println(c)
		}

		players = append(players, player)
	}

	return players
}

func TestMoveTo(t *testing.T) {
	player := createTestPlayer(1)[0]

	for range 7 {
		player.DrawCard()
	}

	card := player.Hand[0]
	err := player.MoveTo(card, ZoneGraveyard)
	if err != nil {
		t.Errorf("MoveTo failed: %v", err)
	}

	if len(player.Hand) != 6 {
		t.Errorf("Card not removed from hand, expected 6 got %d", len(player.Hand))
	}

	if len(player.Graveyard) != 1 {
		t.Errorf("Card not added to graveyard, expected 1 got %d", len(player.Graveyard))
	}

	if card.CurrentZone != ZoneGraveyard {
		t.Errorf("Card zone not updated, expected %d got %d", ZoneGraveyard, card.CurrentZone)
	}
}

func TestMoveToWrongOwner(t *testing.T) {
	players := createTestPlayer(2)
	player1 := players[0]
	player2 := players[1]

	player1.DrawCard()
	card := player1.Hand[0]

	err := player2.MoveTo(card, ZoneGraveyard)
	if err == nil {
		t.Errorf("Expected error when moving card owned by different player")
	}
}
