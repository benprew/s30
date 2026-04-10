package imageutil

import (
	"bytes"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

//
// # Image Utility Functions
//
// Various image utility functions to load and modify images.
//

func ScaleImage(img *ebiten.Image, scale float64) *ebiten.Image {
	return ScaleImageInd(img, scale, scale)
}

// Scale image X and Y independently. Used to scale text boxes to fit text (among
// other things).
func ScaleImageInd(img *ebiten.Image, scaleX, scaleY float64) *ebiten.Image {
	X := img.Bounds().Dx()
	Y := img.Bounds().Dy()

	newImg := ebiten.NewImage(int(float64(X)*scaleX), int(float64(Y)*scaleY))

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Scale(scaleX, scaleY)
	newImg.DrawImage(img, opts)
	return newImg
}

// Make loading images from the embedded file system easier.
func LoadImage(asset []byte) (*ebiten.Image, error) {
	key := cacheKey(asset, "image")
	if cached, ok := registryGet(key); ok {
		return cached.(*ebiten.Image), nil
	}

	img, _, err := image.Decode(bytes.NewReader(asset))
	if err != nil {
		return nil, err
	}

	result := ebiten.NewImageFromImage(img)
	registrySet(key, result)
	return result, nil
}

// TileImage creates a new image of size (width, height) by tiling the src image.
func TileImage(src *ebiten.Image, width, height int) *ebiten.Image {
	newImg := ebiten.NewImage(width, height)
	srcW, srcH := src.Bounds().Dx(), src.Bounds().Dy()

	op := &ebiten.DrawImageOptions{}
	for y := 0; y < height; y += srcH {
		for x := 0; x < width; x += srcW {
			op.GeoM.Reset()
			op.GeoM.Translate(float64(x), float64(y))
			newImg.DrawImage(src, op)
		}
	}
	return newImg
}
