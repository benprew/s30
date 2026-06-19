package dragdrop

import (
	"image"

	"github.com/benprew/s30/game/ui/elements"
	"github.com/hajimehoshi/ebiten/v2"
)

type CardDragData struct {
	ID   string
	Card any
}

func (cdd *CardDragData) GetID() string {
	return cdd.ID
}

func (cdd *CardDragData) GetData() any {
	return cdd.Card
}

type DraggableButton struct {
	*elements.Button
	isDraggable bool
	dragData    any
}

func NewDraggableButton(button *elements.Button, data any) *DraggableButton {
	return &DraggableButton{
		Button:      button,
		isDraggable: true,
		dragData:    data,
	}
}

func (db *DraggableButton) IsDraggable() bool {
	return db.isDraggable
}

func (db *DraggableButton) StartDrag(x, y int) DragData {
	return &CardDragData{
		ID:   db.ID,
		Card: db.dragData,
	}
}

func (db *DraggableButton) GetDragImage() *ebiten.Image {
	return db.Normal
}

func (db *DraggableButton) GetBounds() image.Rectangle {
	return db.Bounds
}

func (db *DraggableButton) OnDragStart() {
	db.State = elements.StatePressed
}

func (db *DraggableButton) OnDragEnd(dropped bool) {
	db.State = elements.StateNormal
}

func (db *DraggableButton) SetDraggable(draggable bool) {
	db.isDraggable = draggable
}
