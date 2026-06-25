package duel

import (
	"testing"

	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/world"
)

func TestDuelWin_RequiresConfirmation(t *testing.T) {
	mountain := domain.FindCardByName("Mountain")
	bolt := domain.FindCardByName("Lightning Bolt")
	forest := domain.FindCardByName("Forest")

	player := &domain.Player{
		Character: domain.Character{
			CardCollection: domain.NewCardCollection(),
		},
	}

	s := NewWinDuelScreen(player, []*domain.Card{mountain, bolt, forest}, nil)

	if s.selected != -1 {
		t.Fatalf("expected no card selected initially, got %d", s.selected)
	}

	// Selecting a card alone must not add it to the collection.
	s.selectCardAt(s.choices[1].rect.Min)
	if s.selected != 1 {
		t.Fatalf("expected selected index 1, got %d", s.selected)
	}
	if _, exists := player.CardCollection[bolt]; exists {
		t.Fatal("card added to collection before confirmation")
	}

	// Confirming with nothing selected is a no-op.
	noSel := NewWinDuelScreen(player, []*domain.Card{mountain}, nil)
	if noSel.confirmSelection() {
		t.Fatal("confirmSelection returned true with no card selected")
	}

	// Confirming adds the selected card.
	if !s.confirmSelection() {
		t.Fatal("confirmSelection returned false with a card selected")
	}
	if item := player.CardCollection[bolt]; item == nil || item.Count != 1 {
		t.Fatal("confirmed card was not added to collection")
	}
}

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

	lvl.Enemies = []domain.Enemy{
		{Character: &domain.Character{Name: "Green Mage", Level: 1, CardCollection: enemy0Collection}},
		{Character: &domain.Character{Name: "Blue Mage", Level: 1, CardCollection: enemy1Collection}},
	}

	enemy := lvl.GetEnemyAt(0)

	s := &DuelScreen{
		player:        player,
		enemy:         enemy,
		lvl:           lvl,
		idx:           0,
		anteCard:      mountain,
		enemyAnteCard: giantGrowth,
	}

	_, screen, err := s.handleWin()
	if err != nil {
		t.Fatalf("handleWin returned error: %v", err)
	}

	winScreen, ok := screen.(*DuelWinScreen)
	if !ok {
		t.Fatalf("expected DuelWinScreen, got %T", screen)
	}

	if len(winScreen.choices) != 3 {
		t.Fatalf("expected 3 reward choices, got %d", len(winScreen.choices))
	}
	if winScreen.choices[0].card.Name() != giantGrowth.Name() {
		t.Errorf("expected first choice to be enemy ante card %q, got %q", giantGrowth.Name(), winScreen.choices[0].card.Name())
	}

	// The reward is a choice, so nothing is added to the collection until the
	// player picks one.
	if _, exists := player.CardCollection[giantGrowth]; exists {
		t.Error("enemy ante card was added to collection before the player chose it")
	}
}
