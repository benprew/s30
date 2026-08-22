package domain

import (
	"archive/zip"
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"net/http"
	"net/http/httptest"
	"path"
	"sync/atomic"
	"testing"
)

func TestLoadCardImagesFromArchiveCachesEveryValidImage(t *testing.T) {
	cardImages.Clear()
	t.Cleanup(cardImages.Clear)

	archive := cardImageArchive(t, map[string]image.Image{
		"tst-1-200-first-card.jpg":   solidCardImage(color.RGBA{R: 0xff, A: 0xff}),
		"tst-2-200-second-card.jpeg": solidCardImage(color.RGBA{G: 0xff, A: 0xff}),
	})

	loaded, err := loadCardImagesFromArchive(archive)
	if err != nil {
		t.Fatalf("loadCardImagesFromArchive() error = %v", err)
	}
	if loaded != 2 {
		t.Fatalf("loadCardImagesFromArchive() loaded = %d, want 2", loaded)
	}
	cards, _ := CardImageCacheStats()
	if cards != 2 {
		t.Fatalf("CardImageCacheStats() cards = %d, want 2", cards)
	}
	for _, id := range []string{"tst-1-first-card", "tst-2-second-card"} {
		cached, ok := cardImages.Load(id)
		if !ok {
			t.Errorf("card image %q was not cached", id)
			continue
		}
		if got := cachedImageBounds(cached); got.Dx() != CardFullWidth {
			t.Errorf("cached image %q width = %d, want %d", id, got.Dx(), CardFullWidth)
		}
	}
}

func TestFetchAndCacheCardImageUsesURLWhenImageIsMissing(t *testing.T) {
	cardImages.Clear()
	t.Cleanup(cardImages.Clear)

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "image/jpeg")
		if err := jpeg.Encode(w, solidCardImage(color.RGBA{B: 0xff, A: 0xff}), nil); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	t.Cleanup(server.Close)

	card := &Card{
		CardName:      "Remote Card",
		BorderCropURL: server.URL,
		cardID:        "tst-3-remote-card",
	}
	fetchAndCacheCardImage(card)

	if requests.Load() != 1 {
		t.Fatalf("HTTP requests = %d, want 1", requests.Load())
	}
	if _, ok := cardImages.Load(card.cardID); !ok {
		t.Fatal("downloaded card image was not cached")
	}
}

func TestPreloadCardImagesOnlyFetchesPriorityCards(t *testing.T) {
	cardImages.Clear()
	t.Cleanup(cardImages.Clear)

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "image/jpeg")
		if err := jpeg.Encode(w, solidCardImage(color.RGBA{B: 0xff, A: 0xff}), nil); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	t.Cleanup(server.Close)

	priority := []*Card{
		{
			CardName:      "Priority 1",
			BorderCropURL: server.URL,
			cardID:        "tst-p1",
		},
		{
			CardName:      "Priority 2",
			BorderCropURL: server.URL,
			cardID:        "tst-p2",
		},
	}

	PreloadCardImages(priority)

	if requests.Load() != 2 {
		t.Fatalf("HTTP requests = %d, want 2", requests.Load())
	}
	if _, ok := cardImages.Load("tst-p1"); !ok {
		t.Fatal("priority card 1 was not cached")
	}
	if _, ok := cardImages.Load("tst-p2"); !ok {
		t.Fatal("priority card 2 was not cached")
	}
}

func TestCollectPriorityCardsOnlyIncludesPlayerCards(t *testing.T) {
	playerCard1 := &Card{cardID: "player-card-1", CardName: "Player Card 1"}
	playerCard2 := &Card{cardID: "player-card-2", CardName: "Player Card 2"}
	bonusCard := &Card{cardID: "bonus-card", CardName: "Bonus Card"}

	collection := NewCardCollection()
	collection.AddCardToDeck(playerCard1, 0, 4)
	collection.AddCardToDeck(playerCard2, 1, 2)

	player := &Player{
		Character: Character{
			CardCollection: collection,
		},
		BonusDuelCards: []*Card{bonusCard},
	}

	priority := CollectPriorityCards(player)

	if len(priority) != 3 {
		t.Fatalf("CollectPriorityCards() returned %d cards, want 3", len(priority))
	}

	seen := make(map[string]bool)
	for _, c := range priority {
		seen[c.cardID] = true
	}

	for _, wantID := range []string{"player-card-1", "player-card-2", "bonus-card"} {
		if !seen[wantID] {
			t.Errorf("missing expected card ID %q in priority cards", wantID)
		}
	}
}

func TestCollectPriorityCardsNilPlayer(t *testing.T) {
	if got := CollectPriorityCards(nil); got != nil {
		t.Fatalf("CollectPriorityCards(nil) = %v, want nil", got)
	}
}

func cardImageArchive(t *testing.T, images map[string]image.Image) []byte {
	t.Helper()

	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, img := range images {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatalf("create ZIP entry: %v", err)
		}
		var encodeErr error
		switch path.Ext(name) {
		case ".jpg", ".jpeg":
			encodeErr = jpeg.Encode(entry, img, nil)
		default:
			encodeErr = png.Encode(entry, img)
		}
		if encodeErr != nil {
			t.Fatalf("encode ZIP entry: %v", encodeErr)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close ZIP: %v", err)
	}
	return buffer.Bytes()
}

func solidCardImage(fill color.Color) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, CardFullWidth, 342))
	for y := range img.Bounds().Dy() {
		for x := range img.Bounds().Dx() {
			img.Set(x, y, fill)
		}
	}
	return img
}

func cachedImageBounds(cached any) image.Rectangle {
	return cached.(interface{ Bounds() image.Rectangle }).Bounds()
}
