package domain

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
)

// CardSet identifies a printing within a set.
type CardSet struct {
	SetID       string
	SetName     string
	SetType     string
	CollectorNo string
}

// Artwork identifies the images and metadata of one printing.
type Artwork struct {
	ID       string
	CardName string
	CardSet
	Artist        string
	PngURL        string
	ArtURL        string
	BorderCropURL string
}

// DefaultArtwork returns the first configured printing, or a named placeholder.
func (c *Card) DefaultArtwork() Artwork {
	if len(c.Artworks) > 0 {
		return c.Artworks[0]
	}
	return Artwork{ID: "unprinted:" + c.Name(), CardName: c.Name()}
}

// ImageLoaded reports whether the printing's image is cached.
func (a Artwork) ImageLoaded() bool {
	_, ok := cardImages.Load(a.ID)
	return ok
}

// CardImage returns the shared image for this printing and view.
func (a Artwork) CardImage(view CardView) (*ebiten.Image, error) {
	return a.imageFrom(nil, view)
}

// ArtworkJSON describes one printing without adding another rules definition.
type ArtworkJSON struct {
	SetID         string
	SetName       string
	CollectorNo   string
	Artist        string
	PngURL        string
	ArtURL        string
	BorderCropURL string
}

func (a ArtworkJSON) toArtwork(name string) Artwork {
	id := fmt.Sprintf("%s-%s-%s", a.SetID, a.CollectorNo, sanitizeFilename(name))
	if a.SetID == "" && a.CollectorNo == "" {
		id = "unprinted:" + name
	}
	return Artwork{ID: id, CardName: name, CardSet: CardSet{SetID: a.SetID, SetName: a.SetName, CollectorNo: a.CollectorNo}, Artist: a.Artist, PngURL: a.PngURL, ArtURL: a.ArtURL, BorderCropURL: a.BorderCropURL}
}
