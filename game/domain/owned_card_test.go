package domain

import (
	"encoding/json"
	"image/color"
	"testing"
)

func TestOwnedCardsKeepIdentityAndArtwork(t *testing.T) {
	card := FindCardByName("Lightning Bolt")
	first := NewOwnedCard(card)
	second := NewOwnedCard(card)
	first.ArtworkID = "4ed-208-lightning-bolt"
	second.ArtworkID = "m10-146-lightning-bolt"
	if first.ID == second.ID {
		t.Fatal("copies must have distinct IDs")
	}
	cc := NewCardCollection()
	cc.AddOwnedCard(first)
	cc.AddOwnedCard(second)
	if err := cc.MoveOwnedCard(first.ID, 0); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(cc)
	if err != nil {
		t.Fatal(err)
	}
	var restored CardCollection
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	deck := restored.CardsInDeck(0)
	if len(deck) != 1 || deck[0].ID != first.ID || deck[0].ArtworkID != first.ArtworkID {
		t.Fatalf("deck lost the owned card: %+v", deck)
	}
	if err := restored.RemoveOwnedCard(second.ID); err != nil {
		t.Fatal(err)
	}
	if restored.GetTotalCount(card) != 1 || restored.GetDeckCount(card, 0) != 1 {
		t.Fatal("selling a spare changed the card in the deck")
	}
	if err := restored.MoveOwnedCard(first.ID, -1); err != nil {
		t.Fatal(err)
	}
	spares := restored.AvailableCards(card)
	if len(spares) != 1 || spares[0].ID != first.ID || spares[0].ArtworkID != first.ArtworkID {
		t.Fatal("moving a card changed its identity or artwork")
	}
}

func TestOwnedCardUsesSavedArtworkAndFallsBackWhenRemoved(t *testing.T) {
	card := FindCardByName("Lightning Bolt")
	previous := card.Artworks
	t.Cleanup(func() { card.Artworks = previous; ClearCardImageCache() })
	card.Artworks = append([]Artwork{card.DefaultArtwork()}, []Artwork{{ID: "m10-146-lightning-bolt", CardSet: CardSet{SetID: "m10", CollectorNo: "146"}, BorderCropURL: "https://example.test/m10.jpg", CardName: card.Name()}}...)
	CacheCardImage(card.DefaultArtwork().ID, solidCardImage(color.RGBA{R: 255, A: 255}))
	CacheCardImage(card.Artworks[1].ID, solidCardImage(color.RGBA{B: 255, A: 255}))
	owned := NewOwnedCard(card)
	owned.ArtworkID = card.Artworks[1].ID
	first, err := owned.CardImage(CardViewFull)
	if err != nil || first == nil {
		t.Fatalf("alternate artwork failed: %v", err)
	}
	data, err := json.Marshal(owned)
	if err != nil {
		t.Fatal(err)
	}
	var restored OwnedCard
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	second, _ := restored.CardImage(CardViewFull)
	if second != first || restored.ID != owned.ID {
		t.Fatal("save changed the copy or its art")
	}
	card.Artworks = card.Artworks[:1]
	fallback, _ := restored.CardImage(CardViewFull)
	want, _ := card.CardImage(CardViewFull)
	if fallback != want || restored.ArtworkID != owned.ArtworkID {
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
	if len(deck) != 1 || deck[0].ArtworkID != forest.DefaultArtwork().ID {
		t.Fatalf("expected default artwork ID %q, got %+v", forest.DefaultArtwork().ID, deck)
	}
}

func TestCollectionCloneDoesNotShareCopies(t *testing.T) {
	cc := NewCardCollection()
	card := FindCardByName("Forest")
	cc.AddCardToDeck(card, 0, 2)
	clone := cc.Clone()
	owned := clone.CardsInDeck(0)[0]
	owned.ArtworkID = "changed"
	if err := clone.MoveOwnedCard(owned.ID, -1); err != nil {
		t.Fatal(err)
	}
	if cc.GetDeckCount(card, 0) != 2 {
		t.Fatal("clone changed the original deck")
	}
	for _, original := range cc.CardsInDeck(0) {
		if original.ArtworkID == "changed" {
			t.Fatal("clone shares an owned copy")
		}
	}
}

func TestLegacyCollectionCreatesCopiesWithDefaultArtwork(t *testing.T) {
	var cc CardCollection
	data := []byte(`[{"card_name":"Forest","count":3,"deck_counts":[1,1]}]`)
	if err := json.Unmarshal(data, &cc); err != nil {
		t.Fatal(err)
	}
	forest := FindCardByName("Forest")
	if len(cc.CardsInDeck(0)) != 1 || len(cc.CardsInDeck(1)) != 1 || len(cc.AvailableCards(forest)) != 1 {
		t.Fatal("migration changed deck membership")
	}
	for _, item := range cc {
		for _, owned := range item.Instances {
			if owned.ID == "" || owned.ArtworkID != forest.DefaultArtwork().ID {
				t.Fatalf("invalid migrated copy: %+v", owned)
			}
		}
	}
}
