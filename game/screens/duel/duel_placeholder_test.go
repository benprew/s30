package duel

import "testing"

func TestPlaceholderSize_Untapped(t *testing.T) {
	w, h := placeholderSize(false)
	if w != fieldCardW || h != fieldCardH {
		t.Fatalf("placeholderSize(false) = %dx%d, want %dx%d", w, h, fieldCardW, fieldCardH)
	}
}

// A tapped permanent without art must lie on its side like a tapped card with
// art, or a tapped token reads as untapped.
func TestPlaceholderSize_Tapped(t *testing.T) {
	w, h := placeholderSize(true)
	if w != fieldCardH || h != fieldCardW {
		t.Fatalf("placeholderSize(true) = %dx%d, want %dx%d", w, h, fieldCardH, fieldCardW)
	}
}
