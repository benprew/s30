package domain

import (
	"archive/zip"
	"bytes"
	"container/list"
	"fmt"
	"image"
	"image/color"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/benprew/s30/assets"
	"github.com/hajimehoshi/ebiten/v2"
)

const cardImageVariantLimit = 1024
const cardArtRetryDelay = 30 * time.Second

var cardImages sync.Map
var frameCache sync.Map
var manaIconsCache []*ebiten.Image
var setIconsCache []*ebiten.Image
var initIconsOnce sync.Once

type cardImageVariantKey struct {
	id   string
	name string
	view CardView
}

type cardImageVariant struct {
	key   cardImageVariantKey
	image *ebiten.Image
	blank bool
}

type cardArtFetch struct {
	done       chan struct{}
	retryAfter time.Time
}

var cardImageVariants = struct {
	sync.Mutex
	entries    map[cardImageVariantKey]*list.Element
	order      list.List
	fetches    map[string]*cardArtFetch
	generation uint64
}{entries: make(map[cardImageVariantKey]*list.Element), fetches: make(map[string]*cardArtFetch)}

// CardImage returns a shared card image. Callers must not change this image.
func (card *Card) CardImage(view CardView) (*ebiten.Image, error) {
	if view.X <= 0 || view.Y <= 0 {
		return nil, fmt.Errorf("invalid card view: %v", view)
	}
	cardImageVariants.Lock()
	defer cardImageVariants.Unlock()
	card.startArtFetchLocked()
	return card.cardImageLocked(view)
}

func (card *Card) cardImageLocked(view CardView) (*ebiten.Image, error) {
	key := cardImageVariantKey{id: card.cardID, name: card.CardName, view: view}
	if element := cardImageVariants.entries[key]; element != nil {
		cardImageVariants.order.MoveToFront(element)
		return element.Value.(cardImageVariant).image, nil
	}
	var art image.Image
	if cached, ok := cardImages.Load(card.cardID); ok {
		art = cached.(image.Image)
	}
	img := renderCardImage(card, art, view)
	if img == nil {
		return nil, fmt.Errorf("render card %s", card.CardName)
	}
	element := cardImageVariants.order.PushFront(cardImageVariant{key: key, image: img, blank: art == nil})
	cardImageVariants.entries[key] = element
	if cardImageVariants.order.Len() > cardImageVariantLimit {
		oldest := cardImageVariants.order.Back()
		delete(cardImageVariants.entries, oldest.Value.(cardImageVariant).key)
		cardImageVariants.order.Remove(oldest)
	}
	return img, nil
}

func (card *Card) fullCardImage() *ebiten.Image {
	img, _ := card.CardImage(CardViewFull)
	return img
}

func (card *Card) startArtFetchLocked() <-chan struct{} {
	if card.ArtURL == "" {
		return nil
	}
	if _, loaded := cardImages.Load(card.cardID); loaded {
		return nil
	}
	if state := cardImageVariants.fetches[card.cardID]; state != nil {
		if state.retryAfter.IsZero() {
			return state.done
		}
		if time.Now().Before(state.retryAfter) {
			return nil
		}
	}
	state := &cardArtFetch{done: make(chan struct{})}
	cardImageVariants.fetches[card.cardID] = state
	generation := cardImageVariants.generation
	snapshot := *card
	go func() {
		art, err := fetchCardArt(&snapshot)
		cardImageVariants.Lock()
		defer cardImageVariants.Unlock()
		defer close(state.done)
		if generation != cardImageVariants.generation {
			return
		}
		if err != nil {
			state.retryAfter = time.Now().Add(cardArtRetryDelay)
			return
		}
		cacheCardArtLocked(snapshot.cardID, art)
		delete(cardImageVariants.fetches, snapshot.cardID)
	}()
	return state.done
}

func cacheCardArtLocked(id string, art image.Image) {
	cardImages.Store(id, art)
	for key, element := range cardImageVariants.entries {
		if key.id == id {
			cardImageVariants.order.Remove(element)
			delete(cardImageVariants.entries, key)
		}
	}
}

// CacheCardImage stores artwork and clears the card's rendered views.
func CacheCardImage(id string, art image.Image) {
	if art == nil || art.Bounds().Empty() {
		return
	}
	cardImageVariants.Lock()
	defer cardImageVariants.Unlock()
	cacheCardArtLocked(id, art)
}

// ClearCardImageCache clears artwork, rendered views, and fetch state.
func ClearCardImageCache() {
	cardImageVariants.Lock()
	defer cardImageVariants.Unlock()
	cardImages.Clear()
	clear(cardImageVariants.entries)
	cardImageVariants.order.Init()
	clear(cardImageVariants.fetches)
	cardImageVariants.generation++
}

// CardImageCacheStats returns the artwork and blank rendered image counts.
func CardImageCacheStats() (cards, blanks int) {
	cardImageVariants.Lock()
	defer cardImageVariants.Unlock()
	cardImages.Range(func(_, _ any) bool { cards++; return true })
	for _, element := range cardImageVariants.entries {
		if element.Value.(cardImageVariant).blank {
			blanks++
		}
	}
	return cards, blanks
}

// PreloadCardImages waits for artwork for the supplied cards.
func PreloadCardImages(priorityCards []*Card) {
	const numWorkers = 4
	ch := make(chan *Card)
	var wg sync.WaitGroup
	for range min(numWorkers, len(priorityCards)) {
		wg.Go(func() {
			for card := range ch {
				cardImageVariants.Lock()
				done := card.startArtFetchLocked()
				cardImageVariants.Unlock()
				if done != nil {
					<-done
				}
			}
		})
	}
	seen := make(map[string]bool)
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

		CacheCardImage(id, ExtractArtSubImage(img))
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

// loadFrameImage loads and caches a card frame from embedded assets.
func loadFrameImage(filename string) image.Image {
	if cached, ok := frameCache.Load(filename); ok {
		return cached.(image.Image)
	}

	data, err := assets.CardFramesFS.ReadFile("art/card/" + filename)
	if err != nil {
		fmt.Printf("WARN: Failed to read frame %s: %v\n", filename, err)
		return nil
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		fmt.Printf("WARN: Failed to decode frame %s: %v\n", filename, err)
		return nil
	}

	frameCache.Store(filename, img)
	return img
}

func initIcons() {
	initIconsOnce.Do(func() {
		// Load mana symbols
		manaData, err := assets.CardFramesFS.ReadFile("art/card/Manasymbols.pic.png")
		if err == nil {
			if sheet, _, decodeErr := image.Decode(bytes.NewReader(manaData)); decodeErr == nil {
				manaIconsCache = make([]*ebiten.Image, 19)
				for i := range 19 {
					iconRGBA := image.NewRGBA(image.Rect(0, 0, 18, 18))
					for y := range 18 {
						for x := range 18 {
							srcX := i*18 + x
							srcColor := color.RGBAModel.Convert(sheet.At(srcX, y)).(color.RGBA)
							dx := float64(x) - 8.5
							dy := float64(y) - 8.5
							dist := dx*dx + dy*dy
							if dist <= 64 && (srcColor.R != 0 || srcColor.G != 0 || srcColor.B != 0) {
								srcColor.A = 255
								iconRGBA.Set(x, y, srcColor)
							} else if dist <= 72 && (srcColor.R != 0 || srcColor.G != 0 || srcColor.B != 0) {
								alpha := uint8(255 * (1.0 - (dist-64.0)/8.0))
								srcColor.A = alpha
								iconRGBA.Set(x, y, srcColor)
							}
						}
					}
					manaIconsCache[i] = ebiten.NewImageFromImage(iconRGBA)
				}
			}
		}

		// Load set symbols
		setData, err := assets.CardFramesFS.ReadFile("art/card/Cardsets.pic.png")
		if err == nil {
			if sheet, _, decodeErr := image.Decode(bytes.NewReader(setData)); decodeErr == nil {
				setIconsCache = make([]*ebiten.Image, 22)
				for i := range 22 {
					iconRGBA := image.NewRGBA(image.Rect(0, 0, 15, 15))
					for y := range 15 {
						for x := range 15 {
							srcX := i*15 + x
							srcColor := color.RGBAModel.Convert(sheet.At(srcX, y)).(color.RGBA)
							if srcColor.R != 0 || srcColor.G != 0 || srcColor.B != 0 {
								srcColor.A = 255
								iconRGBA.Set(x, y, srcColor)
							}
						}
					}
					setIconsCache[i] = ebiten.NewImageFromImage(iconRGBA)
				}
			}
		}
	})
}
