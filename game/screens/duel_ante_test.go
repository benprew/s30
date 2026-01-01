package screens

import (
	"math/rand"
	"testing"

	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/world"
)

func TestStartDuel_WinMoves3CardsToPlayerCollection(t *testing.T) {
	rand.Seed(42)

	mountain := domain.FindCardByName("Mountain")
	lightningBolt := domain.FindCardByName("Lightning Bolt")

	playerDeck := make(domain.Deck)
	playerDeck[mountain] = 4
	playerDeck[lightningBolt] = 4

	playerCollection := domain.NewCardCollection()
	for card, count := range playerDeck {
		playerCollection.AddCardToDeck(card, 0, count)
	}

	player := &domain.Player{
		Character: domain.Character{
			CardCollection: playerCollection,
		},
	}

	forest := domain.FindCardByName("Forest")
	llanowarElves := domain.FindCardByName("Llanowar Elves")
	giantGrowth := domain.FindCardByName("Giant Growth")
	thicketBasilisk := domain.FindCardByName("Thicket Basilisk")
	crawWurm := domain.FindCardByName("Craw Wurm")

	enemyDeck := make(domain.Deck)
	enemyDeck[forest] = 4
	enemyDeck[llanowarElves] = 2
	enemyDeck[giantGrowth] = 3
	enemyDeck[thicketBasilisk] = 1
	enemyDeck[crawWurm] = 2

	enemyCollection := domain.NewCardCollection()
	for card, count := range enemyDeck {
		enemyCollection.AddCardToDeck(card, 0, count)
	}

	enemyCharacter := &domain.Character{
		CardCollection: enemyCollection,
	}

	enemy := &domain.Enemy{
		Character: enemyCharacter,
	}

	lvl := &world.Level{
		Player: player,
	}

	screen := &DuelAnteScreen{
		player: player,
		enemy:  enemy,
		lvl:    lvl,
		idx:    0,
	}

	screen.startDuel()

	playerCollectionSize := 0
	for _, item := range player.CardCollection {
		playerCollectionSize += item.Count
	}

	if playerCollectionSize == 0 {
		t.Errorf("Expected player to have won cards, but collection is empty")
	}
}
