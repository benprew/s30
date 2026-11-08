package domain

import (
	"image"
	"testing"
)

func TestCardImageSharesVariantsAndRefreshesArt(t *testing.T) {
	ClearCardImageCache()
	t.Cleanup(ClearCardImageCache)
	card := &Card{CardName: "Test", Artworks: []Artwork{{ID: "variant-test", CardName: "Test"}}}
	fetchingSet.Store(card.DefaultArtwork().ID, true)
	t.Cleanup(func() { fetchingSet.Delete(card.DefaultArtwork().ID) })
	placeholder, err := card.CardImage(CardViewArtMini)
	if err != nil {
		t.Fatal(err)
	}
	copyCard := *card
	repeated, _ := copyCard.CardImage(CardViewArtMini)
	if repeated != placeholder {
		t.Fatal("placeholder was resized again")
	}
	CacheCardImage(card.DefaultArtwork().ID, image.NewRGBA(image.Rect(0, 0, CardFullWidth, 342)))
	loaded, _ := card.CardImage(CardViewArtMini)
	if loaded == placeholder {
		t.Fatal("placeholder was retained after art arrived")
	}
	if loaded.Bounds().Dx() != 110 || loaded.Bounds().Dy() != 96 {
		t.Fatalf("unexpected bounds: %v", loaded.Bounds())
	}
	repeated, _ = copyCard.CardImage(CardViewArtMini)
	if repeated != loaded {
		t.Fatal("card copies do not share images")
	}
	full, _ := card.CardImage(CardViewFullMini)
	if full == loaded || full.Bounds().Dy() != 256 {
		t.Fatal("full view was not kept separate")
	}
	CacheCardImage(card.DefaultArtwork().ID, image.NewRGBA(image.Rect(0, 0, CardFullWidth, 342)))
	replaced, _ := card.CardImage(CardViewArtMini)
	if replaced == loaded {
		t.Fatal("replacement art was not used")
	}
	ClearCardImageCache()
	cleared, _ := card.CardImage(CardViewArtMini)
	if cleared == replaced {
		t.Fatal("cleared art was retained")
	}
}

func TestCardImageSeparatesPrintings(t *testing.T) {
	t.Cleanup(ClearCardImageCache)
	first := &Card{CardName: "Same name", Artworks: []Artwork{{ID: "printing-one", CardName: "Same name"}}}
	second := &Card{CardName: "Same name", Artworks: []Artwork{{ID: "printing-two", CardName: "Same name"}}}
	for _, card := range []*Card{first, second} {
		CacheCardImage(card.DefaultArtwork().ID, image.NewRGBA(image.Rect(0, 0, CardFullWidth, 342)))
	}
	a, _ := first.CardImage(CardViewArtMini)
	b, _ := second.CardImage(CardViewArtMini)
	if a == b {
		t.Fatal("different printings share an image")
	}
	if _, err := first.CardImage(CardView(-1)); err == nil {
		t.Fatal("invalid view was accepted")
	}
}

func TestCardImageVariantCacheLimit(t *testing.T) {
	ClearCardImageCache()
	t.Cleanup(ClearCardImageCache)
	art := Artwork{ID: "cache-limit", CardName: "Test"}
	CacheCardImage(art.ID, image.NewRGBA(image.Rect(0, 0, CardFullWidth, 342)))
	oldest, _ := art.CardImage(CardViewArtMini)
	for i := 2; i <= cardImageVariantLimit; i++ {
		art.CardName = string(rune(i))
		if _, err := art.CardImage(CardViewArtMini); err != nil {
			t.Fatal(err)
		}
	}
	art.CardName = "Test"
	retained, _ := art.CardImage(CardViewArtMini)
	if retained != oldest {
		t.Fatal("cache removed an image before reaching its limit")
	}
	art.CardName = "extra"
	_, _ = art.CardImage(CardViewArtMini)
	art.CardName = "Test"
	retained, _ = art.CardImage(CardViewArtMini)
	if retained != oldest {
		t.Fatal("cache removed a recently used image")
	}
	if len(cardImageVariants.entries) != cardImageVariantLimit {
		t.Fatal("cache exceeded its limit")
	}
}

func TestCardImagePresets(t *testing.T) {
	ClearCardImageCache()
	t.Cleanup(ClearCardImageCache)
	card := &Card{CardName: "Presets", Artworks: []Artwork{{ID: "presets", CardName: "Presets"}}}
	CacheCardImage(card.DefaultArtwork().ID, image.NewRGBA(image.Rect(0, 0, CardFullWidth, 342)))
	for _, tc := range []struct {
		view CardView
		size image.Point
	}{
		{CardViewFull, image.Pt(245, 342)},
		{CardViewFullMini, image.Pt(183, 256)},
		{CardViewArtOnly, image.Pt(147, 129)},
		{CardViewArtMini, image.Pt(110, 96)},
	} {
		img, err := card.CardImage(tc.view)
		if err != nil {
			t.Fatal(err)
		}
		if img.Bounds().Size() != tc.size {
			t.Fatalf("view %d size = %v, want %v", tc.view, img.Bounds().Size(), tc.size)
		}
		again, _ := card.CardImage(tc.view)
		if img != again {
			t.Fatalf("view %d was not shared", tc.view)
		}
	}
}
