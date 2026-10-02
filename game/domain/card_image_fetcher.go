package domain

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"time"
)

var cardArtClient = &http.Client{Timeout: 30 * time.Second}

func fetchCardArt(card *Card) (image.Image, error) {
	if card.ArtURL == "" {
		return nil, fmt.Errorf("no art URL for card %s", card.CardName)
	}
	resp, err := cardArtClient.Get(card.ArtURL)
	if err != nil {
		return nil, fmt.Errorf("fetch art for %s: %w", card.CardName, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch art for %s: HTTP %d", card.CardName, resp.StatusCode)
	}
	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("decode art for %s: %w", card.CardName, err)
	}
	return img, nil
}
