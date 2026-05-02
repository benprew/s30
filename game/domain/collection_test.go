package domain

import (
	"encoding/json"
	"testing"
)

func TestCardCollectionMarshalJSON(t *testing.T) {
	cc := NewCardCollection()

	card := FindCardByName("Lightning Bolt")
	if card == nil {
		t.Fatal("Could not find Lightning Bolt card")
	}

	cc.AddCardToDeck(card, 0, 3)
	cc.AddCard(card, 1) // 1 extra not in any deck

	data, err := json.Marshal(cc)
	if err != nil {
		t.Fatalf("Failed to marshal CardCollection: %v", err)
	}

	cc2 := NewCardCollection()
	err = json.Unmarshal(data, &cc2)
	if err != nil {
		t.Fatalf("Failed to unmarshal CardCollection: %v", err)
	}

	if cc2.GetTotalCount(card) != 4 {
		t.Errorf("Expected total count 4, got %d", cc2.GetTotalCount(card))
	}
	if cc2.GetDeckCount(card, 0) != 3 {
		t.Errorf("Expected deck 0 count 3, got %d", cc2.GetDeckCount(card, 0))
	}
}

func TestCardCollectionMarshalJSONEmpty(t *testing.T) {
	cc := NewCardCollection()

	data, err := json.Marshal(cc)
	if err != nil {
		t.Fatalf("Failed to marshal empty CardCollection: %v", err)
	}

	cc2 := NewCardCollection()
	err = json.Unmarshal(data, &cc2)
	if err != nil {
		t.Fatalf("Failed to unmarshal empty CardCollection: %v", err)
	}

	if len(cc2) != 0 {
		t.Errorf("Expected empty collection, got %d items", len(cc2))
	}
}

func TestCardCollectionMarshalJSONMultiplePrintings(t *testing.T) {
	forests := FindAllCardsByName("Forest")
	if len(forests) < 2 {
		t.Skipf("need at least 2 Forest printings, got %d", len(forests))
	}

	cc := NewCardCollection()
	cc.AddCardToDeck(forests[0], 0, 5)
	cc.AddCardToDeck(forests[1], 0, 8)

	data, err := json.Marshal(cc)
	if err != nil {
		t.Fatalf("Failed to marshal CardCollection: %v", err)
	}

	cc2 := NewCardCollection()
	if err := json.Unmarshal(data, &cc2); err != nil {
		t.Fatalf("Failed to unmarshal CardCollection: %v", err)
	}

	total := cc2.GetTotalCount(forests[0]) + cc2.GetTotalCount(forests[1])
	if total != 13 {
		t.Errorf("Expected 13 total Forests across printings, got %d", total)
	}

	deckTotal := cc2.GetDeckCount(forests[0], 0) + cc2.GetDeckCount(forests[1], 0)
	if deckTotal != 13 {
		t.Errorf("Expected 13 Forests in deck 0 across printings, got %d", deckTotal)
	}
}

func TestCardCollectionMarshalJSONMultipleCards(t *testing.T) {
	cc := NewCardCollection()

	bolt := FindCardByName("Lightning Bolt")
	giant := FindCardByName("Hill Giant")
	if bolt == nil || giant == nil {
		t.Fatal("Could not find test cards")
	}

	cc.AddCardToDeck(bolt, 0, 4)
	cc.AddCardToDeck(giant, 0, 2)
	cc.AddCardToDeck(giant, 1, 1)

	data, err := json.Marshal(cc)
	if err != nil {
		t.Fatalf("Failed to marshal CardCollection: %v", err)
	}

	cc2 := NewCardCollection()
	err = json.Unmarshal(data, &cc2)
	if err != nil {
		t.Fatalf("Failed to unmarshal CardCollection: %v", err)
	}

	if cc2.GetTotalCount(bolt) != 4 {
		t.Errorf("Expected bolt count 4, got %d", cc2.GetTotalCount(bolt))
	}
	if cc2.GetDeckCount(bolt, 0) != 4 {
		t.Errorf("Expected bolt deck 0 count 4, got %d", cc2.GetDeckCount(bolt, 0))
	}
	if cc2.GetTotalCount(giant) != 3 {
		t.Errorf("Expected giant count 3, got %d", cc2.GetTotalCount(giant))
	}
	if cc2.GetDeckCount(giant, 1) != 1 {
		t.Errorf("Expected giant deck 1 count 1, got %d", cc2.GetDeckCount(giant, 1))
	}
}
