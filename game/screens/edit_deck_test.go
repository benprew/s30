package screens

import (
	"testing"

	"github.com/benprew/s30/game/domain"
)

func TestCreateCollectionButtons_ExcludesDeckCards(t *testing.T) {
	mountain := domain.FindCardByName("Mountain")
	forest := domain.FindCardByName("Forest")
	plains := domain.FindCardByName("Plains")

	collection := domain.NewCardCollection()
	collection.AddCard(mountain, 13)
	collection.MoveCardToDeck(mountain, 0, 8)
	collection.AddCard(forest, 5)
	collection.MoveCardToDeck(forest, 0, 5)
	collection.AddCard(plains, 3)

	player := &domain.Player{
		Character: domain.Character{
			CardCollection: collection,
		},
		ActiveDeck: 0,
	}

	screen := &EditDeckScreen{
		Player:           player,
		collectionGroups: make(map[string]*cardGroup),
	}

	buttons, err := screen.createCollectionButtons()
	if err != nil {
		t.Fatalf("Failed to create collection buttons: %v", err)
	}

	var mountainGroup *cardGroup
	var forestGroup *cardGroup
	var plainsGroup *cardGroup

	for name, group := range screen.collectionGroups {
		switch name {
		case "Mountain":
			mountainGroup = group
		case "Forest":
			forestGroup = group
		case "Plains":
			plainsGroup = group
		}
	}

	if mountainGroup == nil {
		t.Error("Mountain should appear in collection list (13 total, 8 in deck, 5 available)")
	} else if mountainGroup.totalCount != 5 {
		t.Errorf("Mountain available count should be 5, got %d", mountainGroup.totalCount)
	}

	if forestGroup != nil {
		t.Error("Forest should NOT appear in collection list (5 total, 5 in deck, 0 available)")
	}

	if plainsGroup == nil {
		t.Error("Plains should appear in collection list (3 total, 0 in deck, 3 available)")
	} else if plainsGroup.totalCount != 3 {
		t.Errorf("Plains available count should be 3, got %d", plainsGroup.totalCount)
	}

	expectedButtonCount := 2
	if len(buttons) != expectedButtonCount {
		t.Errorf("Expected %d buttons (Mountain and Plains), got %d", expectedButtonCount, len(buttons))
	}
}

func TestCreateCollectionButtons_RemovesWhenAllInDeck(t *testing.T) {
	mountain := domain.FindCardByName("Mountain")

	collection := domain.NewCardCollection()
	collection.AddCard(mountain, 10)
	collection.MoveCardToDeck(mountain, 0, 10)

	player := &domain.Player{
		Character: domain.Character{
			CardCollection: collection,
		},
		ActiveDeck: 0,
	}

	screen := &EditDeckScreen{
		Player:           player,
		collectionGroups: make(map[string]*cardGroup),
	}
	buttons, err := screen.createCollectionButtons()
	if err != nil {
		t.Fatalf("Failed to create collection buttons: %v", err)
	}

	if len(buttons) != 0 {
		t.Errorf("Expected 0 buttons when all 10 cards are in deck, got %d buttons", len(buttons))
	}

	mountainGroup := screen.collectionGroups["Mountain"]
	if mountainGroup != nil {
		t.Error("Mountain should NOT appear in collection list when all copies are in deck")
	}
}

func TestMoveCardFromDeck(t *testing.T) {
	mountain := domain.FindCardByName("Mountain")

	collection := domain.NewCardCollection()
	collection.AddCard(mountain, 10)
	collection.MoveCardToDeck(mountain, 0, 5)

	totalCount := collection.GetTotalCount(mountain)
	deckCount := collection.GetDeckCount(mountain, 0)

	if totalCount != 10 {
		t.Errorf("Expected total count to be 10, got %d", totalCount)
	}
	if deckCount != 5 {
		t.Errorf("Expected deck count to be 5, got %d", deckCount)
	}

	err := collection.MoveCardFromDeck(mountain, 0, 2)
	if err != nil {
		t.Fatalf("Failed to move card from deck: %v", err)
	}

	totalCount = collection.GetTotalCount(mountain)
	deckCount = collection.GetDeckCount(mountain, 0)

	if totalCount != 10 {
		t.Errorf("Expected total count to remain 10, got %d", totalCount)
	}
	if deckCount != 3 {
		t.Errorf("Expected deck count to be 3 after removing 2, got %d", deckCount)
	}

	player := &domain.Player{
		Character: domain.Character{
			CardCollection: collection,
		},
		ActiveDeck: 0,
	}

	screen := &EditDeckScreen{
		Player:           player,
		collectionGroups: make(map[string]*cardGroup),
	}

	buttons, err := screen.createCollectionButtons()
	if err != nil {
		t.Fatalf("Failed to create collection buttons: %v", err)
	}

	if len(buttons) != 1 {
		t.Errorf("Expected 1 button, got %d", len(buttons))
	}

	mountainGroup := screen.collectionGroups["Mountain"]
	if mountainGroup == nil {
		t.Fatal("Mountain should appear in collection list")
	}
	if mountainGroup.totalCount != 7 {
		t.Errorf("Expected 7 available (10 total - 3 in deck), got %d", mountainGroup.totalCount)
	}
}
