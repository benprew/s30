package screens

import (
	"testing"

	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/world"
)

func TestHandleWin_RewardsFromCorrectEnemy(t *testing.T) {
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
		Amulets: make(map[domain.ColorMask]int),
	}

	forest := domain.FindCardByName("Forest")
	giantGrowth := domain.FindCardByName("Giant Growth")

	enemy0Deck := make(domain.Deck)
	enemy0Deck[forest] = 4
	enemy0Deck[giantGrowth] = 4

	enemy0Collection := domain.NewCardCollection()
	for card, count := range enemy0Deck {
		enemy0Collection.AddCardToDeck(card, 0, count)
	}

	island := domain.FindCardByName("Island")
	counterspell := domain.FindCardByName("Counterspell")

	enemy1Deck := make(domain.Deck)
	enemy1Deck[island] = 4
	enemy1Deck[counterspell] = 4

	enemy1Collection := domain.NewCardCollection()
	for card, count := range enemy1Deck {
		enemy1Collection.AddCardToDeck(card, 0, count)
	}

	lvl := &world.Level{
		Player: player,
	}

	lvl.SetEnemies([]domain.Enemy{
		{Character: &domain.Character{Name: "Green Mage", Level: 1, CardCollection: enemy0Collection}},
		{Character: &domain.Character{Name: "Blue Mage", Level: 1, CardCollection: enemy1Collection}},
	})

	enemy := lvl.GetEnemyAt(0)

	s := &DuelScreen{
		player:   player,
		enemy:    enemy,
		lvl:      lvl,
		idx:      0,
		anteCard: mountain,
	}

	_, screen, err := s.handleWin()
	if err != nil {
		t.Fatalf("handleWin returned error: %v", err)
	}

	winScreen, ok := screen.(*DuelWinScreen)
	if !ok {
		t.Fatalf("expected DuelWinScreen, got %T", screen)
	}

	for _, card := range winScreen.cards {
		if card.Name() != forest.Name() && card.Name() != giantGrowth.Name() {
			t.Errorf("won card %q is not from the defeated enemy's deck (expected Forest or Giant Growth)", card.Name())
		}
	}
}
