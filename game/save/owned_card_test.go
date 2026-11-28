package save

import (
	"testing"

	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/world"
)

func TestSavePreservesOwnedCollectionAndShopArtwork(t *testing.T) {
	card := domain.FindCardByName("Mountain")
	owned := card.Printing("leb-300-mountain")
	shop := card.Printing("unh-139-mountain")
	cc := domain.NewCardCollection()
	cc.AddPrinting(owned, 1)
	for slot := range domain.MaxPlayerDecks {
		if err := cc.MovePrintingToDeck(owned, slot, 1); err != nil {
			t.Fatal(err)
		}
	}
	level := &world.Level{
		Player: &domain.Player{Character: domain.Character{CardCollection: cc}},
		Tiles:  [][]*world.Tile{{{City: &domain.City{CardsForSale: domain.ShopStock{shop}}}}},
	}
	data, err := serializeSave(level)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := deserializeSave(data)
	if err != nil {
		t.Fatal(err)
	}
	for slot := range domain.MaxPlayerDecks {
		copies := loaded.World.Player.CardCollection.CardsInDeck(slot)
		if len(copies) != 1 || copies[0] != owned {
			t.Fatalf("collection copy changed in slot %d after save", slot)
		}
	}
	stock := loaded.World.Tiles[0][0].City.CardsForSale
	if len(stock) != 1 || stock[0] != shop {
		t.Fatal("shop copy changed after save")
	}
}
