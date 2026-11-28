package screens

import (
	"image"
	"testing"

	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/ui"
	"github.com/benprew/s30/game/ui/dragdrop"
	"github.com/benprew/s30/game/ui/elements"
	"github.com/benprew/s30/game/ui/layout"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestCollectionDragIgnoresHiddenCardsWithOverlappingBounds(t *testing.T) {
	bounds := image.Rect(12, 598, 112, 698)
	previous := &elements.Button{ID: "Forest", Bounds: bounds}
	visible := &elements.Button{ID: "Mountain", Bounds: bounds}
	next := &elements.Button{ID: "Plains", Bounds: bounds}
	list, err := elements.NewScrollableList(
		[]*elements.Button{visible, next}, nil, 150, 180,
		elements.OrientationHorizontal, &layout.Position{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if list.GetVisibleCount() != 1 {
		t.Fatalf("visible count = %d, want 1", list.GetVisibleCount())
	}
	screen := &EditDeckScreen{
		CollectionList: list,
		draggableItems: []*dragdrop.DraggableButton{
			dragdrop.NewDraggableButton(previous, domain.FindCardByName(previous.ID)),
			dragdrop.NewDraggableButton(visible, domain.FindCardByName(visible.ID)),
			dragdrop.NewDraggableButton(next, domain.FindCardByName(next.ID)),
		},
	}
	var dropped dragdrop.DragData
	manager := dragdrop.NewDragManager()
	manager.RegisterDroppable(dragdrop.NewDropArea(image.Rect(300, 0, 1024, 540), []string{"*"}, func(data dragdrop.DragData) bool {
		dropped = data
		return true
	}))
	drag := ui.Drag{Start: image.Pt(20, 610), Position: image.Pt(400, 100)}
	manager.Start(drag, screen.visibleDraggables())
	manager.End(drag)
	if dropped == nil {
		t.Fatal("visible card was not dropped")
	}
	if dropped.GetData() != domain.FindCardByName("Mountain") {
		t.Fatalf("dropped card = %s, want Mountain", dropped.GetID())
	}
	if got := len(screen.visibleDraggables()); got != 1 {
		t.Fatalf("draggable count = %d, want 1", got)
	}
	deck := dragdrop.NewDraggableButton(&elements.Button{ID: "deck:Forest"}, domain.FindCardByName("Forest"))
	screen.deckDraggableItems = append(screen.deckDraggableItems, deck)
	items := screen.visibleDraggables()
	if len(items) != 2 || items[1] != deck {
		t.Fatal("deck card is not available for dragging")
	}
}

func TestDeckDragUsesExposedCard(t *testing.T) {
	screen := &EditDeckScreen{
		deckDropArea:   dragdrop.NewDropArea(image.Rect(300, 0, 1024, 540), nil, nil),
		CollectionList: &elements.ScrollableList{},
	}
	cardImage := ebiten.NewImage(domain.CardArtMiniWidth, domain.CardArtMiniHeight)
	defer cardImage.Deallocate()
	for _, card := range domain.CARDS[:31] {
		screen.deckCardDisplays = append(screen.deckCardDisplays, DeckCardDisplay{
			Owned: domain.NewOwnedCard(card),
			Image: cardImage,
		})
	}
	screen.calculateDeckCardPositions()
	screen.createDeckDraggableItems()

	manager := dragdrop.NewDragManager()
	var dropped dragdrop.DragData
	manager.RegisterDroppable(dragdrop.NewDropArea(image.Rect(0, 588, 1024, 768), []string{"*"}, func(data dragdrop.DragData) bool {
		dropped = data
		return true
	}))

	for _, tc := range []struct {
		point image.Point
		index int
	}{
		{image.Pt(315, 131), 0},
		{image.Pt(315, 132), 6},
		{image.Pt(433, 450), 25},
	} {
		dropped = nil
		if got := screen.deckCardAt(tc.point); got != tc.index {
			t.Errorf("hover at %v picked index %d, want %d", tc.point, got, tc.index)
		}
		drag := ui.Drag{Start: tc.point, Position: image.Pt(500, 630)}
		manager.Start(drag, screen.visibleDraggables())
		manager.End(drag)
		want := screen.deckCardDisplays[tc.index].Owned.Card
		if dropped == nil || dropped.GetData() != want {
			t.Errorf("drag from %v picked %v, want %s", tc.point, dropped, want.Name())
		}
	}
}

func TestDeckPageClampsAfterRemovingLastCardOnPage(t *testing.T) {
	screen := &EditDeckScreen{
		deckDropArea:   dragdrop.NewDropArea(image.Rect(300, 0, 1024, 540), nil, nil),
		CollectionList: &elements.ScrollableList{},
	}
	cardImage := ebiten.NewImage(domain.CardArtMiniWidth, domain.CardArtMiniHeight)
	defer cardImage.Deallocate()
	for _, card := range domain.CARDS[:121] {
		screen.deckCardDisplays = append(screen.deckCardDisplays, DeckCardDisplay{
			Owned: domain.NewOwnedCard(card),
			Image: cardImage,
		})
	}
	screen.calculateDeckCardPositions()
	screen.createDeckDraggableItems()
	if got := len(screen.visibleDraggables()); got != 120 {
		t.Fatalf("first page has %d drag targets, want 120", got)
	}
	point := image.Pt(320, 60)
	if !screen.setDeckPage(1) {
		t.Fatal("second page did not open")
	}
	items := screen.visibleDraggables()
	wantID := deckCardDragIDPrefix + screen.deckCardDisplays[120].Owned.Name()
	if len(items) != 1 || items[0].StartDrag(point.X, point.Y).GetID() != wantID {
		t.Fatalf("second page drag targets do not match %s", wantID)
	}
	if got := screen.deckCardAt(point); got != 120 {
		t.Errorf("hover on second page picked index %d, want 120", got)
	}
	screen.deckCardDisplays = screen.deckCardDisplays[:120]
	screen.calculateDeckCardPositions()
	screen.createDeckDraggableItems()
	if got := screen.deckCardAt(point); got != 0 {
		t.Errorf("hover after removing last card on page picked index %d, want 0", got)
	}
}
