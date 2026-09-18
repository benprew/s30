package dragdrop

import "image"

type DropArea struct {
	bounds        image.Rectangle
	acceptTypes   []string
	onDropFunc    func(DragData) bool
	canAcceptFunc func(DragData) bool
}

func NewDropArea(bounds image.Rectangle, acceptTypes []string, onDrop func(DragData) bool) *DropArea {
	return &DropArea{
		bounds:      bounds,
		acceptTypes: acceptTypes,
		onDropFunc:  onDrop,
	}
}

// SetCanAccept sets an optional custom predicate to validate accepted drag data.
func (da *DropArea) SetCanAccept(fn func(DragData) bool) {
	da.canAcceptFunc = fn
}

func (da *DropArea) CanAcceptDrop(data DragData) bool {
	if da.canAcceptFunc != nil && !da.canAcceptFunc(data) {
		return false
	}

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

// OnDragOver does nothing on purpose. A drop target used to tint itself while a
// card passed over it, which on a panel this size read as a flash rather than a
// cue; the card following the cursor already says where it will land. The method
// stays because Droppable asks for it.
func (da *DropArea) OnDragOver(data DragData) {
}

// OnDragLeave does nothing, paired with OnDragOver.
func (da *DropArea) OnDragLeave() {
}

func (da *DropArea) SetBounds(bounds image.Rectangle) {
	da.bounds = bounds
}
