package elements

import (
	"image"
	"testing"
)

func TestButtonClicksWhenGestureStartsAndEndsInside(t *testing.T) {
	button := &Button{}
	bounds := image.Rect(10, 10, 30, 30)
	if !button.updatePointer(bounds, image.Pt(20, 20), false, true) {
		t.Fatal("button did not handle click")
	}
	if button.State != StateClicked {
		t.Fatalf("button state = %v; want StateClicked", button.State)
	}
}

func TestButtonDoesNotClickWithoutClickGesture(t *testing.T) {
	button := &Button{}
	bounds := image.Rect(10, 10, 30, 30)

	if button.updatePointer(bounds, image.Pt(20, 20), false, false) {
		t.Fatal("button handled pointer movement as a click")
	}
	if button.State != StateHover {
		t.Fatalf("button state = %v; want StateHover", button.State)
	}
}

func TestButtonShowsPressedStateWhilePointerIsDownInside(t *testing.T) {
	button := &Button{}
	bounds := image.Rect(10, 10, 30, 30)

	if button.updatePointer(bounds, image.Pt(20, 20), true, false) {
		t.Fatal("button handled pointer press as a completed click")
	}
	if button.State != StatePressed {
		t.Fatalf("button state = %v; want StatePressed", button.State)
	}
}
