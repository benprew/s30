package domain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"testing"
)

func TestAddCardWithDeckAssignment(t *testing.T) {
	card := FindCardByName("Plains")
	for _, slot := range []int{-1, 0, 2} {
		t.Run(fmt.Sprintf("slot_%d", slot), func(t *testing.T) {
			cc := NewCardCollection()
			cc.AddCard(card, 3, slot)
			cc.AddCard(card, 2, slot)
			cc.AddCard(card, 0, slot)
			cc.AddCard(card, -1, slot)
			if got := cc.GetTotalCount(card); got != 5 {
				t.Fatalf("owned count = %d, want 5", got)
			}
			for deck := range MaxPlayerDecks {
				want := 0
				if deck == slot {
					want = 5
				}
				if got := cc.GetDeckCount(card, deck); got != want {
					t.Errorf("deck %d count = %d, want %d", deck, got, want)
				}
			}
		})
	}
}

func TestRemoveCardAndPrintingWithDeckSelection(t *testing.T) {
	card := FindCardByName("Plains")
	printing := card.Printing("")
	cases := []struct {
		name      string
		count     int
		slot      int
		wantCount int
		wantDecks []int
		wantError bool
	}{
		{"collection", 1, -1, 2, []int{1, 2, 2}, false},
		{"deck", 1, 1, 2, []int{1, 1, 2}, false},
		{"zero", 0, 0, 3, []int{1, 2, 3}, false},
		{"too many owned", 4, -1, 3, []int{1, 2, 3}, true},
		{"too many in deck", 2, 0, 3, []int{1, 2, 3}, true},
		{"empty deck", 1, 3, 3, []int{1, 2, 3}, true},
		{"invalid slot", 1, -2, 3, []int{1, 2, 3}, true},
		{"negative count", -1, -1, 3, []int{1, 2, 3}, true},
		{"remove all", 3, 2, 0, nil, false},
	}
	removals := []struct {
		name string
		run  func(CardCollection, int, int) error
	}{
		{"card", func(cc CardCollection, count, slot int) error { return cc.RemoveCard(card, count, slot) }},
		{"printing", func(cc CardCollection, count, slot int) error { return cc.RemovePrinting(printing, count, slot) }},
	}
	for _, removal := range removals {
		for _, tc := range cases {
			t.Run(removal.name+"/"+tc.name, func(t *testing.T) {
				cc := NewCardCollection()
				item := cc.AddPrinting(printing, 3)
				item.DeckCounts = []int{1, 2, 3}
				err := removal.run(cc, tc.count, tc.slot)
				if (err != nil) != tc.wantError {
					t.Fatalf("RemoveCard() error = %v, want error %t", err, tc.wantError)
				}
				if cc.GetTotalCount(card) != tc.wantCount {
					t.Fatalf("owned count = %d, want %d", cc.GetTotalCount(card), tc.wantCount)
				}
				if tc.wantCount == 0 {
					if len(cc) != 0 {
						t.Fatal("empty printing remains in the collection")
					}
				} else if !slices.Equal(cc[printing].DeckCounts, tc.wantDecks) {
					t.Fatalf("deck counts = %v, want %v", cc[printing].DeckCounts, tc.wantDecks)
				}
			})
		}
	}
}

func TestRemoveCardUsesSparePrintingFirst(t *testing.T) {
	card := FindCardByName("Forest")
	first, second := card.Printing("a-forest"), card.Printing("b-forest")
	cc := NewCardCollection()
	cc.AddPrinting(first, 1).DeckCounts = []int{1, 1}
	cc.AddPrinting(second, 2).DeckCounts = []int{1, 1}
	if err := cc.RemoveCard(card, 1, -1); err != nil {
		t.Fatal(err)
	}
	if cc[first] == nil || cc[first].Count != 1 || cc[second] == nil || cc[second].Count != 1 {
		t.Fatal("removal did not use the spare printing")
	}
	if !slices.Equal(cc[first].DeckCounts, []int{1, 1}) || !slices.Equal(cc[second].DeckCounts, []int{1, 1}) {
		t.Fatal("removal reduced a deck that had enough owned cards")
	}
}

func TestRemovePrintingFromDeckCapsOtherDecks(t *testing.T) {
	printing := FindCardByName("Plains").Printing("")
	cc := NewCardCollection()
	cc.AddPrinting(printing, 2)
	for slot, count := range []int{1, 1, 2} {
		if err := cc.MovePrintingToDeck(printing, slot, count); err != nil {
			t.Fatal(err)
		}
	}
	if err := cc.RemovePrinting(printing, 1, 0); err != nil {
		t.Fatal(err)
	}
	item := cc[printing]
	if item.Count != 1 || !slices.Equal(item.DeckCounts, []int{0, 1, 1}) {
		t.Fatalf("after sale: %+v, want one owned and deck counts [0 1 1]", item)
	}
}

func TestLegacyCopyIDsLoadAsCountsPerPrinting(t *testing.T) {
	data := []byte(`[{"card_id":"Forest","instances":[
		{"id":"a","artwork_id":"leb-301-forest"},
		{"id":"b","artwork_id":"leb-301-forest"},
		{"id":"c","artwork_id":"unh-140-forest"}
	],"deck_copy_ids":[["a","a","b"],["a","c"],["a","b","c"]]}]`)
	var cc CardCollection
	if err := json.Unmarshal(data, &cc); err != nil {
		t.Fatal(err)
	}
	forest := FindCardByName("Forest")
	first, second := cc[forest.Printing("leb-301-forest")], cc[forest.Printing("unh-140-forest")]
	if first == nil || first.Count != 2 || !slices.Equal(first.DeckCounts, []int{2, 1, 2}) {
		t.Fatalf("first printing = %+v, want two owned and deck counts [2 1 2]", first)
	}
	if second == nil || second.Count != 1 || !slices.Equal(second.DeckCounts, []int{0, 1, 1}) {
		t.Fatalf("second printing = %+v, want one owned and deck counts [0 1 1]", second)
	}
	current, err := json.Marshal(cc)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(current, []byte(`"id"`)) || bytes.Contains(current, []byte(`"deck_copy_ids"`)) {
		t.Fatalf("new save contains individual copy IDs: %s", current)
	}
}

func TestSharedPrintingCountsSurviveReload(t *testing.T) {
	bolt := FindCardByName("Lightning Bolt")
	cc := NewCardCollection()
	cc.AddPrinting(bolt.Printing("4ed-208-lightning-bolt"), 2)
	cc.AddPrinting(bolt.Printing("m10-146-lightning-bolt"), 2)
	for slot := range MaxPlayerDecks {
		for printing := range cc {
			if err := cc.MovePrintingToDeck(printing, slot, 2); err != nil {
				t.Fatalf("add printing to slot %d: %v", slot, err)
			}
			if err := cc.MovePrintingToDeck(printing, slot, 1); err == nil {
				t.Fatalf("slot %d accepted an extra card", slot)
			}
		}
	}
	data, err := json.Marshal(cc)
	if err != nil {
		t.Fatal(err)
	}
	var loaded CardCollection
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 2 || loaded.GetTotalCount(bolt) != 4 {
		t.Fatal("reload changed the owned printings or quantities")
	}
	for printing, item := range loaded {
		if item.Count != 2 || !slices.Equal(item.DeckCounts, []int{2, 2, 2}) || cc[printing] == nil {
			t.Fatalf("unexpected printing after reload: %+v", item)
		}
	}
}

func TestLegacyCollectionPreservesCountInEachDeck(t *testing.T) {
	bolt := FindCardByName("Lightning Bolt")
	data := []byte(`[{"card_name":"Lightning Bolt","count":4,"deck_counts":[4,4,4]}]`)
	var cc CardCollection
	if err := json.Unmarshal(data, &cc); err != nil {
		t.Fatal(err)
	}
	counts := make([]int, MaxPlayerDecks)
	for slot := range counts {
		counts[slot] = cc.GetDeckCount(bolt, slot)
	}
	if !slices.Equal(counts, []int{4, 4, 4}) || cc.GetTotalCount(bolt) != 4 {
		t.Fatalf("migrated %d copies into decks with counts %v, want 4 owned copies used in all three decks", cc.GetTotalCount(bolt), counts)
	}
}

func TestCardCollectionMarshalJSON(t *testing.T) {
	cc := NewCardCollection()

	card := FindCardByName("Lightning Bolt")
	if card == nil {
		t.Fatal("Could not find Lightning Bolt card")
	}

	cc.AddCard(card, 3, 0)
	cc.AddCard(card, 1, -1) // 1 extra not in any deck

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

func TestCardCollectionMarshalJSONMultipleArtworks(t *testing.T) {
	card := FindCardByName("Forest")
	cc := NewCardCollection()
	first, second := card.Printing(""), card.Printing("unh-140-forest")
	cc.AddPrinting(first, 5)
	cc.AddPrinting(second, 8)
	for _, item := range cc {
		if err := cc.MovePrintingToDeck(item.Card, 0, item.Count); err != nil {
			t.Fatal(err)
		}
	}
	data, err := json.Marshal(cc)
	if err != nil {
		t.Fatal(err)
	}
	var restored CardCollection
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	if len(restored) != 2 || restored.GetDeckCount(card, 0) != 13 {
		t.Fatal("reload changed the artwork groups or the combined deck count")
	}
	if restored[first] == nil || restored[first].Count != 5 || restored[second] == nil || restored[second].Count != 8 {
		t.Fatalf("artwork counts changed: %+v, %+v", restored[first], restored[second])
	}
}

func TestCardCollectionMarshalJSONMultipleCards(t *testing.T) {
	cc := NewCardCollection()

	bolt := FindCardByName("Lightning Bolt")
	giant := FindCardByName("Hill Giant")
	if bolt == nil || giant == nil {
		t.Fatal("Could not find test cards")
	}

	cc.AddCard(bolt, 4, 0)
	cc.AddCard(giant, 2, 0)
	cc.AddCard(giant, 1, 1)

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

func TestCardCollectionUnmarshalJSONMissingSetIDFallback(t *testing.T) {
	bolt := FindCardByName("Lightning Bolt")
	if bolt == nil {
		t.Fatal("Could not find Lightning Bolt")
	}

	jsonData := []byte(`[
		{"card_id": "4ed-999-lightning-bolt", "card_name": "Lightning Bolt", "count": 2, "deck_counts": [2]},
		{"card_id": "lea-999-lightning-bolt", "card_name": "", "count": 1, "deck_counts": [1]}
	]`)

	var cc CardCollection
	if err := json.Unmarshal(jsonData, &cc); err != nil {
		t.Fatalf("Failed to unmarshal collection with old card IDs: %v", err)
	}

	if cc.GetTotalCount(bolt) != 3 {
		t.Errorf("Expected total count 3, got %d", cc.GetTotalCount(bolt))
	}
	if cc.GetDeckCount(bolt, 0) != 3 {
		t.Errorf("Expected deck 0 count 3, got %d", cc.GetDeckCount(bolt, 0))
	}
}

func TestCardCollectionUnmarshalJSONSkipsRemovedCards(t *testing.T) {
	bolt := FindCardByName("Lightning Bolt")
	if bolt == nil {
		t.Fatal("Could not find Lightning Bolt")
	}

	jsonData := []byte(`[
		{"card_id": "2ed-136-word-of-command", "card_name": "Word of Command", "count": 2, "deck_counts": [2]},
		{"card_id": "2ed-236-chaos-orb", "card_name": "Chaos Orb", "count": 1, "deck_counts": [1]},
		{"card_id": "2ed-161-lightning-bolt", "card_name": "Lightning Bolt", "count": 4, "deck_counts": [4]}
	]`)

	var cc CardCollection
	if err := json.Unmarshal(jsonData, &cc); err != nil {
		t.Fatalf("Failed to unmarshal collection with removed cards: %v", err)
	}

	if len(cc) != 1 {
		t.Errorf("Expected exactly 1 card in collection, got %d", len(cc))
	}
	if cc.GetTotalCount(bolt) != 4 {
		t.Errorf("Expected bolt count 4, got %d", cc.GetTotalCount(bolt))
	}
}
