package domain

import (
	"archive/zip"
	"bytes"
	"fmt"
	"image"
	"image/color"
	"net/http"
	"path"
	"strings"
	"sync"

	_ "image/jpeg"
	_ "image/png"

	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/ui/fonts"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/draw"
)

var cardImages sync.Map
var fetchingSet sync.Map
var labeledBlankCards sync.Map

var blankCardOnce sync.Once
var blankCardImage *ebiten.Image

// CardImageCacheStats reports retained full-card and generated placeholder
// images. Profiling harnesses use these counts alongside heap and RSS samples.
func CardImageCacheStats() (cards, labeledPlaceholders int) {
	cardImages.Range(func(_, _ any) bool {
		cards++
		return true
	})
	labeledBlankCards.Range(func(_, _ any) bool {
		labeledPlaceholders++
		return true
	})
	return cards, labeledPlaceholders
}

func blankCard() *ebiten.Image {
	blankCardOnce.Do(func() {
		img, _, err := image.Decode(bytes.NewReader(assets.CardBlank_png))
		if err != nil {
			panic(fmt.Sprintf("failed to decode blank card: %v", err))
		}
		blankCardImage = ebiten.NewImageFromImage(img)
	})
	return blankCardImage
}

func labeledBlankCard(name string) *ebiten.Image {
	if cached, ok := labeledBlankCards.Load(name); ok {
		return cached.(*ebiten.Image)
	}

	base := blankCard()
	bounds := base.Bounds()
	img := ebiten.NewImage(bounds.Dx(), bounds.Dy())
	img.DrawImage(base, nil)

	face := &text.GoTextFace{Source: fonts.MtgFont, Size: 14}
	w, h := text.Measure(name, face, 0)

	x := (float64(bounds.Dx()) - w) / 2
	y := (float64(bounds.Dy()) - h) / 2

	opts := text.DrawOptions{}
	opts.GeoM.Translate(x, y)
	opts.ColorScale.ScaleWithColor(color.Black)
	text.Draw(img, name, face, &opts)

	labeledBlankCards.Store(name, img)
	return img
}

func resizeToWidthWithInterpolator(src image.Image, targetWidth int, scaler draw.Interpolator) image.Image {
	bounds := src.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()
	scale := float64(targetWidth) / float64(srcW)
	targetHeight := int(float64(srcH) * scale)
	dst := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))
	scaler.Scale(dst, dst.Bounds(), src, bounds, draw.Over, nil)
	return dst
}

func resizeToWidth(src image.Image, targetWidth int) image.Image {
	return resizeToWidthWithInterpolator(src, targetWidth, draw.ApproxBiLinear)
}

func cacheCardImage(id string, img image.Image) {
	if img.Bounds().Dx() != CardFullWidth {
		img = resizeToWidth(img, CardFullWidth)
	}
	cardImages.Store(id, ebiten.NewImageFromImage(img))
	InvalidateResizedCardCache(id)
}

// CacheCardImage stores a decoded image for a given card ID in the cache.
func CacheCardImage(id string, img image.Image) {
	cacheCardImage(id, img)
}

// ClearCardImageCache clears all cached card images.
func ClearCardImageCache() {
	cardImages.Clear()
	ClearResizedCardCache()
}

func cardIDFromImageFilename(name string) (string, bool) {
	name = path.Base(name)
	ext := strings.ToLower(path.Ext(name))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		return "", false
	}

	id := strings.TrimSuffix(name, path.Ext(name))
	if before, after, found := strings.Cut(id, "-200-"); found {
		id = before + "-" + after
	}
	return id, id != ""
}

func loadCardImagesFromArchive(data []byte) (int, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return 0, fmt.Errorf("open embedded card image archive: %w", err)
	}

	loaded := 0
	for _, file := range reader.File {
		id, ok := cardIDFromImageFilename(file.Name)
		if !ok {
			continue
		}

		entry, err := file.Open()
		if err != nil {
			fmt.Printf("WARN: Failed to open embedded card image %s: %v\n", file.Name, err)
			continue
		}
		img, _, decodeErr := image.Decode(entry)
		closeErr := entry.Close()
		if decodeErr != nil {
			fmt.Printf("WARN: Failed to decode embedded card image %s: %v\n", file.Name, decodeErr)
			continue
		}
		if closeErr != nil {
			fmt.Printf("WARN: Failed to close embedded card image %s: %v\n", file.Name, closeErr)
			continue
		}

		cacheCardImage(id, img)
		loaded++
	}

	return loaded, nil
}

// LoadEmbeddedCardImages loads all artwork included in the binary into the
// card image cache. Builds without embedded artwork have nothing to load.
func LoadEmbeddedCardImages() (int, error) {
	if len(assets.CardImagesZip) == 0 {
		return 0, nil
	}
	return loadCardImagesFromArchive(assets.CardImagesZip)
}

func fetchAndCacheCardImage(card *Card) {
	id := card.cardID
	imageURL := card.BorderCropURL
	if imageURL == "" {
		imageURL = card.ArtURL
	}
	if imageURL == "" {
		fmt.Printf("WARN: No image URL for card: %s\n", card.CardName)
		cardImages.Store(id, blankCard())
		return
	}

	resp, err := http.Get(imageURL)
	if err != nil {
		fmt.Printf("WARN: Failed to fetch card image for %s: %v\n", card.CardName, err)
		cardImages.Store(id, blankCard())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("WARN: HTTP %d fetching card image for %s\n", resp.StatusCode, card.CardName)
		cardImages.Store(id, blankCard())
		return
	}

	img, _, err := image.Decode(resp.Body)
	if err != nil {
		fmt.Printf("WARN: Failed to decode card image for %s: %v\n", card.CardName, err)
		cardImages.Store(id, blankCard())
		return
	}

	cacheCardImage(id, img)
}

// CollectPriorityCards returns all unique cards in the player's card collection
// and bonus duel cards for upfront preloading.
func CollectPriorityCards(player *Player) []*Card {
	if player == nil {
		return nil
	}

	seen := make(map[string]bool)
	var priority []*Card

	for card := range player.CardCollection {
		if card != nil && !seen[card.cardID] {
			seen[card.cardID] = true
			priority = append(priority, card)
		}
	}

	for _, card := range player.BonusDuelCards {
		if card != nil && !seen[card.cardID] {
			seen[card.cardID] = true
			priority = append(priority, card)
		}
	}

	return priority
}

func PreloadCardImages(priorityCards []*Card) {
	if len(priorityCards) == 0 {
		return
	}

	const numWorkers = 4
	ch := make(chan *Card, len(priorityCards))

	var wg sync.WaitGroup
	workers := min(numWorkers, len(priorityCards))
	for range workers {
		wg.Go(func() {
			for card := range ch {
				if _, loaded := cardImages.Load(card.cardID); loaded {
					continue
				}
				fetchAndCacheCardImage(card)
			}
		})
	}

	seen := make(map[string]bool, len(priorityCards))
	for _, card := range priorityCards {
		if card == nil || seen[card.cardID] {
			continue
		}
		seen[card.cardID] = true
		ch <- card
	}

	close(ch)
	wg.Wait()
}
