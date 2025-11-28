package dragdrop

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

type DragData interface {
	GetID() string
	GetData() interface{}
}

type Draggable interface {
	IsDraggable() bool
	StartDrag(x, y int) DragData
	GetDragImage() *ebiten.Image
	GetBounds() image.Rectangle
	OnDragStart()
	OnDragEnd(dropped bool)
}

type Droppable interface {
	CanAcceptDrop(data DragData) bool
	OnDrop(data DragData) bool
	GetDropBounds() image.Rectangle
	OnDragOver(data DragData)
	OnDragLeave()
}