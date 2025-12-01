package dragdrop

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

type DropArea struct {
	bounds      image.Rectangle
	acceptTypes []string
	isHovered   bool
	onDropFunc  func(DragData) bool
}

func NewDropArea(bounds image.Rectangle, acceptTypes []string, onDrop func(DragData) bool) *DropArea {
	return &DropArea{
		bounds:      bounds,
		acceptTypes: acceptTypes,
		onDropFunc:  onDrop,
	}
}

func (da *DropArea) CanAcceptDrop(data DragData) bool {
	if len(da.acceptTypes) == 0 {
		return true
	}

	for _, acceptType := range da.acceptTypes {
		if acceptType == data.GetID() || acceptType == "*" {
			return true
		}
	}
	return false
}

func (da *DropArea) OnDrop(data DragData) bool {
	if da.onDropFunc != nil {
		return da.onDropFunc(data)
	}
	return false
}

func (da *DropArea) GetDropBounds() image.Rectangle {
	return da.bounds
}

func (da *DropArea) OnDragOver(data DragData) {
	da.isHovered = true
}

func (da *DropArea) OnDragLeave() {
	da.isHovered = false
}

func (da *DropArea) Draw(screen *ebiten.Image) {
	if da.isHovered {
		img := ebiten.NewImage(da.bounds.Dx(), da.bounds.Dy())
		img.Fill(color.RGBA{0, 255, 0, 64})

		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(float64(da.bounds.Min.X), float64(da.bounds.Min.Y))
		screen.DrawImage(img, opts)
	}
}

func (da *DropArea) SetBounds(bounds image.Rectangle) {
	da.bounds = bounds
}
