package domain

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"net/http"
	"sync"

	"github.com/benprew/s30/assets"
	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/draw"
)

var cardImages sync.Map
var fetchingSet sync.Map

var blankCardOnce sync.Once
var blankCardImage *ebiten.Image

const cardTargetWidth = 300

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

func resizeToWidth(src image.Image, targetWidth int) image.Image {
	bounds := src.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()
	scale := float64(targetWidth) / float64(srcW)
	targetHeight := int(float64(srcH) * scale)
	dst := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, bounds, draw.Over, nil)
	return dst
}

func fetchAndCacheCardImage(card *Card) {
	id := card.OracleID
	if card.PngURL == "" {
		fmt.Printf("WARN: No PngURL for card: %s\n", card.CardName)
		cardImages.Store(id, blankCard())
		return
	}

	resp, err := http.Get(card.PngURL)
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

	img, err := png.Decode(resp.Body)
	if err != nil {
		fmt.Printf("WARN: Failed to decode PNG for %s: %v\n", card.CardName, err)
		cardImages.Store(id, blankCard())
		return
	}

	resized := resizeToWidth(img, cardTargetWidth)
	cardImages.Store(id, ebiten.NewImageFromImage(resized))
}

func CollectPriorityCards(player *Player) []*Card {
	seen := make(map[string]bool)
	var priority []*Card

	for card := range player.CardCollection {
		if !seen[card.OracleID] {
			seen[card.OracleID] = true
			priority = append(priority, card)
		}
	}

	for _, rogue := range Rogues {
		for card := range rogue.CardCollection {
			if !seen[card.OracleID] {
				seen[card.OracleID] = true
				priority = append(priority, card)
			}
		}
	}

	return priority
}

func PreloadCardImages(priorityCards []*Card) {
	const numWorkers = 6
	ch := make(chan *Card, 64)

	var wg sync.WaitGroup
	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for card := range ch {
				if _, loaded := cardImages.Load(card.OracleID); loaded {
					continue
				}
				fetchAndCacheCardImage(card)
			}
		}()
	}

	prioritySet := make(map[string]bool, len(priorityCards))
	for _, card := range priorityCards {
		prioritySet[card.OracleID] = true
		ch <- card
	}

	for _, card := range CARDS {
		if !prioritySet[card.OracleID] {
			ch <- card
		}
	}

	close(ch)
	wg.Wait()
}
