package domain

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io/fs"
	"sync"
	"testing"
	"testing/fstest"
)

// countingFS records how often each path is opened, so a test can prove the
// land art cache stops going back to disk.
type countingFS struct {
	inner fs.FS
	mu    sync.Mutex
	opens map[string]int
}

func newCountingFS(inner fs.FS) *countingFS {
	return &countingFS{inner: inner, opens: make(map[string]int)}
}

func (c *countingFS) Open(name string) (fs.File, error) {
	c.mu.Lock()
	c.opens[name]++
	c.mu.Unlock()
	return c.inner.Open(name)
}

func (c *countingFS) count(name string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.opens[name]
}

func landArtPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	img.Set(0, 0, color.RGBA{R: 0xff, A: 0xff})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("failed to encode png: %v", err)
	}
	return buf.Bytes()
}

func landArtJPEG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatalf("failed to encode jpeg: %v", err)
	}
	return buf.Bytes()
}

func TestLoadImageNormalizesWidth(t *testing.T) {
	lib := ScanLandArt(fstest.MapFS{
		"Forest.full.png": &fstest.MapFile{Data: landArtPNG(t, 490, 684)},
	})

	img, err := lib.loadImage("Forest.full.png")
	if err != nil {
		t.Fatalf("loadImage returned an error: %v", err)
	}
	if got := img.Bounds().Dx(); got != CardFullWidth {
		t.Errorf("width = %d, want %d", got, CardFullWidth)
	}
	if got := img.Bounds().Dy(); got != 342 {
		t.Errorf("height = %d, want 342 (aspect ratio preserved)", got)
	}
}

func TestLoadImageJPEG(t *testing.T) {
	lib := ScanLandArt(fstest.MapFS{
		"swamp.jpg": &fstest.MapFile{Data: landArtJPEG(t, 490, 684)},
	})

	img, err := lib.loadImage("swamp.jpg")
	if err != nil {
		t.Fatalf("loadImage returned an error: %v", err)
	}
	if got := img.Bounds().Dx(); got != CardFullWidth {
		t.Errorf("width = %d, want %d", got, CardFullWidth)
	}
}

func TestLoadImageLeavesCorrectWidthAlone(t *testing.T) {
	lib := ScanLandArt(fstest.MapFS{
		"island.png": &fstest.MapFile{Data: landArtPNG(t, CardFullWidth, 400)},
	})

	img, err := lib.loadImage("island.png")
	if err != nil {
		t.Fatalf("loadImage returned an error: %v", err)
	}
	if got := img.Bounds().Dy(); got != 400 {
		t.Errorf("height = %d, want 400 unchanged", got)
	}
}

func TestLoadImageCachesDecodedArt(t *testing.T) {
	counting := newCountingFS(fstest.MapFS{
		"mountain.png": &fstest.MapFile{Data: landArtPNG(t, 490, 684)},
	})
	lib := ScanLandArt(counting)

	first, err := lib.loadImage("mountain.png")
	if err != nil {
		t.Fatalf("first loadImage returned an error: %v", err)
	}
	second, err := lib.loadImage("mountain.png")
	if err != nil {
		t.Fatalf("second loadImage returned an error: %v", err)
	}

	if first != second {
		t.Error("loadImage decoded the same art twice instead of caching it")
	}
	if got := counting.count("mountain.png"); got != 1 {
		t.Errorf("art file opened %d times, want 1", got)
	}
}

func TestLoadImageCachesFailure(t *testing.T) {
	counting := newCountingFS(fstest.MapFS{
		"plains.png": &fstest.MapFile{Data: []byte("this is not an image")},
	})
	lib := ScanLandArt(counting)

	if _, err := lib.loadImage("plains.png"); err == nil {
		t.Fatal("loadImage accepted a corrupt file")
	}
	if _, err := lib.loadImage("plains.png"); err == nil {
		t.Fatal("loadImage accepted a corrupt file on the second call")
	}

	if got := counting.count("plains.png"); got != 1 {
		t.Errorf("corrupt file opened %d times, want 1: a broken file must not be retried every frame", got)
	}
}

func TestLoadImageMissingPath(t *testing.T) {
	lib := ScanLandArt(fstest.MapFS{})

	if _, err := lib.loadImage("forest.png"); err == nil {
		t.Error("loadImage accepted a path that is not in the filesystem")
	}
}

func TestLoadImageWithoutLibrary(t *testing.T) {
	var missing *LandArtLibrary
	if _, err := missing.loadImage("forest.png"); err == nil {
		t.Error("loadImage on a nil library returned no error")
	}

	noFS := ScanLandArt(nil)
	if _, err := noFS.loadImage("forest.png"); err == nil {
		t.Error("loadImage with no filesystem returned no error")
	}
}
