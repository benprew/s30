package elements

import (
	"bytes"
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

func ScaleImage(img *ebiten.Image, scale float64) *ebiten.Image {
	X := img.Bounds().Dx()
	Y := img.Bounds().Dy()

	newImg := ebiten.NewImage(int(float64(X)*scale), int(float64(Y)*scale))

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Scale(scale, scale)
	newImg.DrawImage(img, opts)
	return newImg
}

func ScaleImageInd(img *ebiten.Image, scaleX, scaleY float64) *ebiten.Image {
	X := img.Bounds().Dx()
	Y := img.Bounds().Dy()

	newImg := ebiten.NewImage(int(float64(X)*scaleX), int(float64(Y)*scaleY))

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Scale(scaleX, scaleY)
	newImg.DrawImage(img, opts)
	return newImg
}

// Helper to load an image (byte array) to *ebiten.Image
// Can pre-scale image by passing scale value
func LoadImage(asset []byte, scale float64) *ebiten.Image {
	img, _, err := image.Decode(bytes.NewReader(asset))
	if err != nil {
		panic(fmt.Sprintf("Unable to load image: %s", err))
	}

	eimg := ebiten.NewImageFromImage(img)
	if scale != 1.0 {
		return ScaleImage(eimg, scale)
	}
	return eimg
}
