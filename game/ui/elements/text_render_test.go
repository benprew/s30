package elements

import (
	"image/color"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestGeoMToTransformAcceptsScaleAndTranslate(t *testing.T) {
	var geoM ebiten.GeoM
	geoM.Scale(2.5, 2.5)
	geoM.Translate(12, 34)

	transform, ok := geoMToTransform(geoM)
	if !ok {
		t.Fatal("expected scale+translate transform to be supported")
	}
	if math.Abs(transform.scale-2.5) > 1e-6 {
		t.Fatalf("scale = %v, want 2.5", transform.scale)
	}
	if math.Abs(transform.x-12) > 1e-6 || math.Abs(transform.y-34) > 1e-6 {
		t.Fatalf("origin = (%v, %v), want (12, 34)", transform.x, transform.y)
	}
}
func TestGeoMToTransformRejectsRotation(t *testing.T) {
	var geoM ebiten.GeoM
	geoM.Rotate(math.Pi / 4)

	if _, ok := geoMToTransform(geoM); ok {
		t.Fatal("expected rotated transform to be rejected")
	}
}

func TestGeoMToTransformRejectsNonUniformScale(t *testing.T) {
	var geoM ebiten.GeoM
	geoM.Scale(2, 3)

	if _, ok := geoMToTransform(geoM); ok {
		t.Fatal("expected non-uniform scale to be rejected")
	}
}

func TestToRGBA(t *testing.T) {
	c1 := color.RGBA{R: 10, G: 20, B: 30, A: 255}
	if got := toRGBA(c1); got != c1 {
		t.Fatalf("toRGBA(color.RGBA) = %v, want %v", got, c1)
	}

	c2 := color.NRGBA{R: 255, G: 128, B: 64, A: 255}
	got2 := toRGBA(c2)
	if got2.R != 255 || got2.A != 255 {
		t.Fatalf("toRGBA(color.NRGBA) = %v", got2)
	}
}

func TestClearRenderedTextCache(t *testing.T) {
	ClearRenderedTextCache()
	if len(renderedTextCache) != 0 || len(renderedTextCacheOrder) != 0 {
		t.Fatal("expected renderedTextCache to be empty")
	}
}
