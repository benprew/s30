package domain

import (
	"container/list"
	"fmt"
	"image"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

const cardImageVariantLimit = 1024

type cardImageVariantKey struct {
	id   string
	name string
	view CardView
}

type cardImageVariant struct {
	key    cardImageVariantKey
	source *ebiten.Image
	image  *ebiten.Image
}

var cardImageVariants = struct {
	sync.Mutex
	entries map[cardImageVariantKey]*list.Element
	order   list.List
}{entries: make(map[cardImageVariantKey]*list.Element)}

// CardImage returns a shared image in one of the four display formats.
// Callers must not change the returned image.
func (card *Card) CardImage(view CardView) (*ebiten.Image, error) {
	var width, height int
	switch view {
	case CardViewFull:
		return card.fullCardImage(), nil
	case CardViewFullMini:
		width, height = CardFullMiniWidth, CardFullMiniHeight
	case CardViewArtOnly:
		width, height = CardArtWidth, CardArtHeight
	case CardViewArtMini:
		width, height = CardArtMiniWidth, CardArtMiniHeight
	default:
		return nil, fmt.Errorf("unknown card view: %d", view)
	}
	source := card.fullCardImage()
	key := cardImageVariantKey{id: card.cardID, name: card.CardName, view: view}
	cardImageVariants.Lock()
	defer cardImageVariants.Unlock()
	if element := cardImageVariants.entries[key]; element != nil {
		entry := element.Value.(cardImageVariant)
		if entry.source == source {
			cardImageVariants.order.MoveToFront(element)
			return entry.image, nil
		}
		cardImageVariants.order.Remove(element)
		delete(cardImageVariants.entries, key)
	}
	img := source
	if view == CardViewArtOnly || view == CardViewArtMini {
		img = source.SubImage(image.Rect(0, 0, source.Bounds().Dx(), cardSourceArtHeight)).(*ebiten.Image)
	}
	scaled := ebiten.NewImage(width, height)
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Scale(float64(width)/float64(img.Bounds().Dx()), float64(height)/float64(img.Bounds().Dy()))
	opts.Filter = ebiten.FilterLinear
	scaled.DrawImage(img, opts)
	img = scaled

	element := cardImageVariants.order.PushFront(cardImageVariant{key: key, source: source, image: img})
	cardImageVariants.entries[key] = element
	if cardImageVariants.order.Len() > cardImageVariantLimit {
		oldest := cardImageVariants.order.Back()
		delete(cardImageVariants.entries, oldest.Value.(cardImageVariant).key)
		cardImageVariants.order.Remove(oldest)
	}
	return img, nil
}

func clearCardImageVariants() {
	cardImageVariants.Lock()
	defer cardImageVariants.Unlock()
	clear(cardImageVariants.entries)
	cardImageVariants.order.Init()
}
