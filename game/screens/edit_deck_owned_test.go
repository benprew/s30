package screens

import (
	"testing"

	"github.com/benprew/s30/game/domain"
)

func TestEditorGroupsArtworkByNameAndSellsDisplayedPrinting(t *testing.T) {
	card := domain.FindCardByName("Mountain")
	cc := domain.NewCardCollection()
	first := card.Printing("leb-300-mountain")
	second := card.Printing("unh-139-mountain")
	cc.AddPrinting(first, 1)
	cc.AddPrinting(second, 1)
	player := &domain.Player{Character: domain.Character{CardCollection: cc}}
	s, err := NewEditDeckScreen(player, &domain.City{Tier: domain.TierTown}, 1024, 768)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.CollectionList.GetItems()) != 1 || s.collectionGroups["Mountain"].totalCount != 2 {
		t.Fatal("artwork variants must share one stack")
	}
	displayed := s.collectionGroups["Mountain"].owned
	if displayed != first {
		t.Fatal("the group must show the first printing in artwork order")
	}
	drag := firstCollectionDrag(s)
	s.collectionGroups["Mountain"].owned = second
	if !s.handleCardDropToSell(drag) {
		t.Fatal("could not sell displayed printing")
	}
	if cc[displayed] != nil || cc.GetTotalCount(card) != 1 || cc[second] == nil {
		t.Fatal("sale reduced the wrong printing")
	}
	if !s.addDeckCard(second) {
		t.Fatal("could not add remaining printing to deck")
	}
	if len(s.deckCardDisplays) != 1 || s.deckCardDisplays[0].owned != second {
		t.Fatal("the deck shows a different printing")
	}
	if !s.removeDeckCard(second) {
		t.Fatal("could not remove the printing from the deck")
	}
	if item := cc[second]; item == nil || item.Count != 1 || item.DeckCount(0) != 0 {
		t.Fatal("removing the card from the deck changed its ownership")
	}
}

func TestBuyCardAddsToShopPrintingCount(t *testing.T) {
	card := domain.FindCardByName("Lightning Bolt")
	owned := card.Printing("m10-146-lightning-bolt")
	collection := domain.NewCardCollection()
	collection.AddPrinting(owned, 1)
	player := &domain.Player{Gold: card.Price, Character: domain.Character{CardCollection: collection}}
	s := &BuyCardsScreen{City: &domain.City{CardsForSale: domain.ShopStock{owned}}, Player: player, PreviewIdx: 0}
	s.buyCard()
	if collection[owned].Count != 2 || len(s.City.CardsForSale) != 0 {
		t.Fatal("purchase did not add one to the displayed printing's quantity")
	}
}
