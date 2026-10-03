package domain

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func altArtSource() *ebiten.Image {
	return ebiten.NewImage(CardFullWidth, 342)
}

func TestCardImageFromSeparatesArtSources(t *testing.T) {
	t.Cleanup(ClearCardImageCache)
	card := &Card{cardID: "alt-art-card", CardName: "Forest"}
	CacheCardImage(card.cardID, image.NewRGBA(image.Rect(0, 0, CardFullWidth, 342)))

	printed, err := card.CardImage(CardViewArtMini)
	if err != nil {
		t.Fatalf("CardImage returned an error: %v", err)
	}

	alt := altArtSource()
	fromAlt, err := card.cardImageFrom(alt, "landArt/beta/Forest.full.png", CardViewArtMini)
	if err != nil {
		t.Fatalf("cardImageFrom returned an error: %v", err)
	}

	if printed == fromAlt {
		t.Error("alternate art reused the printed art's cache entry")
	}
}

func TestCardImageFromCachesByArtID(t *testing.T) {
	t.Cleanup(ClearCardImageCache)
	card := &Card{cardID: "alt-art-cached", CardName: "Swamp"}
	alt := altArtSource()

	first, err := card.cardImageFrom(alt, "landArt/swamp.png", CardViewArtMini)
	if err != nil {
		t.Fatalf("cardImageFrom returned an error: %v", err)
	}
	second, err := card.cardImageFrom(alt, "landArt/swamp.png", CardViewArtMini)
	if err != nil {
		t.Fatalf("cardImageFrom returned an error: %v", err)
	}

	if first != second {
		t.Error("the same art and view were derived twice instead of being cached")
	}

	other, err := card.cardImageFrom(altArtSource(), "landArt/swamp2.png", CardViewArtMini)
	if err != nil {
		t.Fatalf("cardImageFrom returned an error: %v", err)
	}
	if other == first {
		t.Error("two different art files share one cache entry")
	}
}

func TestCardImageFromViewSizes(t *testing.T) {
	t.Cleanup(ClearCardImageCache)
	card := &Card{cardID: "alt-art-sizes", CardName: "Island"}
	alt := altArtSource()

	tests := []struct {
		view CardView
		want image.Point
	}{
		{CardViewFullMini, image.Pt(CardFullMiniWidth, CardFullMiniHeight)},
		{CardViewArtOnly, image.Pt(CardArtWidth, CardArtHeight)},
		{CardViewArtMini, image.Pt(CardArtMiniWidth, CardArtMiniHeight)},
	}
	for _, tt := range tests {
		img, err := card.cardImageFrom(alt, "landArt/island.png", tt.view)
		if err != nil {
			t.Fatalf("cardImageFrom(view %d) returned an error: %v", tt.view, err)
		}
		if got := img.Bounds().Size(); got != tt.want {
			t.Errorf("view %d size = %v, want %v", tt.view, got, tt.want)
		}
	}
}

func TestCardImageFromFullReturnsSource(t *testing.T) {
	t.Cleanup(ClearCardImageCache)
	card := &Card{cardID: "alt-art-full", CardName: "Mountain"}
	alt := altArtSource()

	got, err := card.cardImageFrom(alt, "landArt/mountain.png", CardViewFull)
	if err != nil {
		t.Fatalf("cardImageFrom returned an error: %v", err)
	}
	if got != alt {
		t.Error("CardViewFull did not return the supplied source image")
	}
}

func TestCardImageFromNilSourceUsesPrintedArt(t *testing.T) {
	t.Cleanup(ClearCardImageCache)
	card := &Card{cardID: "alt-art-nil-source", CardName: "Plains"}
	CacheCardImage(card.cardID, image.NewRGBA(image.Rect(0, 0, CardFullWidth, 342)))

	viaNil, err := card.cardImageFrom(nil, card.cardID, CardViewArtMini)
	if err != nil {
		t.Fatalf("cardImageFrom returned an error: %v", err)
	}
	viaCardImage, err := card.CardImage(CardViewArtMini)
	if err != nil {
		t.Fatalf("CardImage returned an error: %v", err)
	}

	if viaNil != viaCardImage {
		t.Error("a nil source did not fall back to the card's printed art")
	}

	full, err := card.cardImageFrom(nil, card.cardID, CardViewFull)
	if err != nil {
		t.Fatalf("cardImageFrom returned an error: %v", err)
	}
	if full != card.fullCardImage() {
		t.Error("a nil source did not fall back to the printed art for CardViewFull")
	}
}

func TestCardImageFromRejectsUnknownView(t *testing.T) {
	t.Cleanup(ClearCardImageCache)
	card := &Card{cardID: "alt-art-bad-view", CardName: "Forest"}

	if _, err := card.cardImageFrom(altArtSource(), "landArt/forest.png", CardView(-1)); err == nil {
		t.Error("cardImageFrom accepted an unknown view")
	}
}
