package domain

import (
	"errors"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

// landArtKey identifies one assignment. The land is part of the key because a
// permanent can change which basic land it is: a land turned into an Island
// should draw Island art, not whatever art it was given as a Forest.
type landArtKey struct {
	instance string
	land     string
}

// LandArtSelection assigns alternate art to basic lands for one context - a
// duel, or the deck editor - and remembers each assignment for as long as that
// context lives. Instances keyed differently may draw different art, and a new
// selection rolls everything again, so a land keeps its art for a whole duel
// but is free to look different in the next one.
//
// A selection belongs to the goroutine that draws it and is not safe for
// concurrent use.
type LandArtSelection struct {
	lib    *LandArtLibrary
	rng    *rand.Rand
	chosen map[landArtKey]string
}

// NewLandArtSelection begins a context over the land art found at startup.
func NewLandArtSelection() *LandArtSelection {
	return newLandArtSelection(LandArt(), rand.New(rand.NewSource(rand.Int63())))
}

// NewLandArtSelectionFor begins a context over a specific library and random
// source, rather than the art found at startup. Screens want
// NewLandArtSelection; this is for callers that need repeatable rolls over art
// they supply themselves.
func NewLandArtSelectionFor(lib *LandArtLibrary, rng *rand.Rand) *LandArtSelection {
	return newLandArtSelection(lib, rng)
}

func newLandArtSelection(lib *LandArtLibrary, rng *rand.Rand) *LandArtSelection {
	return &LandArtSelection{lib: lib, rng: rng, chosen: make(map[landArtKey]string)}
}

// CardImage returns the art for one card instance, identified by key. Anything
// with no alternate art available - every card that is not a basic land, and
// basic lands the player supplied no art for - falls through to the printed
// art, so callers need no conditionals of their own.
func (s *LandArtSelection) CardImage(card *Card, key string, view CardView) (*ebiten.Image, error) {
	if card == nil {
		return nil, errors.New("no card to draw art for")
	}

	artPath, source := s.artFor(card, key)
	if source == nil {
		return card.CardImage(view)
	}
	return card.cardImageFrom(source, artPath, view)
}

// artFor resolves this instance's assignment, making one the first time the
// instance is seen. A nil image means there is no alternate art to draw, either
// because none was offered or because the file would not decode.
func (s *LandArtSelection) artFor(card *Card, key string) (string, *ebiten.Image) {
	if s == nil {
		return "", nil
	}

	paths := s.lib.Paths(card.CardName)
	if len(paths) == 0 {
		return "", nil
	}

	assignment := landArtKey{instance: key, land: card.CardName}
	artPath, assigned := s.chosen[assignment]
	if !assigned {
		artPath = paths[s.rng.Intn(len(paths))]
		s.chosen[assignment] = artPath
	}
	return artPath, s.lib.loadImage(artPath)
}
