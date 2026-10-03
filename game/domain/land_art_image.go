package domain

import (
	"errors"
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

// landArtEntry is a decode attempt. Failures are cached alongside successes so
// that one unreadable file in a player's art pack costs a single decode rather
// than a fresh one on every frame the land is drawn.
type landArtEntry struct {
	img *ebiten.Image
	err error
}

// loadImage decodes one art file, normalized to the width the rest of the card
// pipeline expects so that every CardView derives from it unchanged. Decoding
// is deferred to the first time the art is drawn, which keeps a player's
// hundred-swamp folder off the startup path and out of memory until it is used.
func (l *LandArtLibrary) loadImage(artPath string) (*ebiten.Image, error) {
	if l == nil || l.fsys == nil {
		return nil, errors.New("no land art folder")
	}
	if cached, ok := l.images.Load(artPath); ok {
		entry := cached.(landArtEntry)
		return entry.img, entry.err
	}

	entry := l.decodeImage(artPath)
	l.images.Store(artPath, entry)
	return entry.img, entry.err
}

func (l *LandArtLibrary) decodeImage(artPath string) landArtEntry {
	file, err := l.fsys.Open(artPath)
	if err != nil {
		return landArtEntry{err: fmt.Errorf("open land art %s: %w", artPath, err)}
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return landArtEntry{err: fmt.Errorf("decode land art %s: %w", artPath, err)}
	}
	if img.Bounds().Empty() {
		return landArtEntry{err: fmt.Errorf("land art %s has no pixels", artPath)}
	}

	if img.Bounds().Dx() != CardFullWidth {
		img = resizeToWidth(img, CardFullWidth)
	}
	return landArtEntry{img: ebiten.NewImageFromImage(img)}
}
