package screens

import (
	"testing"

	"github.com/benprew/s30/game/domain"
)

func TestEditorGroupsArtworkByNameAndSellsDisplayedCopy(t *testing.T) {
	card := domain.FindCardByName("Mountain")
	cc := domain.NewCardCollection()
	first, second := domain.NewOwnedCard(card), domain.NewOwnedCard(card)
	first.ArtworkID, second.ArtworkID = "leb-300-mountain", "unh-139-mountain"
	cc.AddOwnedCard(first)
	cc.AddOwnedCard(second)
	player := &domain.Player{Character: domain.Character{CardCollection: cc}}
	s, err := NewEditDeckScreen(player, &domain.City{Tier: domain.TierTown}, 1024, 768)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.CollectionList.GetItems()) != 1 || s.collectionGroups["Mountain"].totalCount != 2 {
		t.Fatal("artwork variants must share one stack")
	}
	displayed := s.collectionGroups["Mountain"].owned
	if !s.sellCard(displayed.Card, false) {
		t.Fatal("could not sell displayed copy")
	}
	remaining := cc.AvailableCards(card)
	if len(remaining) != 1 || remaining[0].ID == displayed.ID {
		t.Fatal("sale removed a different copy")
	}
	s.updateState()
	s.handleDeckCardAdd(0)
	if len(s.deckCardDisplays) != 1 || s.deckCardDisplays[0].Owned.ID != remaining[0].ID {
		t.Fatal("adding the stack changed the remaining copy")
	}
	s.handleDeckCardRemove(0)
	if got := cc.AvailableCards(card); len(got) != 1 || got[0].ArtworkID != remaining[0].ArtworkID {
		t.Fatal("removing from the deck changed the artwork")
	}
}

func TestBuyCardKeepsShopCopy(t *testing.T) {
	card := domain.FindCardByName("Lightning Bolt")
	owned := domain.NewOwnedCard(card)
	owned.ArtworkID = "m10-146-lightning-bolt"
	player := &domain.Player{Gold: card.Price, Character: domain.Character{CardCollection: domain.NewCardCollection()}}
	s := &BuyCardsScreen{City: &domain.City{CardsForSale: []*domain.OwnedCard{owned}}, Player: player, PreviewIdx: 0}
	s.buyCard()
	copies := player.CardCollection.AvailableCards(card)
	if len(copies) != 1 || copies[0].ID != owned.ID || copies[0].ArtworkID != owned.ArtworkID {
		t.Fatal("purchase changed the displayed shop copy")
	}
}
