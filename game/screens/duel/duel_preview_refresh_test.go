package duel

import (
	"image"
	"testing"

	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/benprew/s30/game/domain"
)

func TestCardPreviewRefreshesWithoutChangingSelection(t *testing.T) {
	card := domain.CARDS[0]
	t.Cleanup(domain.ClearCardImageCache)
	domain.CacheCardImage(card.CardID(), image.NewRGBA(image.Rect(0, 0, 245, 342)))
	s := &DuelScreen{cardImageMap: map[string]*domain.Card{card.Name(): card}}
	perm := &interactive.PermanentState{Name: card.Name(), Power: 7, Toughness: 8}
	s.loadCardPreview(card.Name(), perm)
	initial := s.previewImage()
	domain.CacheCardImage(card.CardID(), image.NewRGBA(image.Rect(0, 0, 245, 342)))
	expected, _ := card.CardImage(domain.CardViewFull)
	if got := s.previewImage(); got == initial || got != expected {
		t.Fatal("preview retained old image")
	}
	if s.preview.Card != card || s.preview.Permanent != perm {
		t.Fatal("preview lost card or permanent state")
	}
}

func TestMulliganPreviewRefreshesWhileHoveringSameCard(t *testing.T) {
	s, _ := newMulliganScreen(t)
	card := domain.FindCardByName("Forest")
	s.cardImageMap = map[string]*domain.Card{card.Name(): card}
	t.Cleanup(domain.ClearCardImageCache)
	domain.CacheCardImage(card.CardID(), image.NewRGBA(image.Rect(0, 0, 245, 342)))
	rects := s.mulliganCardRects(1024, 768)
	point := rects[0].Min.Add(image.Pt(5, 5))
	s.updateMulliganPreview(rects, point.X, point.Y)
	initial := s.mulliganPreviewImg
	domain.CacheCardImage(card.CardID(), image.NewRGBA(image.Rect(0, 0, 245, 342)))
	s.updateMulliganPreview(rects, point.X, point.Y)
	expected, _ := card.CardImage(domain.CardViewFull)
	if s.mulliganPreviewImg == initial || s.mulliganPreviewImg != expected {
		t.Fatal("mulligan preview retained old image")
	}
}

func TestCardPreviewUsesCurrentLandArt(t *testing.T) {
	forest := domain.FindCardByName("Forest")
	island := domain.FindCardByName("Island")
	t.Cleanup(domain.ClearCardImageCache)
	for _, card := range []*domain.Card{forest, island} {
		domain.CacheCardImage(card.CardID(), image.NewRGBA(image.Rect(0, 0, 245, 342)))
	}
	s := &DuelScreen{cardImageMap: map[string]*domain.Card{"Forest": forest, "Island": island}}
	perm := &interactive.PermanentState{Name: "Forest", IsLand: true, SubTypes: "Forest"}
	s.loadCardPreview("Forest", perm)
	initial, _ := forest.CardImage(domain.CardViewFull)
	if s.previewImage() != initial {
		t.Fatal("preview did not use printed land art")
	}
	perm.SubTypes = "Island"
	expected, _ := island.CardImage(domain.CardViewFull)
	if s.previewImage() != expected || s.preview.Card != forest {
		t.Fatal("preview did not preserve printed card with current land art")
	}
}

func TestCardPreviewClearsUnknownCard(t *testing.T) {
	card := domain.CARDS[0]
	s := &DuelScreen{cardImageMap: map[string]*domain.Card{card.Name(): card}}
	s.loadCardPreviewByName(card.Name())
	s.loadCardPreviewByName("unknown card")
	if s.preview != nil || s.previewImage() != nil {
		t.Fatal("preview retained unknown card")
	}
}
