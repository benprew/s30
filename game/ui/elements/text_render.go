package elements

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const maxRenderedTextCacheEntries = 256

type renderedTextCacheKey struct {
	Text             string
	SourceID         string
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

func drawCachedText(screen *ebiten.Image, txt string, face text.Face, clr color.Color, lineSpacing float64, geoM ebiten.GeoM, shadow bool) bool {
	transform, ok := geoMToTransform(geoM)
	if !ok {
		return false
	}

	entry, ok := getRenderedTextCacheEntry(txt, face, clr, lineSpacing, transform.scale, shadow)
	if !ok {
		return false
	}

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(math.Round(transform.x), math.Round(transform.y))
	screen.DrawImage(entry.Image, opts)
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

	scaledFace := &text.GoTextFace{
		Source:    goFace.Source,
		Direction: goFace.Direction,
		Language:  goFace.Language,
		Size:      goFace.Size * scale,
	}
	scaledLineSpacing := lineSpacing * scale
	key := renderedTextCacheKey{
		Text:             txt,
		SourceID:         fmt.Sprintf("%p", goFace.Source),
		SizeMillis:       int(math.Round(scaledFace.Size * 1000)),
		LineSpacingMilli: int(math.Round(scaledLineSpacing * 1000)),
		ColorRGBA:        color.RGBAModel.Convert(clr).(color.RGBA),
		ScaleMillis:      int(math.Round(scale * 1000)),
		Shadow:           shadow,
	}
	if entry, ok := renderedTextCache[key]; ok {
		return entry, true
	}

	textW, textH := text.Measure(txt, scaledFace, scaledLineSpacing)
	shadowX := 0
	shadowY := 0
	if shadow {
		shadowX = max(1, int(math.Round(scale)))
		shadowY = max(1, int(math.Round(2*scale)))
	}

	imgW := max(1, int(math.Ceil(textW))+shadowX)
	imgH := max(1, int(math.Ceil(textH))+shadowY)
	img := ebiten.NewImage(imgW, imgH)

	alpha := float32(key.ColorRGBA.A) / 255
	if shadow {
		shadowOpts := &text.DrawOptions{}
		shadowOpts.GeoM.Translate(float64(shadowX), float64(shadowY))
		shadowOpts.ColorScale.Scale(0, 0, 0, alpha)
		shadowOpts.LineSpacing = scaledLineSpacing
		text.Draw(img, txt, scaledFace, shadowOpts)
	}

	textOpts := &text.DrawOptions{}
	textOpts.ColorScale.Scale(
		float32(key.ColorRGBA.R)/255,
		float32(key.ColorRGBA.G)/255,
		float32(key.ColorRGBA.B)/255,
		alpha,
	)
	textOpts.LineSpacing = scaledLineSpacing
	text.Draw(img, txt, scaledFace, textOpts)

	entry := &renderedTextCacheEntry{Image: img}
	if len(renderedTextCacheOrder) >= maxRenderedTextCacheEntries {
		oldest := renderedTextCacheOrder[0]
		renderedTextCacheOrder = renderedTextCacheOrder[1:]
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
