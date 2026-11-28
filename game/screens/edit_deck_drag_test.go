package screens

import (
	"image"
	"testing"

	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/ui"
	"github.com/benprew/s30/game/ui/dragdrop"
	"github.com/benprew/s30/game/ui/elements"
	"github.com/benprew/s30/game/ui/layout"
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
		draggableItems: map[*elements.Button]*dragdrop.DraggableButton{
			previous: dragdrop.NewDraggableButton(previous, domain.FindCardByName(previous.ID)),
			visible:  dragdrop.NewDraggableButton(visible, domain.FindCardByName(visible.ID)),
			next:     dragdrop.NewDraggableButton(next, domain.FindCardByName(next.ID)),
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
}

func TestDeckDragUsesExposedCard(t *testing.T) {
	screen := newEditorWithDistinctDeck(t, 31)
	manager := dragdrop.NewDragManager()
	var dropped dragdrop.DragData
	manager.RegisterDroppable(dragdrop.NewDropArea(image.Rect(0, 588, 1024, 768), []string{"*"}, func(data dragdrop.DragData) bool {
		dropped = data
		return true
	}))

	for _, test := range []struct {
		point image.Point
		index int
	}{
		{image.Pt(315, 131), 0},
		{image.Pt(315, 132), 6},
		{image.Pt(433, 450), 25},
	} {
		want := screen.deckCardDisplays[test.index].owned
		if got, fromDeck := screen.cardAt(test.point); got != want || !fromDeck {
			t.Errorf("hover at %v picked %v, want %s", test.point, got, want.Name())
		}
		dropped = nil
		drag := ui.Drag{Start: test.point, Position: image.Pt(500, 630)}
		manager.Start(drag, screen.visibleDraggables())
		manager.End(drag)
		if dropped == nil || dropped.GetData().(deckCardDrag).printing != want {
			t.Errorf("drag from %v picked %v, want %s", test.point, dropped, want.Name())
		}
	}
}
