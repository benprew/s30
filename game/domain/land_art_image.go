package domain

import (
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

// loadImage decodes one art file, normalized to the width the rest of the card
// pipeline expects so that every CardView derives from it unchanged. Decoding
// is deferred to the first time the art is drawn, which keeps a player's
// hundred-swamp folder off the startup path and out of memory until it is used.
//
// A nil result means the art is unavailable and the printed card should be
// drawn instead. Unreadable files are cached as nil so that one bad file in a
// player's pack costs a single decode and a single warning, rather than a
// fresh attempt on every frame the land is drawn.
func (l *LandArtLibrary) loadImage(artPath string) *ebiten.Image {
	if l == nil || l.fsys == nil {
		return nil
	}
	if cached, ok := l.images.Load(artPath); ok {
		img, _ := cached.(*ebiten.Image)
		return img
	}

	img := l.decodeImage(artPath)
	l.images.Store(artPath, img)
	return img
}

// cachedImages returns how many decoded land art images the library is holding
// onto. Files that failed to decode are cached too, but retain no image and so
// cost no memory to report.
func (l *LandArtLibrary) cachedImages() int {
	if l == nil {
		return 0
	}
	count := 0
	l.images.Range(func(_, value any) bool {
		if img, _ := value.(*ebiten.Image); img != nil {
			count++
		}
		return true
	})
	return count
}

func (l *LandArtLibrary) decodeImage(artPath string) *ebiten.Image {
	file, err := l.fsys.Open(artPath)
	if err != nil {
		fmt.Printf("WARN: Failed to open land art %s: %v\n", artPath, err)
		return nil
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		fmt.Printf("WARN: Failed to decode land art %s: %v\n", artPath, err)
		return nil
	}
	if img.Bounds().Empty() {
		fmt.Printf("WARN: Land art %s has no pixels\n", artPath)
		return nil
	}

	if img.Bounds().Dx() != CardFullWidth {
		img = resizeToWidth(img, CardFullWidth)
	}
	return ebiten.NewImageFromImage(img)
}
