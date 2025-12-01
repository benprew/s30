package dragdrop

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

type DragState int

const (
	StateIdle DragState = iota
	StateDragging
)

type DragManager struct {
	state        DragState
	dragData     DragData
	dragImage    *ebiten.Image
	dragSource   Draggable
	mouseX       int
	mouseY       int
	dragStartX   int
	dragStartY   int
	droppables   []Droppable
	currentHover Droppable
}

func NewDragManager() *DragManager {
	return &DragManager{
		state:      StateIdle,
		droppables: make([]Droppable, 0),
	}
}

func (dm *DragManager) RegisterDroppable(droppable Droppable) {
	dm.droppables = append(dm.droppables, droppable)
}

func (dm *DragManager) UnregisterDroppable(droppable Droppable) {
	for i, d := range dm.droppables {
		if d == droppable {
			dm.droppables = append(dm.droppables[:i], dm.droppables[i+1:]...)
			break
		}
	}
}

func (dm *DragManager) Update(mouseX, mouseY int, leftPressed, leftJustReleased bool, draggables []Draggable) {
	dm.mouseX = mouseX
	dm.mouseY = mouseY

	switch dm.state {
	case StateIdle:
		if leftPressed {
			for _, draggable := range draggables {
				if draggable.IsDraggable() {
					bounds := draggable.GetBounds()
					mousePoint := image.Point{mouseX, mouseY}
					if mousePoint.In(bounds) {
						dm.startDrag(draggable, mouseX, mouseY)
						break
					}
				}
			}
		}
	case StateDragging:
		if leftJustReleased {
			dm.endDrag()
		} else {
			dm.updateDragHover()
		}
	}
}

func (dm *DragManager) startDrag(draggable Draggable, x, y int) {
	dm.state = StateDragging
	dm.dragSource = draggable
	dm.dragData = draggable.StartDrag(x, y)
	dm.dragImage = draggable.GetDragImage()
	dm.dragStartX = x
	dm.dragStartY = y
	draggable.OnDragStart()
}

func (dm *DragManager) updateDragHover() {
	var newHover Droppable
	mousePoint := image.Point{dm.mouseX, dm.mouseY}

	for _, droppable := range dm.droppables {
		if mousePoint.In(droppable.GetDropBounds()) && droppable.CanAcceptDrop(dm.dragData) {
			newHover = droppable
			break
		}
	}

	if newHover != dm.currentHover {
		if dm.currentHover != nil {
			dm.currentHover.OnDragLeave()
		}
		dm.currentHover = newHover
		if dm.currentHover != nil {
			dm.currentHover.OnDragOver(dm.dragData)
		}
	}
}

func (dm *DragManager) endDrag() {
	dropped := false
	if dm.currentHover != nil {
		dropped = dm.currentHover.OnDrop(dm.dragData)
		dm.currentHover.OnDragLeave()
		dm.currentHover = nil
	}

	if dm.dragSource != nil {
		dm.dragSource.OnDragEnd(dropped)
	}

	dm.state = StateIdle
	dm.dragData = nil
	dm.dragImage = nil
	dm.dragSource = nil
}

func (dm *DragManager) IsDragging() bool {
	return dm.state == StateDragging
}

func (dm *DragManager) GetDragImage() *ebiten.Image {
	return dm.dragImage
}

func (dm *DragManager) GetDragPosition() (int, int) {
	if dm.state != StateDragging {
		return 0, 0
	}
	return dm.mouseX, dm.mouseY
}

func (dm *DragManager) Draw(screen *ebiten.Image) {
	if dm.state == StateDragging && dm.dragImage != nil {
		opts := &ebiten.DrawImageOptions{}
		bounds := dm.dragImage.Bounds()
		offsetX := float64(bounds.Dx()) / 2
		offsetY := float64(bounds.Dy()) / 2
		opts.GeoM.Translate(float64(dm.mouseX)-offsetX, float64(dm.mouseY)-offsetY)
		opts.ColorScale.Scale(1.0, 1.0, 1.0, 0.8)
		screen.DrawImage(dm.dragImage, opts)
	}
}
