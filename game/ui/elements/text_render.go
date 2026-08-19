package elements

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const maxRenderedTextCacheEntries = 256

type renderedTextCacheKey struct {
	Text             string
	Source           *text.GoTextFaceSource
	SizeMillis       int
	LineSpacingMilli int
	ColorRGBA        color.RGBA
	ScaleMillis      int
	Shadow           bool
}

type renderedTextCacheEntry struct {
	Image *ebiten.Image
}

type transformInfo struct {
	scale float64
	x     float64
	y     float64
}

var renderedTextCache = map[renderedTextCacheKey]*renderedTextCacheEntry{}
var renderedTextCacheOrder []renderedTextCacheKey

func toRGBA(clr color.Color) color.RGBA {
	if c, ok := clr.(color.RGBA); ok {
		return c
	}
	r, g, b, a := clr.RGBA()
	return color.RGBA{
		R: uint8(r >> 8),
		G: uint8(g >> 8),
		B: uint8(b >> 8),
		A: uint8(a >> 8),
	}
}

// ClearRenderedTextCache frees all cached text textures and resets the cache.
func ClearRenderedTextCache() {
	for k, entry := range renderedTextCache {
		if entry != nil && entry.Image != nil {
			entry.Image.Deallocate()
		}
		delete(renderedTextCache, k)
	}
	renderedTextCacheOrder = renderedTextCacheOrder[:0]
}

func drawCachedText(screen *ebiten.Image, txt string, face text.Face, clr color.Color, lineSpacing float64, geoM ebiten.GeoM, shadow bool) bool {
	transform, ok := geoMToTransform(geoM)
	if !ok {
		return false
	}

	entry, ok := getRenderedTextCacheEntry(txt, face, clr, lineSpacing, transform.scale, shadow)
	if !ok {
		return false
	}

	var opts ebiten.DrawImageOptions
	opts.GeoM.Translate(math.Round(transform.x), math.Round(transform.y))
	screen.DrawImage(entry.Image, &opts)
	return true
}

func getRenderedTextCacheEntry(txt string, face text.Face, clr color.Color, lineSpacing, scale float64, shadow bool) (*renderedTextCacheEntry, bool) {
	goFace, ok := face.(*text.GoTextFace)
	if !ok {
		return nil, false
	}
	if scale <= 0 {
		return nil, false
	}

	scaledFace := text.GoTextFace{
		Source:    goFace.Source,
		Direction: goFace.Direction,
		Language:  goFace.Language,
		Size:      goFace.Size * scale,
	}
	scaledLineSpacing := lineSpacing * scale
	rgba := toRGBA(clr)
	key := renderedTextCacheKey{
		Text:             txt,
		Source:           goFace.Source,
		SizeMillis:       int(math.Round(scaledFace.Size * 1000)),
		LineSpacingMilli: int(math.Round(scaledLineSpacing * 1000)),
		ColorRGBA:        rgba,
		ScaleMillis:      int(math.Round(scale * 1000)),
		Shadow:           shadow,
	}
	if entry, ok := renderedTextCache[key]; ok {
		return entry, true
	}

	textW, textH := text.Measure(txt, &scaledFace, scaledLineSpacing)
	shadowX := 0
	shadowY := 0
	if shadow {
		shadowX = max(1, int(math.Round(scale)))
		shadowY = max(1, int(math.Round(2*scale)))
	}

	imgW := max(1, int(math.Ceil(textW))+shadowX)
	imgH := max(1, int(math.Ceil(textH))+shadowY)
	img := ebiten.NewImage(imgW, imgH)

	alpha := float32(rgba.A) / 255
	if shadow {
		var shadowOpts text.DrawOptions
		shadowOpts.GeoM.Translate(float64(shadowX), float64(shadowY))
		shadowOpts.ColorScale.Scale(0, 0, 0, alpha)
		shadowOpts.LineSpacing = scaledLineSpacing
		text.Draw(img, txt, &scaledFace, &shadowOpts)
	}

	var textOpts text.DrawOptions
	textOpts.ColorScale.Scale(
		float32(rgba.R)/255,
		float32(rgba.G)/255,
		float32(rgba.B)/255,
		alpha,
	)
	textOpts.LineSpacing = scaledLineSpacing
	text.Draw(img, txt, &scaledFace, &textOpts)

	entry := &renderedTextCacheEntry{Image: img}
	if len(renderedTextCacheOrder) >= maxRenderedTextCacheEntries {
		oldest := renderedTextCacheOrder[0]
		renderedTextCacheOrder = renderedTextCacheOrder[1:]
		if oldEntry, exists := renderedTextCache[oldest]; exists && oldEntry.Image != nil {
			oldEntry.Image.Deallocate()
		}
		delete(renderedTextCache, oldest)
	}
	renderedTextCache[key] = entry
	renderedTextCacheOrder = append(renderedTextCacheOrder, key)
	return entry, true
}

func geoMToTransform(geoM ebiten.GeoM) (transformInfo, bool) {
	const epsilon = 1e-6

	b := geoM.Element(0, 1)
	c := geoM.Element(1, 0)
	if math.Abs(b) > epsilon || math.Abs(c) > epsilon {
		return transformInfo{}, false
	}

	scaleX := geoM.Element(0, 0)
	scaleY := geoM.Element(1, 1)
	if math.Abs(scaleX-scaleY) > epsilon {
		return transformInfo{}, false
	}

	return transformInfo{
		scale: scaleX,
		x:     geoM.Element(0, 2),
		y:     geoM.Element(1, 2),
	}, true
}
