package domain

import (
	"encoding/json"
	"image/color"
	"maps"
	"slices"
	"testing"
)

func TestOwnedCardJSONReferencesCatalog(t *testing.T) {
	card := FindCardByName("Lightning Bolt")
	printing := card.Printing("4ed-208-lightning-bolt")
	data, err := json.Marshal(printing)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"card_id": card.CardID(), "artwork_id": printing.ArtworkID()}
	var fields map[string]string
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatalf("expected a flat printing reference in %s: %v", data, err)
	}
	if !maps.Equal(fields, want) {
		t.Fatalf("saved fields = %v, want %v", fields, want)
	}
	var stock ShopStock
	if err := json.Unmarshal([]byte("["+string(data)+"]"), &stock); err != nil {
		t.Fatal(err)
	}
	if len(stock) != 1 || stock[0] != printing {
		t.Fatal("loading did not resolve the shared printing")
	}
}

func TestOwnedCardUsesSavedArtworkAndFallsBackWhenRemoved(t *testing.T) {
	card := FindCardByName("Lightning Bolt")
	previous := card.Artworks
	t.Cleanup(func() { card.Artworks = previous; ClearCardImageCache() })
	card.Artworks = append([]Artwork{card.DefaultArtwork()}, []Artwork{{ID: "m10-146-lightning-bolt", CardSet: CardSet{SetID: "m10", CollectorNo: "146"}, BorderCropURL: "https://example.test/m10.jpg", CardName: card.Name()}}...)
	CacheCardImage(card.DefaultArtwork().ID, solidCardImage(color.RGBA{R: 255, A: 255}))
	CacheCardImage(card.Artworks[1].ID, solidCardImage(color.RGBA{B: 255, A: 255}))
	printing := card.Printing(card.Artworks[1].ID)
	first, err := printing.CardImage(CardViewFull)
	if err != nil || first == nil {
		t.Fatalf("alternate artwork failed: %v", err)
	}
	data, err := json.Marshal(ShopStock{printing})
	if err != nil {
		t.Fatal(err)
	}
	var stock ShopStock
	if err := json.Unmarshal(data, &stock); err != nil {
		t.Fatal(err)
	}
	second, _ := stock[0].CardImage(CardViewFull)
	if second != first || stock[0] != printing {
		t.Fatal("save changed the printing or its artwork")
	}
	card.Artworks = card.Artworks[:1]
	fallback, _ := stock[0].CardImage(CardViewFull)
	want, _ := card.CardImage(CardViewFull)
	if fallback != want || stock[0].ArtworkID() != "m10-146-lightning-bolt" {
		t.Fatal("removed artwork must use default art and retain its ID")
	}
}

func TestEnemiesShareStaticRogueCollection(t *testing.T) {
	first, err := NewEnemy("Crusader")
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewEnemy("Crusader")
	if err != nil {
		t.Fatal(err)
	}
	if first.Character != Rogues["Crusader"] || second.Character != Rogues["Crusader"] {
		t.Fatal("enemy did not reuse static rogue character")
	}
}

func TestCollectionUnmarshalBackfillsMissingInstanceArtworkID(t *testing.T) {
	var cc CardCollection
	data := []byte(`[{"card_name":"Forest","count":1,"deck_counts":[1],"instances":[{"id":"instance-1","deck_index":0}]}]`)
	if err := json.Unmarshal(data, &cc); err != nil {
		t.Fatal(err)
	}
	forest := FindCardByName("Forest")
	deck := cc.CardsInDeck(0)
	if len(deck) != 1 || deck[0] != forest.Printing("") {
		t.Fatalf("expected default artwork ID %q, got %+v", forest.DefaultArtwork().ID, deck)
	}
}

func TestCollectionCloneSharesPrintingsAndCopiesCounts(t *testing.T) {
	cc := NewCardCollection()
	printing := FindCardByName("Forest").Printing("")
	cc.AddPrinting(printing, 2)
	for slot := range 2 {
		if err := cc.MovePrintingToDeck(printing, slot, 2); err != nil {
			t.Fatal(err)
		}
	}
	clone := cc.Clone()
	if clone[printing].Card != cc[printing].Card {
		t.Fatal("the clone did not reuse the printing")
	}
	if err := clone.RemovePrinting(printing, 1, 0); err != nil {
		t.Fatal(err)
	}
	if cc[printing].Count != 2 || !slices.Equal(cc[printing].DeckCounts, []int{2, 2}) {
		t.Fatal("editing the clone changed the original quantity or decks")
	}
}

func TestLegacyCollectionUsesDefaultPrinting(t *testing.T) {
	var cc CardCollection
	data := []byte(`[{"card_name":"Forest","count":3,"deck_counts":[1,1]}]`)
	if err := json.Unmarshal(data, &cc); err != nil {
		t.Fatal(err)
	}
	printing := FindCardByName("Forest").Printing("")
	item := cc[printing]
	if len(cc) != 1 || item == nil || item.Count != 3 || !slices.Equal(item.DeckCounts, []int{1, 1}) {
		t.Fatalf("migrated default printing = %+v", item)
	}
}
