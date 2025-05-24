package core_engine

import (
	"testing"
)

func TestLoadCardDatabase(t *testing.T) {
	// Ensure the database is initially empty (or reset it if tests might run multiple times)
	// For simplicity in this test, we assume it's run in isolation or the map is global and persistent.
	// A more robust test might reset the map before loading.
	CardDatabase := LoadCardDatabase()

	// Check if the database was populated
	if len(CardDatabase) == 0 {
		t.Errorf("CardDatabase is empty after loading")
	}

	// Check for a known card (assuming "Forest" is in testset/cards.json)
	forestCard, ok := CardDatabase["Forest"]
	if !ok {
		t.Errorf("Card 'Forest' not found in database")
	}

	// Check some properties of the known card
	if forestCard.Name() != "Forest" {
		t.Errorf("Expected Forest name 'Forest', got '%s'", forestCard.Name())
	}

	if forestCard.CardType != CardTypeLand {
		t.Errorf("Expected Forest card type '%s', got '%s'", CardTypeLand, forestCard.CardType)
	}
}
