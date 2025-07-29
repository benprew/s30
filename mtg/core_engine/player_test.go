package core_engine

import (
	"fmt"
	"testing"

	"github.com/benprew/s30/game/domain"
)

func createTestPlayer(numPlayers int) []*Player {
	players := []*Player{}
	entityID := EntityID(1) // Start with ID 1 and increment for each card

	for i := range numPlayers {
		library := []*Card{}

		// Add 2 Forest cards
		for range 2 {
			domainCard := domain.FindCardByName("Forest")
			fmt.Println(domainCard)
			if domainCard != nil {
				coreCard := NewCardFromDomain(domainCard, entityID)
				library = append(library, coreCard)
				entityID++
			}
		}

		// Add 2 Llanowar Elves cards
		for range 2 {
			domainCard := domain.FindCardByName("Llanowar Elves")
			fmt.Println(domainCard)
			if domainCard != nil {
				coreCard := NewCardFromDomain(domainCard, entityID)
				library = append(library, coreCard)
				entityID++
			}
		}

		// Add Lightning Bolt
		domainCard := domain.FindCardByName("Lightning Bolt")
		if domainCard != nil {
			coreCard := NewCardFromDomain(domainCard, entityID)
			library = append(library, coreCard)
			entityID++
		}

		// Add Mountain
		domainCard = domain.FindCardByName("Mountain")
		if domainCard != nil {
			coreCard := NewCardFromDomain(domainCard, entityID)
			library = append(library, coreCard)
			entityID++
		}

		// Add Sol Ring
		domainCard = domain.FindCardByName("Sol Ring")
		if domainCard != nil {
			coreCard := NewCardFromDomain(domainCard, entityID)
			library = append(library, coreCard)
			entityID++
		}

		fmt.Println("Library")
		for _, c := range library {
			fmt.Println(c)
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
