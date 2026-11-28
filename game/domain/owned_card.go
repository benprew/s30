package domain

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

// OwnedCard identifies one printing of a card.
type OwnedCard struct {
	card      *Card
	artworkID string
}

type printingKey struct {
	card      *Card
	artworkID string
}

var printings = struct {
	sync.Mutex
	byKey map[printingKey]*OwnedCard
}{byKey: make(map[printingKey]*OwnedCard)}

// Printing returns the shared printing. An empty ID selects the default artwork.
func (c *Card) Printing(artworkID string) *OwnedCard {
	if c == nil {
		return nil
	}
	if artworkID == "" {
		artworkID = c.DefaultArtwork().ID
	}

	key := printingKey{card: c, artworkID: artworkID}
	printings.Lock()
	defer printings.Unlock()
	printing := printings.byKey[key]
	if printing == nil {
		printing = &OwnedCard{card: c, artworkID: artworkID}
		printings.byKey[key] = printing
	}
	return printing
}

// Card returns the rules definition for this printing.
func (p *OwnedCard) Card() *Card {
	if p == nil {
		return nil
	}
	return p.card
}

func (p *OwnedCard) Name() string { return p.card.Name() }

func (p *OwnedCard) ArtworkID() string { return p.artworkID }

// Artwork returns the printing's artwork. Removed printings use the default artwork.
func (p *OwnedCard) Artwork() Artwork {
	for _, art := range p.card.Artworks {
		if art.ID == p.artworkID {
			return art
		}
	}
	return p.card.DefaultArtwork()
}

// ImageLoaded reports whether the printing has an image ready.
func (p *OwnedCard) ImageLoaded() bool {
	return p.Card() != nil && p.Artwork().ImageLoaded()
}

// CardImage returns the image for this printing.
func (p *OwnedCard) CardImage(view CardView) (*ebiten.Image, error) {
	if p.Card() == nil {
		return nil, fmt.Errorf("no printing to draw")
	}
	return p.Artwork().CardImage(view)
}

// RandomPrinting selects an artwork for a card awarded by the game.
func (c *Card) RandomPrinting() *OwnedCard {
	if len(c.Artworks) == 0 {
		return c.Printing("")
	}
	return c.Printing(c.Artworks[rand.Intn(len(c.Artworks))].ID)
}

type printingJSON struct {
	CardID    string `json:"card_id"`
	ArtworkID string `json:"artwork_id"`
}

func (p *OwnedCard) reference() printingJSON {
	return printingJSON{CardID: p.card.CardID(), ArtworkID: p.artworkID}
}

// MarshalJSON saves the IDs that identify the printing.
func (p *OwnedCard) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.reference())
}

type legacyCardReference struct {
	CardID      string
	CardName    string
	SetID       string
	CollectorNo string
}

type savedPrinting struct {
	printingJSON
	Card *legacyCardReference `json:"card"`
	legacyCardReference
}

func (saved savedPrinting) bareReference() bool {
	return saved.printingJSON.CardID == "" && saved.Card == nil &&
		(saved.legacyCardReference.CardID != "" || saved.CardName != "")
}

func (saved savedPrinting) resolve(card *Card) *OwnedCard {
	legacy := saved.legacyCardReference
	if saved.Card != nil {
		legacy = *saved.Card
	}
	if card == nil {
		if saved.printingJSON.CardID != "" {
			card = FindCard(saved.printingJSON.CardID, "")
		} else {
			card = FindCard(legacy.CardID, legacy.CardName)
		}
	}
	if card == nil {
		return nil
	}
	artworkID := saved.ArtworkID
	if artworkID == "" {
		artworkID = legacy.artworkID(card)
		if artworkID == "" && saved.Card != nil {
			artworkID = saved.artworkID(card)
		}
	}
	return card.Printing(artworkID)
}

func (r legacyCardReference) artworkID(card *Card) string {
	if r.CardID != "" && r.CardID != card.CardID() {
		return r.CardID
	}
	if r.SetID != "" {
		return (ArtworkJSON{SetID: r.SetID, CollectorNo: r.CollectorNo}).toArtwork(card.Name()).ID
	}
	return ""
}
