package screens

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"testing"
	"testing/fstest"

	"github.com/benprew/s30/game/domain"
)

func editDeckLandArtFixture(t *testing.T, names ...string) *domain.LandArtLibrary {
	t.Helper()
	files := fstest.MapFS{}
	for i, name := range names {
		img := image.NewRGBA(image.Rect(0, 0, 490, 680))
		img.Set(0, 0, color.RGBA{R: uint8(i + 1), A: 0xff})
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			t.Fatalf("failed to encode %s: %v", name, err)
		}
		files[name] = &fstest.MapFile{Data: buf.Bytes()}
	}
	return domain.ScanLandArt(files)
}

func editDeckWithLandArt(lib *domain.LandArtLibrary, seed int64) *EditDeckScreen {
	var sel *domain.LandArtSelection
	if lib != nil {
		sel = domain.NewLandArtSelectionFor(lib, rand.New(rand.NewSource(seed)))
	}
	return &EditDeckScreen{landArt: sel}
}

// The editor resolves the magnifier's card with FindCardByName while a tile
// holds its group's representative card, so the two are different pointers that
// must still land on the same art.
func TestEditDeckMagnifierMatchesTheTile(t *testing.T) {
	t.Cleanup(domain.ClearCardImageCache)
	s := editDeckWithLandArt(editDeckLandArtFixture(t,
		"forest1.png", "forest2.png", "forest3.png", "forest4.png"), 5)

	tileCard := domain.FindCardByName("Forest")
	if tileCard == nil {
		t.Fatal("Forest is missing from the card database")
	}
	magnifierCard := &domain.Card{CardName: "Forest"}

	tile, err := s.cardArt(tileCard, domain.CardViewFull)
	if err != nil {
		t.Fatalf("cardArt returned an error: %v", err)
	}
	magnified, err := s.cardArt(magnifierCard, domain.CardViewFull)
	if err != nil {
		t.Fatalf("cardArt returned an error: %v", err)
	}

	if tile == nil {
		t.Fatal("no art was drawn for the tile")
	}
	if tile != magnified {
		t.Error("the magnifier shows different land art than the tile it is magnifying")
	}
}

func TestEditDeckRollsNewArtEachSession(t *testing.T) {
	t.Cleanup(domain.ClearCardImageCache)
	lib := editDeckLandArtFixture(t, "forest1.png", "forest2.png", "forest3.png", "forest4.png")
	forest := domain.FindCardByName("Forest")
	if forest == nil {
		t.Fatal("Forest is missing from the card database")
	}

	first, err := editDeckWithLandArt(lib, 1).cardArt(forest, domain.CardViewFull)
	if err != nil {
		t.Fatalf("cardArt returned an error: %v", err)
	}

	// Seeds are fixed, so this is deterministic: some later session differs.
	sawDifferent := false
	for seed := int64(2); seed < 8; seed++ {
		next, err := editDeckWithLandArt(lib, seed).cardArt(forest, domain.CardViewFull)
		if err != nil {
			t.Fatalf("cardArt returned an error: %v", err)
		}
		if next != first {
			sawDifferent = true
			break
		}
	}
	if !sawDifferent {
		t.Error("every editing session reused the same art, so it is never re-rolled")
	}
}

func TestEditDeckWithoutLandArtDrawsPrintedArt(t *testing.T) {
	t.Cleanup(domain.ClearCardImageCache)
	s := editDeckWithLandArt(nil, 1)

	forest := domain.FindCardByName("Forest")
	if forest == nil {
		t.Fatal("Forest is missing from the card database")
	}
	domain.CacheCardImage(forest.CardID(), image.NewRGBA(image.Rect(0, 0, domain.CardFullWidth, 342)))

	want, err := forest.CardImage(domain.CardViewArtMini)
	if err != nil {
		t.Fatalf("CardImage returned an error: %v", err)
	}
	got, err := s.cardArt(forest, domain.CardViewArtMini)
	if err != nil {
		t.Fatalf("cardArt returned an error: %v", err)
	}
	if got != want {
		t.Error("with no land art installed, a Forest did not draw its printed art")
	}
}

func TestEditDeckCardArtRejectsNilCard(t *testing.T) {
	s := editDeckWithLandArt(editDeckLandArtFixture(t, "forest1.png"), 1)

	if _, err := s.cardArt(nil, domain.CardViewArtMini); err == nil {
		t.Error("cardArt accepted a nil card")
	}
}
