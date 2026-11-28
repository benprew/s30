package duel

import (
	"image"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/world"
)

func TestDuelUsesPrintingCountsAndRemovesAnte(t *testing.T) {
	t.Cleanup(domain.ClearCardImageCache)
	card := domain.FindCardByName("Lightning Bolt")
	domain.CacheCardImage(card.DefaultArtwork().ID, image.NewRGBA(image.Rect(0, 0, domain.CardFullWidth, 342)))
	cc := domain.NewCardCollection()
	first := card.Printing("4ed-208-lightning-bolt")
	second := card.Printing("m10-146-lightning-bolt")
	cc.AddPrinting(first, 1)
	cc.AddPrinting(second, 1)
	for slot := range domain.MaxPlayerDecks {
		for _, printing := range []*domain.OwnedCard{first, second} {
			if err := cc.MovePrintingToDeck(printing, slot, 1); err != nil {
				t.Fatal(err)
			}
		}
	}
	s := &DuelScreen{
		player:        &domain.Player{Character: domain.Character{CardCollection: cc}, ActiveDeck: 1},
		anteDeckIndex: 1,
		lvl:           &world.Level{},
		cardImageMap:  map[string]*domain.Card{card.Name(): card},
	}
	player := interactive.NewHumanPlayer("You")
	ante, ownedAnte := s.addOwnedDeckToLibrary(player, s.player.GetActiveDeck(), cc, s.player.ActiveDeck, card)
	if len(player.Library()) != 2 || len(ante) != 1 || ownedAnte == nil {
		t.Fatal("deck did not retain its copies and ante")
	}
	if s.cardInstances[ante[0].ID().String()] != ownedAnte || ownedAnte != anteOwnedCard(cc, 1, card) {
		t.Fatal("the engine and preview used different ante artwork")
	}
	for _, engineCard := range player.Library() {
		owned := s.cardInstances[engineCard.ID().String()]
		if owned == nil {
			t.Fatal("engine card lost its owned copy")
		}
		img := s.cardArtImg(card, engineCard.ID().String(), domain.CardViewFull)
		want, _ := owned.CardImage(domain.CardViewFull)
		if img != want {
			t.Fatal("duel used different artwork")
		}
		s.loadCardPreviewInstance(card.Name(), engineCard.ID().String())
		if s.previewImage() != want {
			t.Fatal("preview lost the copy's artwork")
		}
		s.loadActionPreview(interactive.ActionOption{CardName: card.Name(), PermanentID: engineCard.ID()})
		if s.previewImage() != want {
			t.Fatal("ability preview lost the copy's artwork")
		}
	}
	s.anteCard, s.anteOwned = card, ownedAnte
	if _, _, err := s.handleLoss(); err != nil {
		t.Fatal(err)
	}
	for slot := range domain.MaxPlayerDecks {
		copies := cc.CardsInDeck(slot)
		if len(copies) != 1 || copies[0] != second {
			t.Fatalf("slot %d kept the lost printing or removed the other printing", slot)
		}
	}
}
