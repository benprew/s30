package duel

import (
	"image"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/benprew/s30/game/domain"
)

func TestDuelUsesOwnedCopiesAndExactAnte(t *testing.T) {
	t.Cleanup(domain.ClearCardImageCache)
	card := domain.FindCardByName("Lightning Bolt")
	domain.CacheCardImage(card.DefaultArtwork().ID, image.NewRGBA(image.Rect(0, 0, domain.CardFullWidth, 342)))
	cc := domain.NewCardCollection()
	first, second := domain.NewOwnedCard(card), domain.NewOwnedCard(card)
	first.ArtworkID, second.ArtworkID = card.DefaultArtwork().ID, "m10-146-lightning-bolt"
	first.DeckIndex, second.DeckIndex = 0, 0
	cc.AddOwnedCard(first)
	cc.AddOwnedCard(second)
	s := &DuelScreen{cardImageMap: map[string]*domain.Card{card.Name(): card}}
	player := interactive.NewHumanPlayer("You")
	ante, ownedAnte := s.addOwnedDeckToLibrary(player, cc.GetDeck(0), cc, 0, card)
	if len(player.Library()) != 2 || len(ante) != 1 || ownedAnte == nil {
		t.Fatal("deck did not retain its copies and ante")
	}
	if s.cardInstances[ante[0].ID().String()].ID != ownedAnte.ID {
		t.Fatal("engine ante differs from the owned ante")
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
	if err := cc.RemoveOwnedCard(ownedAnte.ID); err != nil {
		t.Fatal(err)
	}
	if len(cc.CardsInDeck(0)) != 1 {
		t.Fatal("ante loss removed another copy")
	}
}
