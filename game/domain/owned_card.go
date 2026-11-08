package domain

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"uuid"

	"github.com/hajimehoshi/ebiten/v2"
)

// OwnedCard stores the identity and artwork of one copy. DeckIndex is -1 for a spare.
type OwnedCard struct {
	ID        string `json:"id"`
	*Card     `json:"card"`
	ArtworkID string `json:"artwork_id"`
	DeckIndex int    `json:"deck_index"`
}

// MarshalJSON includes the copy's fields instead of using Card.MarshalJSON.
func (c OwnedCard) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID        string `json:"id"`
		Card      *Card  `json:"card"`
		ArtworkID string `json:"artwork_id"`
		DeckIndex int    `json:"deck_index"`
	}{c.ID, c.Card, c.ArtworkID, c.DeckIndex})
}

// UnmarshalJSON also accepts card definitions from older shop saves.
func (c *OwnedCard) UnmarshalJSON(data []byte) error {
	var saved struct {
		ID        string `json:"id"`
		Card      *Card  `json:"card"`
		ArtworkID string `json:"artwork_id"`
		DeckIndex int    `json:"deck_index"`
	}
	if err := json.Unmarshal(data, &saved); err != nil {
		return err
	}
	if saved.Card == nil {
		var legacy Card
		if err := json.Unmarshal(data, &legacy); err != nil {
			return err
		}
		if legacy.CardID() != "" || legacy.Name() != "" {
			saved.Card = FindCard(legacy.CardID(), legacy.Name())
			saved.DeckIndex = -1
		}
	} else {
		saved.Card = FindCard(saved.Card.CardID(), saved.Card.Name())
	}
	if saved.Card == nil {
		*c = OwnedCard{ID: saved.ID, ArtworkID: saved.ArtworkID, DeckIndex: saved.DeckIndex}
		return nil
	}
	if saved.ID == "" {
		saved.ID = uuid.New().String()
	}
	if saved.ArtworkID == "" {
		var old struct {
			CardID      string
			SetID       string
			CollectorNo string
			Card        json.RawMessage
		}
		if err := json.Unmarshal(data, &old); err != nil {
			return err
		}
		legacyID := old.CardID
		if len(old.Card) > 0 {
			var reference struct{ CardID, SetID, CollectorNo string }
			if err := json.Unmarshal(old.Card, &reference); err != nil {
				return err
			}
			legacyID = reference.CardID
			if legacyID == "" && reference.SetID != "" {
				legacyID = (ArtworkJSON{SetID: reference.SetID, CollectorNo: reference.CollectorNo}).toArtwork(saved.Card.Name()).ID
			}
		}
		if legacyID == "" && old.SetID != "" {
			legacyID = (ArtworkJSON{SetID: old.SetID, CollectorNo: old.CollectorNo}).toArtwork(saved.Card.Name()).ID
		}
		saved.ArtworkID = saved.Card.DefaultArtwork().ID
		if legacyID != "" && legacyID != saved.Card.Name() {
			saved.ArtworkID = legacyID
		}
	}
	*c = OwnedCard{ID: saved.ID, Card: saved.Card, ArtworkID: saved.ArtworkID, DeckIndex: saved.DeckIndex}
	return nil
}

// ImageLoaded reports whether the selected printing has an image ready.
func (c *OwnedCard) ImageLoaded() bool {
	if c == nil || c.Card == nil {
		return false
	}
	return c.Artwork().ImageLoaded()
}

// NewOwnedCard assigns artwork once, when the copy is created.
func NewOwnedCard(card *Card) *OwnedCard {
	owned := newDefaultOwnedCard(card)
	arts := card.Artworks
	if len(arts) > 0 {
		owned.ArtworkID = arts[rand.Intn(len(arts))].ID
	}
	return owned
}

func newDefaultOwnedCard(card *Card) *OwnedCard {
	return &OwnedCard{ID: uuid.New().String(), Card: card, ArtworkID: card.DefaultArtwork().ID, DeckIndex: -1}
}

// CardImage returns this copy's artwork. Missing images use the default art.
func (c *OwnedCard) CardImage(view CardView) (*ebiten.Image, error) {
	if c == nil || c.Card == nil {
		return nil, fmt.Errorf("no owned card to draw")
	}
	return c.Artwork().CardImage(view)
}

// Artwork returns the selected printing. Removed printings use the default artwork.
func (c *OwnedCard) Artwork() Artwork {
	for _, art := range c.Artworks {
		if art.ID == c.ArtworkID {
			return art
		}
	}
	return c.DefaultArtwork()
}
