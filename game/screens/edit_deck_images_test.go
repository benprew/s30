package screens

import (
	"image"
	"strconv"
	"testing"

	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/ui/dragdrop"
	"github.com/benprew/s30/game/ui/elements"
)

func TestRefreshDeckDragImagesPreservesDragButton(t *testing.T) {
	card := domain.CARDS[0]
	owned := card.Printing("")
	domain.CacheCardImage(card.DefaultArtwork().ID, image.NewRGBA(image.Rect(0, 0, domain.CardFullWidth, 342)))
	t.Cleanup(domain.ClearCardImageCache)
	old, err := card.CardImage(domain.CardViewArtMini)
	if err != nil {
		t.Fatal(err)
	}
	collection := domain.NewCardCollection()
	collection.AddPrinting(owned, 1)
	if addErr := collection.MovePrintingToDeck(owned, 0, 1); addErr != nil {
		t.Fatal(addErr)
	}
	player := &domain.Player{Character: domain.Character{CardCollection: collection}}
	screen, err := NewEditDeckScreen(player, nil, 1024, 768)
	if err != nil {
		t.Fatal(err)
	}
	button := screen.deckCardDisplays[0].drag
	button.State = elements.StatePressed
	domain.CacheCardImage(card.DefaultArtwork().ID, image.NewRGBA(image.Rect(0, 0, domain.CardFullWidth, 342)))
	screen.refreshDeckDragImages()
	updated, err := owned.CardImage(domain.CardViewArtMini)
	if err != nil {
		t.Fatal(err)
	}
	if updated == old {
		t.Fatal("deck display retained old artwork")
	}
	if screen.deckCardDisplays[0].drag != button || button.State != elements.StatePressed {
		t.Fatal("art refresh changed the active drag button")
	}
	if button.Normal != updated || button.Hover != updated || button.Pressed != updated {
		t.Fatal("drag button did not receive the new artwork")
	}
}

func TestDeckCardLayoutFitsSharedCardSize(t *testing.T) {
	card := domain.CARDS[0]
	domain.CacheCardImage(card.DefaultArtwork().ID, image.NewRGBA(image.Rect(0, 0, domain.CardFullWidth, 342)))
	t.Cleanup(domain.ClearCardImageCache)
	img, err := card.CardImage(domain.CardViewArtMini)
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Size() != image.Pt(110, 96) {
		t.Fatalf("deck card size = %v", img.Bounds().Size())
	}
	area := image.Rect(0, 0, 360, 500)
	cards := layoutDeckCardsForTest(area, 4)
	for i, card := range cards {
		bounds := card.drag.Bounds
		if !bounds.In(area) {
			t.Fatalf("card %d extends outside deck area", i)
		}
		if bounds.Size() != img.Bounds().Size() {
			t.Fatalf("card %d hit area size = %v, image size %v", i, bounds.Size(), img.Bounds().Size())
		}
		for _, next := range cards[i+1:] {
			if bounds.Overlaps(next.drag.Bounds) {
				t.Fatal("deck cards overlap")
			}
		}
	}
}

func TestDeckCardLayoutHasSixCardsPerRow(t *testing.T) {
	area := image.Rect(300, 0, 1024, 540)
	cards := layoutDeckCardsForTest(area, 7)
	first := cards[0].drag.Bounds
	for i, card := range cards {
		bounds := card.drag.Bounds
		if !bounds.In(area) {
			t.Fatalf("card %d is outside the deck area: %v", i, bounds)
		}
		if i < 6 && bounds.Min.Y != first.Min.Y {
			t.Fatalf("card %d should be in the first row", i)
		}
		if i > 0 && i < 6 && bounds.Min.X < cards[i-1].drag.Bounds.Max.X {
			t.Fatalf("card %d overlaps the previous card", i)
		}
	}
	seventh := cards[6].drag.Bounds
	if seventh.Min.X != first.Min.X || seventh.Min.Y < first.Max.Y {
		t.Fatalf("seventh card should start the second row: %v", seventh)
	}
}

func TestDeckCardSpacingFitsUpTo120DistinctCards(t *testing.T) {
	area := image.Rect(300, 0, 1024, 540)
	for _, test := range []struct {
		cards   int
		rowStep int
	}{
		{24, 106},
		{30, 97},
		{31, 77},
		{42, 64},
		{84, 29},
		{120, 20},
	} {
		t.Run(strconv.Itoa(test.cards), func(t *testing.T) {
			cards := layoutDeckCardsForTest(area, test.cards)
			if len(cards) != test.cards {
				t.Fatalf("first page contains %d cards, want %d", len(cards), test.cards)
			}
			for i, card := range cards {
				bounds := card.drag.Bounds
				wantY := 55 + i/6*test.rowStep
				if bounds.Min.Y != wantY {
					t.Errorf("card %d starts at y=%d, want %d", i, bounds.Min.Y, wantY)
				}
				if !bounds.In(area) {
					t.Errorf("card %d at %v overflows deck area %v", i, bounds, area)
				}
				wantHeight := domain.CardArtMiniHeight
				if i+6 < len(cards) {
					wantHeight = min(wantHeight, test.rowStep)
				}
				if bounds.Dy() != wantHeight {
					t.Errorf("card %d has %d exposed pixels, want %d", i, bounds.Dy(), wantHeight)
				}
			}
		})
	}
}

func layoutDeckCardsForTest(area image.Rectangle, count int) []deckCardDisplay {
	screen := &EditDeckScreen{
		deckDropArea:     dragdrop.NewDropArea(area, nil, nil),
		deckCardDisplays: make([]deckCardDisplay, count),
	}
	for i := range screen.deckCardDisplays {
		screen.deckCardDisplays[i].drag = dragdrop.NewDraggableButton(&elements.Button{}, nil)
	}
	screen.calculateDeckCardPositions()
	return screen.visibleDeckCards()
}
