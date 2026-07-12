package dragdrop

import (
	"image"
	"testing"

	"github.com/benprew/s30/game/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

type testDragData struct{}

func (testDragData) GetID() string { return "test" }
func (testDragData) GetData() any  { return nil }

type testDraggable struct {
	bounds  image.Rectangle
	started bool
	ended   bool
	dropped bool
}

func (d *testDraggable) IsDraggable() bool           { return true }
func (d *testDraggable) StartDrag(_, _ int) DragData { return testDragData{} }
func (d *testDraggable) GetDragImage() *ebiten.Image { return nil }
func (d *testDraggable) GetBounds() image.Rectangle  { return d.bounds }
func (d *testDraggable) OnDragStart()                { d.started = true }
func (d *testDraggable) OnDragEnd(dropped bool)      { d.ended, d.dropped = true, dropped }

type testDroppable struct {
	bounds  image.Rectangle
	dropped bool
}

func (d *testDroppable) CanAcceptDrop(DragData) bool    { return true }
func (d *testDroppable) OnDrop(DragData) bool           { d.dropped = true; return true }
func (d *testDroppable) GetDropBounds() image.Rectangle { return d.bounds }
func (d *testDroppable) OnDragOver(DragData)            {}
func (d *testDroppable) OnDragLeave()                   {}

func TestDragManagerUsesGestureStartAndEnd(t *testing.T) {
	manager := NewDragManager()
	source := &testDraggable{bounds: image.Rect(10, 10, 30, 30)}
	target := &testDroppable{bounds: image.Rect(50, 50, 80, 80)}
	manager.RegisterDroppable(target)

	manager.Start(ui.Drag{Start: image.Pt(20, 20), Position: image.Pt(30, 30)}, []Draggable{source})
	if !source.started || !manager.IsDragging() {
		t.Fatal("drag did not start from the gesture's start position")
	}

	manager.End(ui.Drag{Start: image.Pt(20, 20), Position: image.Pt(60, 60)})
	if !target.dropped || !source.ended || !source.dropped || manager.IsDragging() {
		t.Fatal("drag did not complete at the gesture's end position")
	}
}
