package core_engine

import (
	"fmt"
	"testing"

	"github.com/benprew/s30/game/domain"
)

func TestLoadCardDatabase(t *testing.T) {
	// Ensure the database is initially empty (or reset it if tests might run multiple times)
	// For simplicity in this test, we assume it's run in isolation or the map is global and persistent.
	// A more robust test might reset the map before loading.

	// Check if the database was populated
	if len(domain.CARDS) == 0 {
		t.Errorf("CardDatabase is empty after loading")
	}

	fmt.Println("Num Cards", len(domain.CARDS))

	// Check for a known card (assuming "Forest" is in testset/cards.json)
	forestCard := domain.FindCardByName("Forest")
	if forestCard == nil {
		t.Errorf("Card 'Forest' not found in database")
	}

	// Check some properties of the known card
	if forestCard.Name() != "Forest" {
		t.Errorf("Expected Forest name 'Forest', got '%s'", forestCard.Name())
	}

	if forestCard.CardType != domain.CardTypeLand {
		t.Errorf("Expected Forest card type '%s', got '%s'", domain.CardTypeLand, forestCard.CardType)
	}
}
