package ui

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	dragDistance   = 8
	longPressTicks = 30
)

// Drag describes the movement of a click-and-drag gesture.
type Drag struct {
	Start    image.Point
	Position image.Point
	Delta    image.Point
}

type pointerSample struct {
	position image.Point
	down     bool
}

// pointerManager provides high-level click and drag gestures for mouse and
// touch input.
type pointerManager struct {
	position    image.Point
	previous    image.Point
	dragStart   image.Point
	down        bool
	dragging    bool
	clicked     bool
	dragStarted bool
	dragEnded   bool
	longPressed bool
	holdTicks   int
	consumed    bool
	activeTouch ebiten.TouchID
	hasTouch    bool
}

var pointer = newPointer()

func newPointer() *pointerManager {
	return &pointerManager{}
}

// UpdatePointer samples the primary pointer. It must be called once at the
// start of each game tick before querying gestures.
func UpdatePointer() {
	pointer.update()
}

// Position returns the pointer's current position.
func Position() image.Point {
	return pointer.position
}

// Pressed reports whether the primary pointer is currently held down.
func Pressed() bool {
	return pointer.Pressed()
}

// LongPress reports whether a stationary press in bounds reached the hold
// threshold this tick.
func LongPress(bounds image.Rectangle) bool {
	return pointer.LongPress(bounds)
}

// Click reports whether a click started and ended within bounds this tick.
func Click(bounds image.Rectangle) bool {
	return pointer.click(bounds)
}

// DragStart returns the drag that crossed the movement threshold this tick.
func DragStart() (Drag, bool) {
	return pointer.drag(), pointer.dragStarted
}

// Dragging returns the active drag, if any.
func Dragging() (Drag, bool) {
	return pointer.drag(), pointer.dragging
}

// DragEnd returns the drag completed during this tick.
func DragEnd() (Drag, bool) {
	return pointer.drag(), pointer.dragEnded
}

func (p *pointerManager) update() {
	p.advance(p.sample())
}

func (p *pointerManager) Click(bounds image.Rectangle) bool {
	return p.click(bounds)
}

func (p *pointerManager) Pressed() bool {
	return p.down
}

func (p *pointerManager) LongPress(bounds image.Rectangle) bool {
	return p.longPressed && p.dragStart.In(bounds) && p.position.In(bounds)
}

// DragStart returns the drag that crossed the movement threshold this tick.
func (p *pointerManager) DragStart() (Drag, bool) {
	return p.drag(), p.dragStarted
}

// Dragging returns the active drag, if any.
func (p *pointerManager) Dragging() (Drag, bool) {
	return p.drag(), p.dragging
}

// DragEnd returns the drag completed during this tick.
func (p *pointerManager) DragEnd() (Drag, bool) {
	return p.drag(), p.dragEnded
}

func (p *pointerManager) advance(sample pointerSample) {
	p.clicked = false
	p.dragStarted = false
	p.dragEnded = false
	p.longPressed = false
	p.previous = p.position
	p.position = sample.position

	pressed := sample.down && !p.down
	released := !sample.down && p.down
	if pressed {
		p.previous = sample.position
		p.dragStart = sample.position
		p.holdTicks = 1
		p.consumed = false
	}
	if sample.down && !pressed {
		p.holdTicks++
	}

	if sample.down && !p.dragging && distanceSquared(sample.position, p.dragStart) >= dragDistance*dragDistance {
		p.dragging = true
		p.dragStarted = true
	}
	if sample.down && !p.dragging && !p.consumed && p.holdTicks >= longPressTicks {
		p.longPressed = true
		p.consumed = true
	}

	if released {
		if p.dragging {
			p.dragEnded = true
			p.dragging = false
		} else if !p.consumed {
			p.clicked = true
		}
		p.holdTicks = 0
	}
	p.down = sample.down
}

func (p *pointerManager) drag() Drag {
	return Drag{
		Start:    p.dragStart,
		Position: p.position,
		Delta:    p.position.Sub(p.previous),
	}
}

func (p *pointerManager) click(bounds image.Rectangle) bool {
	return p.clicked && p.dragStart.In(bounds) && p.position.In(bounds)
}

func (p *pointerManager) sample() pointerSample {
	if p.hasTouch {
		for _, id := range ebiten.AppendTouchIDs(nil) {
			if id == p.activeTouch {
				x, y := ebiten.TouchPosition(id)
				return pointerSample{position: image.Pt(x, y), down: true}
			}
		}
		for _, id := range inpututil.AppendJustReleasedTouchIDs(nil) {
			if id == p.activeTouch {
				x, y := inpututil.TouchPositionInPreviousTick(id)
				p.hasTouch = false
				return pointerSample{position: image.Pt(x, y)}
			}
		}
		p.hasTouch = false
		return pointerSample{position: p.position}
	}

	if touches := ebiten.AppendTouchIDs(nil); len(touches) > 0 {
		p.activeTouch = touches[0]
		p.hasTouch = true
		x, y := ebiten.TouchPosition(p.activeTouch)
		return pointerSample{position: image.Pt(x, y), down: true}
	}

	x, y := ebiten.CursorPosition()
	return pointerSample{
		position: image.Pt(x, y),
		down:     ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft),
	}
}

func distanceSquared(a, b image.Point) int {
	delta := a.Sub(b)
	return delta.X*delta.X + delta.Y*delta.Y
}
