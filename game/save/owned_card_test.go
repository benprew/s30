package save

import (
	"testing"

	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/world"
)

func TestSavePreservesOwnedCollectionAndShopArtwork(t *testing.T) {
	card := domain.FindCardByName("Mountain")
	owned, shop := domain.NewOwnedCard(card), domain.NewOwnedCard(card)
	owned.ArtworkID, shop.ArtworkID = "leb-300-mountain", "unh-139-mountain"
	owned.DeckIndex = 0
	cc := domain.NewCardCollection()
	cc.AddOwnedCard(owned)
	level := &world.Level{
		Player: &domain.Player{Character: domain.Character{CardCollection: cc}},
		Tiles:  [][]*world.Tile{{{City: &domain.City{CardsForSale: []*domain.OwnedCard{shop}}}}},
	}
	data, err := serializeSave(level)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := deserializeSave(data)
	if err != nil {
		t.Fatal(err)
	}
	copies := loaded.World.Player.CardCollection.CardsInDeck(0)
	if len(copies) != 1 || copies[0].ID != owned.ID || copies[0].ArtworkID != owned.ArtworkID {
		t.Fatal("collection copy changed after save")
	}
	stock := loaded.World.Tiles[0][0].City.CardsForSale
	if len(stock) != 1 || stock[0].ID != shop.ID || stock[0].ArtworkID != shop.ArtworkID {
		t.Fatal("shop copy changed after save")
	}
}
