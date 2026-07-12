package ui

import (
	"image"
	"testing"
)

func TestPointerClick(t *testing.T) {
	pointer := newPointer()
	pointer.advance(pointerSample{position: image.Pt(10, 10), down: true})
	pointer.advance(pointerSample{position: image.Pt(11, 10)})

	if !pointer.Click(image.Rect(5, 5, 20, 20)) {
		t.Fatal("Click() = false; want true")
	}
	if _, dragging := pointer.Dragging(); dragging {
		t.Fatal("a click must not also be a drag")
	}
}

func TestPointerClickRequiresPressAndReleaseInsideBounds(t *testing.T) {
	bounds := image.Rect(10, 10, 30, 30)

	tests := []struct {
		name    string
		start   image.Point
		end     image.Point
		clicked bool
	}{
		{name: "both inside", start: image.Pt(12, 12), end: image.Pt(20, 20), clicked: true},
		{name: "press outside", start: image.Pt(5, 5), end: image.Pt(20, 20)},
		{name: "release outside", start: image.Pt(20, 20), end: image.Pt(35, 35)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pointer := newPointer()
			pointer.advance(pointerSample{position: test.start, down: true})
			pointer.advance(pointerSample{position: test.end})
			if clicked := pointer.Click(bounds); clicked != test.clicked {
				t.Fatalf("Click() = %t; want %t", clicked, test.clicked)
			}
		})
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
	if pointer.Click(image.Rect(0, 0, 100, 100)) {
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

	if pointer.Click(image.Rect(0, 0, 100, 100)) {
		t.Fatal("click remained set after the following update")
	}
}
