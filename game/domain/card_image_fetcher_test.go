package domain

import (
	"archive/zip"
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/image/draw"
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

func TestResizeMethodsComparison(t *testing.T) {
	const srcW, srcH = 488, 680
	const targetWidth = CardFullWidth

	srcImg := generateTestCardImage(srcW, srcH)

	outDir := filepath.Join("testdata", "resize_comparison")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("failed to create output dir: %v", err)
	}

	savePNG := func(name string, img image.Image) {
		f, err := os.Create(filepath.Join(outDir, name))
		if err != nil {
			t.Fatalf("failed to create %s: %v", name, err)
		}
		defer f.Close()
		if err := png.Encode(f, img); err != nil {
			t.Fatalf("failed to encode %s: %v", name, err)
		}
	}
	savePNG("original_488x680.png", srcImg)

	methods := []struct {
		name       string
		scaler     draw.Interpolator
		outputFile string
	}{
		{"CatmullRom", draw.CatmullRom, "catmull_rom_200.png"},
		{"BiLinear", draw.BiLinear, "bi_linear_200.png"},
		{"ApproxBiLinear", draw.ApproxBiLinear, "approx_bi_linear_200.png"},
		{"NearestNeighbor", draw.NearestNeighbor, "nearest_neighbor_200.png"},
	}

	results := make(map[string]image.Image)
	timings := make(map[string]time.Duration)

	const iterations = 20
	for _, m := range methods {
		var out image.Image
		start := time.Now()
		for range iterations {
			out = resizeToWidthWithInterpolator(srcImg, targetWidth, m.scaler)
		}
		dur := time.Since(start) / iterations
		timings[m.name] = dur
		results[m.name] = out
		savePNG(m.outputFile, out)

		if out.Bounds().Dx() != targetWidth {
			t.Errorf("%s output width = %d, want %d", m.name, out.Bounds().Dx(), targetWidth)
		}
	}

	ref := results["CatmullRom"]
	t.Logf("\n=== Image Resizing Methods Comparison (488x680 -> 200x%d) ===", ref.Bounds().Dy())
	t.Logf("%-16s | %-12s | %-10s | %-12s | %-10s | %-10s", "Method", "Avg Time/Img", "Speedup", "RMSE (vs CR)", "PSNR (dB)", "Max Diff")
	t.Logf("----------------------------------------------------------------------------------------")

	catmullTime := timings["CatmullRom"].Seconds()
	for _, m := range methods {
		img := results[m.name]
		rmse, psnr, maxDiff := compareImages(ref, img)
		speedup := catmullTime / timings[m.name].Seconds()
		t.Logf("%-16s | %10.2f ms | %8.2fx | %12.2f | %8.2f dB | %10.0f",
			m.name,
			float64(timings[m.name].Microseconds())/1000.0,
			speedup,
			rmse,
			psnr,
			maxDiff,
		)
	}
	t.Logf("Saved output images to: %s", outDir)
}

func BenchmarkResizeMethods(b *testing.B) {
	srcImg := generateTestCardImage(488, 680)
	const targetWidth = CardFullWidth

	b.Run("CatmullRom", func(b *testing.B) {
		for b.Loop() {
			_ = resizeToWidthWithInterpolator(srcImg, targetWidth, draw.CatmullRom)
		}
	})
	b.Run("BiLinear", func(b *testing.B) {
		for b.Loop() {
			_ = resizeToWidthWithInterpolator(srcImg, targetWidth, draw.BiLinear)
		}
	})
	b.Run("ApproxBiLinear", func(b *testing.B) {
		for b.Loop() {
			_ = resizeToWidthWithInterpolator(srcImg, targetWidth, draw.ApproxBiLinear)
		}
	})
	b.Run("NearestNeighbor", func(b *testing.B) {
		for b.Loop() {
			_ = resizeToWidthWithInterpolator(srcImg, targetWidth, draw.NearestNeighbor)
		}
	})
}

func generateTestCardImage(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			r := uint8((x * 255) / w)
			g := uint8((y * 255) / h)
			b := uint8(((x + y) * 128) / (w + h))
			if (x+y)%16 < 4 {
				r = 255 - r
				g = 255 - g
				b = 255 - b
			}
			if x < 15 || x >= w-15 || y < 15 || y >= h-15 {
				r, g, b = 20, 20, 20
			}
			cx, cy := w/2, h/2
			dx, dy := x-cx, y-cy
			if dx*dx+dy*dy < 4000 {
				r, g, b = 240, 200, 50
			}
			img.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}
	return img
}

func compareImages(ref, img image.Image) (rmse, psnr, maxDiff float64) {
	bRef := ref.Bounds()
	bImg := img.Bounds()
	if bRef != bImg {
		return math.Inf(1), 0, 255
	}

	var sumSqDiff float64
	var maxD float64
	totalPixels := float64(bRef.Dx() * bRef.Dy() * 3)

	for y := bRef.Min.Y; y < bRef.Max.Y; y++ {
		for x := bRef.Min.X; x < bRef.Max.X; x++ {
			r1, g1, b1, _ := ref.At(x, y).RGBA()
			r2, g2, b2, _ := img.At(x, y).RGBA()

			dr := float64(int(r1>>8) - int(r2>>8))
			dg := float64(int(g1>>8) - int(g2>>8))
			db := float64(int(b1>>8) - int(b2>>8))

			sumSqDiff += dr*dr + dg*dg + db*db

			if math.Abs(dr) > maxD {
				maxD = math.Abs(dr)
			}
			if math.Abs(dg) > maxD {
				maxD = math.Abs(dg)
			}
			if math.Abs(db) > maxD {
				maxD = math.Abs(db)
			}
		}
	}

	mse := sumSqDiff / totalPixels
	rmse = math.Sqrt(mse)
	if mse > 0 {
		psnr = 20*math.Log10(255) - 10*math.Log10(mse)
	} else {
		psnr = 99.99
	}
	return rmse, psnr, maxDiff
}

func TestResizeRealMTGCardImages(t *testing.T) {
	cardNames := []string{"Black Lotus", "Shivan Dragon", "Lightning Bolt", "Birds of Paradise"}
	targetWidth := CardFullWidth // 200

	outDir := filepath.Join("testdata", "resize_comparison")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("failed to create output dir: %v", err)
	}

	savePNG := func(filename string, img image.Image) {
		f, err := os.Create(filepath.Join(outDir, filename))
		if err != nil {
			t.Fatalf("failed to create %s: %v", filename, err)
		}
		defer f.Close()
		if err := png.Encode(f, img); err != nil {
			t.Fatalf("failed to encode %s: %v", filename, err)
		}
	}

	methods := []struct {
		name   string
		scaler draw.Interpolator
		suffix string
	}{
		{"CatmullRom", draw.CatmullRom, "catmull_rom"},
		{"BiLinear", draw.BiLinear, "bi_linear"},
		{"ApproxBiLinear", draw.ApproxBiLinear, "approx_bi_linear"},
		{"NearestNeighbor", draw.NearestNeighbor, "nearest_neighbor"},
	}

	client := &http.Client{Timeout: 10 * time.Second}

	for _, name := range cardNames {
		card := FindCardByName(name)
		if card == nil {
			t.Logf("Card %s not found, skipping", name)
			continue
		}
		if card.BorderCropURL == "" {
			t.Logf("Card %s has no BorderCropURL, skipping", name)
			continue
		}

		resp, err := client.Get(card.BorderCropURL)
		if err != nil {
			t.Logf("Failed to fetch %s from %s: %v (skipping real fetch)", name, card.BorderCropURL, err)
			continue
		}
		img, format, err := image.Decode(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			t.Logf("Failed to decode image for %s: %v", name, err)
			continue
		}

		b := img.Bounds()
		safeName := strings.ToLower(strings.ReplaceAll(name, " ", "_"))
		savePNG(fmt.Sprintf("%s_original.png", safeName), img)

		t.Logf("\n==================== %s (%s, %dx%d -> %dx%d) ====================",
			name, format, b.Dx(), b.Dy(), targetWidth, int(float64(b.Dy())*(float64(targetWidth)/float64(b.Dx()))))
		t.Logf("%-16s | %-12s | %-10s | %-12s | %-10s | %-10s", "Method", "Avg Time/Img", "Speedup", "RMSE (vs CR)", "PSNR (dB)", "Max Diff")
		t.Logf("----------------------------------------------------------------------------------------")

		results := make(map[string]image.Image)
		timings := make(map[string]time.Duration)

		const iterations = 20
		for _, m := range methods {
			var out image.Image
			start := time.Now()
			for range iterations {
				out = resizeToWidthWithInterpolator(img, targetWidth, m.scaler)
			}
			dur := time.Since(start) / iterations
			timings[m.name] = dur
			results[m.name] = out
			savePNG(fmt.Sprintf("%s_%s.png", safeName, m.suffix), out)
		}

		ref := results["CatmullRom"]
		catmullTime := timings["CatmullRom"].Seconds()
		for _, m := range methods {
			res := results[m.name]
			rmse, psnr, maxDiff := compareImages(ref, res)
			speedup := catmullTime / timings[m.name].Seconds()
			t.Logf("%-16s | %10.2f ms | %8.2fx | %12.2f | %8.2f dB | %10.0f",
				m.name,
				float64(timings[m.name].Microseconds())/1000.0,
				speedup,
				rmse,
				psnr,
				maxDiff,
			)
		}
	}
}
