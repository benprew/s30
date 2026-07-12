package ui

import (
	"image"
	"testing"
)

func TestPointerClick(t *testing.T) {
	pointer := newPointer()
	pointer.advance(pointerSample{position: image.Pt(10, 10), down: true})
	pointer.advance(pointerSample{position: image.Pt(11, 10)})

	position, clicked := pointer.Click()
	if !clicked || position != image.Pt(11, 10) {
		t.Fatalf("Click() = %v, %t; want (11,10), true", position, clicked)
	}
	if _, dragging := pointer.Dragging(); dragging {
		t.Fatal("a click must not also be a drag")
	}
}

func TestPointerDragLifecycle(t *testing.T) {
	pointer := newPointer()
	pointer.advance(pointerSample{position: image.Pt(10, 10), down: true})
	pointer.advance(pointerSample{position: image.Pt(15, 10), down: true})
	if _, started := pointer.DragStart(); started {
		t.Fatal("drag started before reaching the movement threshold")
	}

	pointer.advance(pointerSample{position: image.Pt(20, 10), down: true})
	drag, started := pointer.DragStart()
	if !started || drag.Start != image.Pt(10, 10) || drag.Position != image.Pt(20, 10) || drag.Delta != image.Pt(5, 0) {
		t.Fatalf("DragStart() = %#v, %t", drag, started)
	}
	if active, dragging := pointer.Dragging(); !dragging || active != drag {
		t.Fatalf("Dragging() = %#v, %t; want active drag", active, dragging)
	}

	pointer.advance(pointerSample{position: image.Pt(24, 12)})
	drag, ended := pointer.DragEnd()
	if !ended || drag.Start != image.Pt(10, 10) || drag.Position != image.Pt(24, 12) || drag.Delta != image.Pt(4, 2) {
		t.Fatalf("DragEnd() = %#v, %t", drag, ended)
	}
	if _, clicked := pointer.Click(); clicked {
		t.Fatal("a completed drag must not also be a click")
	}
	if _, dragging := pointer.Dragging(); dragging {
		t.Fatal("drag remained active after release")
	}
}

func TestPointerGesturesLastOneTick(t *testing.T) {
	pointer := newPointer()
	pointer.advance(pointerSample{position: image.Pt(10, 10), down: true})
	pointer.advance(pointerSample{position: image.Pt(10, 10)})
	pointer.advance(pointerSample{position: image.Pt(10, 10)})

	if _, clicked := pointer.Click(); clicked {
		t.Fatal("click remained set after the following update")
	}
}
