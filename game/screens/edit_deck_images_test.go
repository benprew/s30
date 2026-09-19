package screens

import (
	"image"
	"testing"

	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/ui/dragdrop"
	"github.com/benprew/s30/game/ui/elements"
)

func TestRefreshDeckImagesPreservesDragButton(t *testing.T) {
	card := domain.CARDS[0]
	domain.CacheCardImage(card.CardID(), image.NewRGBA(image.Rect(0, 0, domain.CardFullWidth, 342)))
	t.Cleanup(domain.ClearCardImageCache)
	old, err := card.CardImage(domain.CardViewArtMini)
	if err != nil {
		t.Fatal(err)
	}
	screen := &EditDeckScreen{deckCardDisplays: []DeckCardDisplay{{Card: card, Image: old}}}
	screen.createDeckDraggableItems()
	button := screen.deckDraggableItems[0]
	button.State = elements.StatePressed
	domain.CacheCardImage(card.CardID(), image.NewRGBA(image.Rect(0, 0, domain.CardFullWidth, 342)))
	screen.refreshDeckImages()
	updated := screen.deckCardDisplays[0].Image
	if updated == old {
		t.Fatal("deck display retained old artwork")
	}
	if screen.deckDraggableItems[0] != button || button.State != elements.StatePressed {
		t.Fatal("art refresh changed the active drag button")
	}
	if button.Normal != updated || button.Hover != updated || button.Pressed != updated {
		t.Fatal("drag button did not receive the new artwork")
	}
}

func TestDeckCardLayoutFitsSharedCardSize(t *testing.T) {
	card := domain.CARDS[0]
	domain.CacheCardImage(card.CardID(), image.NewRGBA(image.Rect(0, 0, domain.CardFullWidth, 342)))
	t.Cleanup(domain.ClearCardImageCache)
	img, err := card.CardImage(domain.CardViewArtMini)
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Size() != image.Pt(110, 96) {
		t.Fatalf("deck card size = %v", img.Bounds().Size())
	}
	screen := &EditDeckScreen{
		deckDropArea:     dragdrop.NewDropArea(image.Rect(0, 0, 360, 500), nil, nil),
		deckCardDisplays: make([]DeckCardDisplay, 4),
	}
	for i := range screen.deckCardDisplays {
		screen.deckCardDisplays[i] = DeckCardDisplay{Card: card, Image: img}
	}
	screen.calculateDeckCardPositions()
	for i, a := range screen.deckCardDisplays {
		bounds := image.Rect(a.X, a.Y, a.X+110, a.Y+96)
		if !bounds.In(screen.deckDropArea.GetDropBounds()) {
			t.Fatalf("card %d extends outside deck area", i)
		}
		for _, b := range screen.deckCardDisplays[i+1:] {
			if bounds.Overlaps(image.Rect(b.X, b.Y, b.X+110, b.Y+96)) {
				t.Fatal("deck cards overlap")
			}
		}
	}
}

func TestDeckCardLayoutHasSixCardsPerRow(t *testing.T) {
	area := image.Rect(300, 0, 1024, 540)
	screen := &EditDeckScreen{
		deckDropArea:     dragdrop.NewDropArea(area, nil, nil),
		deckCardDisplays: make([]DeckCardDisplay, 7),
	}
	screen.calculateDeckCardPositions()
	first := screen.deckCardDisplays[0]
	for i, card := range screen.deckCardDisplays {
		bounds := image.Rect(card.X, card.Y, card.X+domain.CardArtMiniWidth, card.Y+domain.CardArtMiniHeight)
		if !bounds.In(area) {
			t.Fatalf("card %d is outside the deck area: %v", i, bounds)
		}
		if i < 6 && card.Y != first.Y {
			t.Fatalf("card %d should be in the first row", i)
		}
		if i > 0 && i < 6 && card.X < screen.deckCardDisplays[i-1].X+domain.CardArtMiniWidth {
			t.Fatalf("card %d overlaps the previous card", i)
		}
	}
	seventh := screen.deckCardDisplays[6]
	if seventh.X != first.X || seventh.Y < first.Y+domain.CardArtMiniHeight {
		t.Fatalf("seventh card should start the second row: %+v", seventh)
	}
}
