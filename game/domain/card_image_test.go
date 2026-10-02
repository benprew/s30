package domain

import (
	"github.com/hajimehoshi/ebiten/v2"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCardImageSharesVariantsAndRefreshesArt(t *testing.T) {
	ClearCardImageCache()
	t.Cleanup(ClearCardImageCache)
	card := &Card{cardID: "variant-test", CardName: "Test"}
	placeholder, err := card.CardImage(CardViewArtMini)
	if err != nil {
		t.Fatal(err)
	}
	copyCard := *card
	repeated, _ := copyCard.CardImage(CardViewArtMini)
	if repeated != placeholder {
		t.Fatal("placeholder was resized again")
	}
	CacheCardImage(card.cardID, image.NewRGBA(image.Rect(0, 0, CardFullWidth, 342)))
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
	CacheCardImage(card.cardID, image.NewRGBA(image.Rect(0, 0, CardFullWidth, 342)))
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
	first := &Card{cardID: "printing-one", CardName: "Same name"}
	second := &Card{cardID: "printing-two", CardName: "Same name"}
	for _, card := range []*Card{first, second} {
		CacheCardImage(card.cardID, image.NewRGBA(image.Rect(0, 0, CardFullWidth, 342)))
	}
	a, _ := first.CardImage(CardViewArtMini)
	b, _ := second.CardImage(CardViewArtMini)
	if a == b {
		t.Fatal("different printings share an image")
	}
}

func TestCardImageVariantCacheLimit(t *testing.T) {
	ClearCardImageCache()
	t.Cleanup(ClearCardImageCache)
	card := &Card{cardID: "cache-limit", CardName: "Test"}
	CacheCardImage(card.cardID, image.NewRGBA(image.Rect(0, 0, CardFullWidth, 342)))
	oldest, _ := card.CardImage(CardViewFull)
	for i := 2; i <= cardImageVariantLimit; i++ {
		card.CardName = string(rune(i))
		if _, err := card.CardImage(CardViewFull); err != nil {
			t.Fatal(err)
		}
	}
	card.CardName = "Test"
	retained, _ := card.CardImage(CardViewFull)
	if retained != oldest {
		t.Fatal("cache removed an image before reaching its limit")
	}
	card.CardName = "extra"
	_, _ = card.CardImage(CardViewFull)
	card.CardName = "Test"
	retained, _ = card.CardImage(CardViewFull)
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
	card := &Card{cardID: "presets", CardName: "Presets"}
	CacheCardImage(card.cardID, image.NewRGBA(image.Rect(0, 0, CardFullWidth, 342)))
	for _, tc := range []struct {
		view CardView
		size image.Point
	}{
		{CardViewFull, image.Point(CardViewFull)},
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

func TestFullCardImageUsesRenderer(t *testing.T) {
	ClearCardImageCache()
	t.Cleanup(ClearCardImageCache)
	card := &Card{cardID: "rendered-full", CardName: "Test", TypeLine: "Creature", Text: "Flying", Power: 2, Toughness: 3}
	CacheCardImage(card.cardID, image.NewRGBA(image.Rect(0, 0, CardFullWidth, 342)))
	raw, _ := cardImages.Load(card.cardID)
	full := card.fullCardImage()
	if full == raw {
		t.Fatal("full image uses the downloaded card instead of the renderer")
	}
	again, _ := card.CardImage(CardViewFull)
	if full != again {
		t.Fatal("full image does not share the rendered card")
	}
}

func TestResizedImageSharesRenderedVariants(t *testing.T) {
	ClearCardImageCache()
	t.Cleanup(ClearCardImageCache)
	card := &Card{cardID: "resized-rendered", CardName: "Test"}
	CacheCardImage(card.cardID, image.NewRGBA(image.Rect(0, 0, CardFullWidth, 342)))
	first, err := card.CardImage(CardViewArtOnly)
	if err != nil {
		t.Fatal(err)
	}
	again, err := card.CardImage(CardViewArtOnly)
	if err != nil {
		t.Fatal(err)
	}
	if first != again {
		t.Fatal("resized image was not shared")
	}
	CacheCardImage(card.cardID, image.NewRGBA(image.Rect(0, 0, CardFullWidth, 342)))
	replaced, err := card.CardImage(CardViewArtOnly)
	if err != nil {
		t.Fatal(err)
	}
	if first == replaced {
		t.Fatal("resized image retained old artwork")
	}
	if _, err := card.CardImage(CardView{}); err == nil {
		t.Fatal("invalid view was accepted")
	}
}

func TestCardImageMarksMissingArtAndRefreshesEveryView(t *testing.T) {
	ClearCardImageCache()
	t.Cleanup(ClearCardImageCache)
	card := &Card{cardID: "blank-views", CardName: "Test"}
	views := []CardView{CardViewFull, CardViewFullMini, CardViewArtOnly, CardViewArtMini}
	previous := make(map[CardView]*ebiten.Image)
	for _, view := range views {
		img, err := card.CardImage(view)
		if err != nil {
			t.Fatal(err)
		}
		previous[view] = img
		key := cardImageVariantKey{id: card.cardID, name: card.CardName, view: view}
		if !cardImageVariants.entries[key].Value.(cardImageVariant).blank {
			t.Fatal("missing art was not marked blank")
		}
	}
	CacheCardImage(card.cardID, image.NewRGBA(image.Rect(0, 0, 186, 118)))
	for _, view := range views {
		img, err := card.CardImage(view)
		if err != nil {
			t.Fatal(err)
		}
		if img == previous[view] {
			t.Fatal("blank view was retained")
		}
		key := cardImageVariantKey{id: card.cardID, name: card.CardName, view: view}
		if cardImageVariants.entries[key].Value.(cardImageVariant).blank {
			t.Fatal("loaded view was marked blank")
		}
	}
}

func TestCardImageFetchesOnceAndRefreshesBlankViews(t *testing.T) {
	ClearCardImageCache()
	t.Cleanup(ClearCardImageCache)
	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		close(started)
		<-release
		if err := png.Encode(w, image.NewRGBA(image.Rect(0, 0, 186, 118))); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	t.Cleanup(unblock)
	card := &Card{cardID: "async-art", CardName: "Test", ArtURL: server.URL}
	first, err := card.CardImage(CardViewFull)
	if err != nil {
		t.Fatal(err)
	}
	<-started
	mini, err := card.CardImage(CardViewArtMini)
	if err != nil {
		t.Fatal(err)
	}
	again, _ := card.CardImage(CardViewFull)
	if again != first {
		t.Fatal("blank view was rendered again while fetching")
	}
	cardImageVariants.Lock()
	done := cardImageVariants.fetches[card.cardID].done
	cardImageVariants.Unlock()
	unblock()
	<-done
	full, _ := card.CardImage(CardViewFull)
	loadedMini, _ := card.CardImage(CardViewArtMini)
	if full == first || loadedMini == mini {
		t.Fatal("views did not refresh after art arrived")
	}
	if requests.Load() != 1 {
		t.Fatalf("requests = %d, want 1", requests.Load())
	}
}

func TestCardImageRetainsBlankAfterFetchFailure(t *testing.T) {
	ClearCardImageCache()
	t.Cleanup(ClearCardImageCache)
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)
	card := &Card{cardID: "failed-art", CardName: "Test", ArtURL: server.URL}
	PreloadCardImages([]*Card{card})
	first, err := card.CardImage(CardViewFull)
	if err != nil {
		t.Fatal(err)
	}
	again, _ := card.CardImage(CardViewFull)
	PreloadCardImages([]*Card{card})
	if first != again || card.ImageLoaded() {
		t.Fatal("failed fetch did not retain a blank rendered card")
	}
	if requests.Load() != 1 {
		t.Fatal("failed fetch was retried without delay")
	}
	cardImageVariants.Lock()
	cardImageVariants.fetches[card.cardID].retryAfter = time.Now().Add(-time.Second)
	cardImageVariants.Unlock()
	PreloadCardImages([]*Card{card})
	if requests.Load() != 2 {
		t.Fatal("failed fetch was not retried after the delay")
	}
}

func TestClearCardImageCacheDiscardsInflightArt(t *testing.T) {
	ClearCardImageCache()
	t.Cleanup(ClearCardImageCache)
	release := make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		<-release
		if err := png.Encode(w, image.NewRGBA(image.Rect(0, 0, 186, 118))); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	t.Cleanup(unblock)
	card := &Card{cardID: "clear-inflight", CardName: "Test", ArtURL: server.URL}
	if _, err := card.CardImage(CardViewFull); err != nil {
		t.Fatal(err)
	}
	cardImageVariants.Lock()
	done := cardImageVariants.fetches[card.cardID].done
	cardImageVariants.Unlock()
	ClearCardImageCache()
	unblock()
	<-done
	if card.ImageLoaded() {
		t.Fatal("old fetch restored cleared art")
	}
}

func TestCardImageRendersOnlyRequestedView(t *testing.T) {
	ClearCardImageCache()
	t.Cleanup(ClearCardImageCache)
	card := &Card{cardID: "direct-view", CardName: "Test"}
	if _, err := card.CardImage(CardViewArtMini); err != nil {
		t.Fatal(err)
	}
	if len(cardImageVariants.entries) != 1 {
		t.Fatal("view was rendered through a cached full card")
	}
}
