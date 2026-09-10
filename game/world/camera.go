package world

import (
	"image"

	"github.com/benprew/s30/game/domain"
)

const (
	NoMaxOffset float64 = -1.0

	// Defaults
	DefaultMaxOffsetX float64 = 500.0
	DefaultMaxOffsetY float64 = 500.0
	DefaultPanSpeed   float64 = 50.0
)

type Camera struct {
	X, Y                   float64
	offsetX, offsetY       float64
	maxOffsetX, maxOffsetY float64
	panSpeed               float64
	followTarget           *domain.CharacterInstance
}

func NewCamera() *Camera {
	return &Camera{
		maxOffsetX: DefaultMaxOffsetX,
		maxOffsetY: DefaultMaxOffsetY,
		panSpeed:   DefaultPanSpeed,
	}
}

func (c *Camera) Loc() image.Point {
	target := c.followTarget
	if target != nil {
		return image.Point{
			X: target.X + int(c.offsetX),
			Y: target.Y + int(c.offsetY),
		}
	}

	return image.Point{
		X: int(c.X),
		Y: int(c.Y),
	}
}

func (c *Camera) SetPosition(x, y float64) {
	c.ResetOffset()
	c.followTarget = nil
	c.X = x
	c.Y = y
}

func (c *Camera) Pan(x, y float64) {
	c.SetPosition(c.X+x, c.Y+y)
}

func (c *Camera) Follow(target *domain.CharacterInstance) {
	c.followTarget = target
}

func (c *Camera) CancelFollow() {
	if c.followTarget == nil {
		return
	}

	c.X = float64(c.followTarget.X) + c.offsetX
	c.Y = float64(c.followTarget.Y) + c.offsetY
	c.followTarget = nil
	c.ResetOffset()
}

// UpdatePosition moves the camera independently, cancelling any follow target.
// UpdateOffset moves the camera relative to its follow target.
func (c *Camera) UpdatePosition(dirBits int, dt float64) {
	c.Pan(calculateDelta(dirBits, dt, c.panSpeed))
}

func (c *Camera) UpdateOffset(dirBits int, dt float64) {
	c.PanOffset(calculateDelta(dirBits, dt, c.panSpeed))
}

func (c *Camera) SetOffset(offsetX, offsetY float64) {
	if c.followTarget == nil || (c.maxOffsetX == 0.0 && c.maxOffsetY == 0.0) {
		return
	}

	if c.maxOffsetX != NoMaxOffset {
		offsetX = max(-c.maxOffsetX, min(offsetX, c.maxOffsetX))
	}
	if c.maxOffsetY != NoMaxOffset {
		offsetY = max(-c.maxOffsetY, min(offsetY, c.maxOffsetY))
	}

	c.offsetX = offsetX
	c.offsetY = offsetY
}

func (c *Camera) PanOffset(x, y float64) {
	c.SetOffset(c.offsetX+x, c.offsetY+y)
}

func (c *Camera) ResetOffset() {
	c.offsetX = 0.0
	c.offsetY = 0.0
}

func calculateDelta(dirBits int, dt float64, speed float64) (dx, dy float64) {
	if speed <= 0.0 {
		return 0.0, 0.0
	}

	if dirBits&domain.DirUp != 0 {
		dy = dt * speed
	} else if dirBits&domain.DirDown != 0 {
		dy = -(dt * speed)
	}
	if dirBits&domain.DirLeft != 0 {
		dx = dt * speed
	} else if dirBits&domain.DirRight != 0 {
		dx = -(dt * speed)
	}

	return dx, dy
}
