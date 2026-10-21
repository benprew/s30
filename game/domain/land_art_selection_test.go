package domain

import (
	"image"
	"math/rand"
	"testing"
	"testing/fstest"

	"github.com/hajimehoshi/ebiten/v2"
)

func seededSelection(lib *LandArtLibrary, seed int64) *LandArtSelection {
	return newLandArtSelection(lib, rand.New(rand.NewSource(seed)))
}

func landArtFixture(t *testing.T, names ...string) *LandArtLibrary {
	t.Helper()
	files := fstest.MapFS{}
	for _, name := range names {
		files[name] = &fstest.MapFile{Data: landArtPNG(t, 490, 684)}
	}
	return ScanLandArt(files)
}

func basicLandCard(t *testing.T, id, name string) *Card {
	t.Helper()
	card := &Card{cardID: id, CardName: name}
	CacheCardImage(id, image.NewRGBA(image.Rect(0, 0, CardFullWidth, 342)))
	return card
}

func variantCachedForArtID(artID string) bool {
	cardImageVariants.Lock()
	defer cardImageVariants.Unlock()
	for key := range cardImageVariants.entries {
		if key.id == artID {
			return true
		}
	}
	return false
}

func TestSelectionReusesArtForOneInstance(t *testing.T) {
	t.Cleanup(ClearCardImageCache)
	lib := landArtFixture(t, "forest1.png", "forest2.png", "forest3.png")
	sel := seededSelection(lib, 1)
	card := basicLandCard(t, "forest-card", "Forest")

	first, err := sel.CardImage(card, "permanent-a", CardViewArtMini)
	if err != nil {
		t.Fatalf("CardImage returned an error: %v", err)
	}
	second, err := sel.CardImage(card, "permanent-a", CardViewArtMini)
	if err != nil {
		t.Fatalf("CardImage returned an error: %v", err)
	}

	if first != second {
		t.Error("one instance was given different art on a second draw")
	}
}

func TestSelectionVariesArtAcrossInstances(t *testing.T) {
	t.Cleanup(ClearCardImageCache)
	lib := landArtFixture(t, "forest1.png", "forest2.png", "forest3.png")
	sel := seededSelection(lib, 1)
	card := basicLandCard(t, "forest-card", "Forest")

	distinct := map[*ebiten.Image]bool{}
	for i := range 20 {
		img, err := sel.CardImage(card, string(rune('a'+i)), CardViewArtMini)
		if err != nil {
			t.Fatalf("CardImage returned an error: %v", err)
		}
		if img == nil {
			t.Fatal("CardImage returned no image")
		}
		distinct[img] = true
	}

	if len(distinct) < 2 {
		t.Errorf("20 instances drew %d distinct art, want at least 2", len(distinct))
	}
}

func TestSelectionNewContextRerolls(t *testing.T) {
	t.Cleanup(ClearCardImageCache)
	lib := landArtFixture(t, "forest1.png", "forest2.png", "forest3.png", "forest4.png")
	card := basicLandCard(t, "forest-card", "Forest")

	// CardViewFull hands back the source image itself, so comparing these
	// compares which art file each instance was actually given.
	assignments := func(sel *LandArtSelection) []*ebiten.Image {
		var art []*ebiten.Image
		for i := range 20 {
			img, err := sel.CardImage(card, string(rune('a'+i)), CardViewFull)
			if err != nil {
				t.Fatalf("CardImage returned an error: %v", err)
			}
			art = append(art, img)
		}
		return art
	}

	first := assignments(seededSelection(lib, 1))
	second := assignments(seededSelection(lib, 99))

	same := true
	for i := range first {
		if first[i] != second[i] {
			same = false
			break
		}
	}
	if same {
		t.Error("a new selection reproduced the previous context's assignments")
	}
}

func TestSelectionRerollsWhenLandTypeChanges(t *testing.T) {
	t.Cleanup(ClearCardImageCache)
	lib := landArtFixture(t, "forest1.png", "island1.png")
	sel := seededSelection(lib, 1)
	forest := basicLandCard(t, "forest-card", "Forest")
	island := basicLandCard(t, "island-card", "Island")

	asForest, err := sel.CardImage(forest, "permanent-a", CardViewFull)
	if err != nil {
		t.Fatalf("CardImage returned an error: %v", err)
	}
	asIsland, err := sel.CardImage(island, "permanent-a", CardViewFull)
	if err != nil {
		t.Fatalf("CardImage returned an error: %v", err)
	}

	if asForest != lib.loadImage("forest1.png") {
		t.Error("the forest did not draw the forest art")
	}
	if asIsland != lib.loadImage("island1.png") {
		t.Error("a land that changed type kept the art of its previous type")
	}

	// Changing back restores the original assignment rather than rolling again.
	backToForest, err := sel.CardImage(forest, "permanent-a", CardViewFull)
	if err != nil {
		t.Fatalf("CardImage returned an error: %v", err)
	}
	if backToForest != asForest {
		t.Error("a land that changed type and back did not keep its original art")
	}
}

func TestSelectionFullViewUsesAlternateArt(t *testing.T) {
	t.Cleanup(ClearCardImageCache)
	lib := landArtFixture(t, "forest1.png")
	sel := seededSelection(lib, 1)
	card := basicLandCard(t, "forest-card", "Forest")

	full, err := sel.CardImage(card, "permanent-a", CardViewFull)
	if err != nil {
		t.Fatalf("CardImage returned an error: %v", err)
	}
	if full == card.fullCardImage() {
		t.Error("CardViewFull drew the printed art instead of the alternate art")
	}
	if full != lib.loadImage("forest1.png") {
		t.Error("CardViewFull did not return the loaded alternate art")
	}
}

func TestSelectionFallsBackForNonBasicLand(t *testing.T) {
	t.Cleanup(ClearCardImageCache)
	lib := landArtFixture(t, "forest1.png")
	sel := seededSelection(lib, 1)
	card := basicLandCard(t, "bolt-card", "Lightning Bolt")

	got, err := sel.CardImage(card, "spell-a", CardViewArtMini)
	if err != nil {
		t.Fatalf("CardImage returned an error: %v", err)
	}
	want, err := card.CardImage(CardViewArtMini)
	if err != nil {
		t.Fatalf("CardImage returned an error: %v", err)
	}
	if got != want {
		t.Error("a card that is not a basic land did not fall through to its printed art")
	}
}

func TestSelectionFallsBackWithoutArt(t *testing.T) {
	t.Cleanup(ClearCardImageCache)
	card := basicLandCard(t, "forest-card", "Forest")

	cases := map[string]*LandArtSelection{
		"empty library": seededSelection(ScanLandArt(fstest.MapFS{}), 1),
		"no library":    seededSelection(nil, 1),
		"nil selection": nil,
	}
	want, err := card.CardImage(CardViewArtMini)
	if err != nil {
		t.Fatalf("CardImage returned an error: %v", err)
	}

	for name, sel := range cases {
		got, err := sel.CardImage(card, "permanent-a", CardViewArtMini)
		if err != nil {
			t.Fatalf("%s: CardImage returned an error: %v", name, err)
		}
		if got != want {
			t.Errorf("%s: did not fall through to the printed art", name)
		}
	}
}

func TestSelectionUnreadableArtLeavesNoVariantEntry(t *testing.T) {
	t.Cleanup(ClearCardImageCache)
	lib := ScanLandArt(fstest.MapFS{
		"forest-broken.png": &fstest.MapFile{Data: []byte("this is not an image")},
	})
	sel := seededSelection(lib, 1)
	card := basicLandCard(t, "forest-card", "Forest")

	got, err := sel.CardImage(card, "permanent-a", CardViewArtMini)
	if err != nil {
		t.Fatalf("CardImage returned an error: %v", err)
	}
	want, err := card.CardImage(CardViewArtMini)
	if err != nil {
		t.Fatalf("CardImage returned an error: %v", err)
	}

	if got != want {
		t.Error("unreadable art did not fall through to the printed art")
	}
	if variantCachedForArtID("forest-broken.png") {
		t.Error("printed art was cached under the alternate art's key")
	}
}

func TestSelectionRejectsNilCard(t *testing.T) {
	sel := seededSelection(landArtFixture(t, "forest1.png"), 1)

	if _, err := sel.CardImage(nil, "permanent-a", CardViewArtMini); err == nil {
		t.Error("CardImage accepted a nil card")
	}
}
