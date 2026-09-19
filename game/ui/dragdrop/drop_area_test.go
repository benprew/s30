package dragdrop

import (
	"image"
	"testing"
)

func TestDropAreaCanAccept(t *testing.T) {
	area := NewDropArea(image.Rect(0, 0, 100, 100), []string{"*"}, nil)
	data1 := &CardDragData{ID: "card1"}
	data2 := &CardDragData{ID: "deck:card1"}

	if !area.CanAcceptDrop(data1) || !area.CanAcceptDrop(data2) {
		t.Fatal("expected default '*' drop area to accept both")
	}

	area.SetCanAccept(func(data DragData) bool {
		return data.GetID() == "card1"
	})

	if !area.CanAcceptDrop(data1) {
		t.Fatal("expected area to accept data1 matching predicate")
	}
	if area.CanAcceptDrop(data2) {
		t.Fatal("expected area to reject data2 failing predicate")
	}
}
