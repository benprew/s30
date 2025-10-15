package elements

import "github.com/hajimehoshi/ebiten/v2"

func ScaleImage(img *ebiten.Image, scale float64) *ebiten.Image {
	X := img.Bounds().Dx()
	Y := img.Bounds().Dy()

	newImg := ebiten.NewImage(int(float64(X)*scale), int(float64(Y)*scale))

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Scale(scale, scale)
	newImg.DrawImage(img, opts)
	return newImg
}
