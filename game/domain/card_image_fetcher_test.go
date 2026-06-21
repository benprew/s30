package domain

import (
	"archive/zip"
	"bytes"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestLoadCardImagesFromArchiveCachesEveryValidImage(t *testing.T) {
	cardImages.Clear()
	t.Cleanup(cardImages.Clear)

	archive := cardImageArchive(t, map[string]image.Image{
		"tst-1-200-first-card.png":  solidCardImage(color.RGBA{R: 0xff, A: 0xff}),
		"tst-2-200-second-card.png": solidCardImage(color.RGBA{G: 0xff, A: 0xff}),
	})

	loaded, err := loadCardImagesFromArchive(archive)
	if err != nil {
		t.Fatalf("loadCardImagesFromArchive() error = %v", err)
	}
	if loaded != 2 {
		t.Fatalf("loadCardImagesFromArchive() loaded = %d, want 2", loaded)
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
		w.Header().Set("Content-Type", "image/png")
		if err := png.Encode(w, solidCardImage(color.RGBA{B: 0xff, A: 0xff})); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	t.Cleanup(server.Close)

	card := &Card{
		CardName: "Remote Card",
		PngURL:   server.URL,
		cardID:   "tst-3-remote-card",
	}
	fetchAndCacheCardImage(card)

	if requests.Load() != 1 {
		t.Fatalf("HTTP requests = %d, want 1", requests.Load())
	}
	if _, ok := cardImages.Load(card.cardID); !ok {
		t.Fatal("downloaded card image was not cached")
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
		if err := png.Encode(entry, img); err != nil {
			t.Fatalf("encode ZIP entry: %v", err)
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
