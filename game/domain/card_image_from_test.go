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
	card := &Card{CardName: "Forest", Artworks: []Artwork{{ID: "alt-art-card", CardName: "Forest"}}}
	CacheCardImage(card.DefaultArtwork().ID, image.NewRGBA(image.Rect(0, 0, CardFullWidth, 342)))

	printed, err := card.CardImage(CardViewArtMini)
	if err != nil {
		t.Fatalf("CardImage returned an error: %v", err)
	}

	alt := altArtSource()
	fromAlt, err := card.cardImageFrom(alt, "leb-300-forest", CardViewArtMini)
	if err != nil {
		t.Fatalf("cardImageFrom returned an error: %v", err)
	}

	if printed == fromAlt {
		t.Error("alternate art reused the printed art's cache entry")
	}
}

func TestCardImageFromCachesByArtID(t *testing.T) {
	t.Cleanup(ClearCardImageCache)
	card := &Card{CardName: "Swamp", Artworks: []Artwork{{ID: "alt-art-cached", CardName: "Swamp"}}}
	alt := altArtSource()

	first, err := card.cardImageFrom(alt, "leb-297-swamp", CardViewArtMini)
	if err != nil {
		t.Fatalf("cardImageFrom returned an error: %v", err)
	}
	second, err := card.cardImageFrom(alt, "leb-297-swamp", CardViewArtMini)
	if err != nil {
		t.Fatalf("cardImageFrom returned an error: %v", err)
	}

	if first != second {
		t.Error("the same art and view were derived twice instead of being cached")
	}

	other, err := card.cardImageFrom(altArtSource(), "unh-138-swamp", CardViewArtMini)
	if err != nil {
		t.Fatalf("cardImageFrom returned an error: %v", err)
	}
	if other == first {
		t.Error("two different art files share one cache entry")
	}
}

func TestCardImageFromViewSizes(t *testing.T) {
	t.Cleanup(ClearCardImageCache)
	card := &Card{CardName: "Island", Artworks: []Artwork{{ID: "alt-art-sizes", CardName: "Island"}}}
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
		img, err := card.cardImageFrom(alt, "leb-294-island", tt.view)
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
	card := &Card{CardName: "Mountain", Artworks: []Artwork{{ID: "alt-art-full", CardName: "Mountain"}}}
	alt := altArtSource()

	got, err := card.cardImageFrom(alt, "leb-299-mountain", CardViewFull)
	if err != nil {
		t.Fatalf("cardImageFrom returned an error: %v", err)
	}
	if got != alt {
		t.Error("CardViewFull did not return the supplied source image")
	}
}

func TestCardImageFromNilSourceUsesPrintedArt(t *testing.T) {
	t.Cleanup(ClearCardImageCache)
	card := &Card{CardName: "Plains", Artworks: []Artwork{{ID: "alt-art-nil-source", CardName: "Plains"}}}
	CacheCardImage(card.DefaultArtwork().ID, image.NewRGBA(image.Rect(0, 0, CardFullWidth, 342)))

	viaNil, err := card.cardImageFrom(nil, card.DefaultArtwork().ID, CardViewArtMini)
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

	full, err := card.cardImageFrom(nil, card.DefaultArtwork().ID, CardViewFull)
	if err != nil {
		t.Fatalf("cardImageFrom returned an error: %v", err)
	}
	if full != card.fullCardImage() {
		t.Error("a nil source did not fall back to the printed art for CardViewFull")
	}
}

func TestCardImageFromRejectsUnknownView(t *testing.T) {
	t.Cleanup(ClearCardImageCache)
	card := &Card{CardName: "Forest", Artworks: []Artwork{{ID: "alt-art-bad-view", CardName: "Forest"}}}

	if _, err := card.cardImageFrom(altArtSource(), "leb-300-forest", CardView(-1)); err == nil {
		t.Error("cardImageFrom accepted an unknown view")
	}
}
