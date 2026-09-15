package world

import (
	"image"
	"math"

	"github.com/benprew/s30/game/domain"
)

type DynamicPanMode uint8

const (
	DynamicPanModeDisabled DynamicPanMode = iota
	DynamicPanModeFollowBehind
	DynamicPanModeLookAhead
)

const (
	// Sentinels
	NoMaxOffset float64 = -1.0

	// Settings
	DynamicOffsetDisplacement float64 = 25.0

	// Defaults
	DefaultMaxManualOffsetX     float64        = 200.0
	DefaultMaxManualOffsetY     float64        = 75.0
	DefaultManualPanSpeed       float64        = 250.0
	DefaultDynamicPanSpeed      float64        = 60.0
	DefaultDynamicPanSmoothTime float64        = 0.35
	DefaultEnableManualPan      bool           = false
	DefaultDynamicPanMode       DynamicPanMode = DynamicPanModeLookAhead
)

type Camera struct {
	X, Y float64

	followTarget *domain.CharacterInstance

	manualOffsetX, manualOffsetY               float64
	dynamicOffsetX, dynamicOffsetY             float64
	targetDynamicOffsetX, targetDynamicOffsetY float64
	dynamicVelocityX, dynamicVelocityY         float64

	maxManualOffsetX, maxManualOffsetY float64

	enableManualPan bool
	manualPanSpeed  float64

	dynamicPanMode       DynamicPanMode
	dynamicPanSpeed      float64
	dynamicPanSmoothTime float64
}

func NewCamera() *Camera {
	return &Camera{
		maxManualOffsetX:     DefaultMaxManualOffsetX,
		maxManualOffsetY:     DefaultMaxManualOffsetY,
		enableManualPan:      DefaultEnableManualPan,
		manualPanSpeed:       DefaultManualPanSpeed,
		dynamicPanMode:       DefaultDynamicPanMode,
		dynamicPanSpeed:      DefaultDynamicPanSpeed,
		dynamicPanSmoothTime: DefaultDynamicPanSmoothTime,
	}
}

func (c *Camera) Loc() image.Point {
	if c == nil {
		return image.Point{}
	}

	target := c.followTarget
	if target != nil {
		return image.Point{
			X: target.X + int(math.Round(c.manualOffsetX)) + int(math.Round(c.dynamicOffsetX)),
			Y: target.Y + int(math.Round(c.manualOffsetY)) + int(math.Round(c.dynamicOffsetY)),
		}
	}

	return image.Point{
		X: int(math.Round(c.X)),
		Y: int(math.Round(c.Y)),
	}
}

func (c *Camera) Update(manualDirBits, movementDirBits int, dt float64) {
	if c == nil || c.followTarget == nil {
		return
	}

	c.UpdateManualOffset(manualDirBits, dt)
	c.UpdateDynamicOffset(movementDirBits, dt)
}

func (c *Camera) SetPosition(x, y float64) {
	if c == nil {
		return
	}

	c.ResetManualOffset()
	c.ResetDynamicOffset()
	c.followTarget = nil
	c.X = x
	c.Y = y
}

func (c *Camera) Pan(x, y float64) {
	if c == nil {
		return
	}

	c.SetPosition(c.X+x, c.Y+y)
}

func (c *Camera) Follow(target *domain.CharacterInstance) {
	if c == nil {
		return
	}

	c.followTarget = target
}

func (c *Camera) CancelFollow() {
	if c == nil || c.followTarget == nil {
		return
	}

	c.X = float64(c.followTarget.X) +
		c.manualOffsetX +
		c.dynamicOffsetX
	c.Y = float64(c.followTarget.Y) +
		c.manualOffsetY +
		c.dynamicOffsetY

	c.followTarget = nil
	c.ResetManualOffset()
	c.ResetDynamicOffset()
}

// UpdatePosition moves the camera independently, canceling any follow target.
// UpdateManualOffset moves the camera relative to its follow target.
func (c *Camera) UpdatePosition(dirBits int, dt float64) {
	if c.manualPanSpeed <= 0.0 {
		return
	}

	c.Pan(calculateDelta(dirBits, dt, c.manualPanSpeed))
}

func (c *Camera) UpdateManualOffset(dirBits int, dt float64) {
	if !c.enableManualPan || c.followTarget == nil || c.manualPanSpeed <= 0.0 {
		return
	}

	c.PanManualOffset(calculateDelta(dirBits, dt, c.manualPanSpeed))
}

func (c *Camera) SetManualOffset(offsetX, offsetY float64) {
	if c.followTarget == nil ||
		(c.maxManualOffsetX == 0.0 && c.maxManualOffsetY == 0.0) {
		return
	}

	if c.maxManualOffsetX != NoMaxOffset {
		offsetX = max(-c.maxManualOffsetX, min(offsetX, c.maxManualOffsetX))
	}
	if c.maxManualOffsetY != NoMaxOffset {
		offsetY = max(-c.maxManualOffsetY, min(offsetY, c.maxManualOffsetY))
	}

	c.manualOffsetX = offsetX
	c.manualOffsetY = offsetY
}

func (c *Camera) PanManualOffset(x, y float64) {
	c.SetManualOffset(c.manualOffsetX+x, c.manualOffsetY+y)
}

func (c *Camera) ResetManualOffset() {
	if c == nil {
		return
	}
	c.manualOffsetX = 0.0
	c.manualOffsetY = 0.0
}

func (c *Camera) SetDynamicOffset(offsetX, offsetY float64) {
	if c == nil {
		return
	}
	c.dynamicOffsetX = offsetX
	c.dynamicOffsetY = offsetY
	c.dynamicVelocityX = 0.0
	c.dynamicVelocityY = 0.0
}

func (c *Camera) PanDynamicOffset(x, y float64) {
	c.SetDynamicOffset(c.dynamicOffsetX+x, c.dynamicOffsetY+y)
}

func (c *Camera) SetDynamicOffsetTarget(dirBits int) {
	c.targetDynamicOffsetX = 0.0
	c.targetDynamicOffsetY = 0.0

	if c.dynamicPanMode == DynamicPanModeDisabled {
		return
	}

	displacement := DynamicOffsetDisplacement

	if c.dynamicPanMode == DynamicPanModeFollowBehind {
		displacement = -displacement
	}

	if dirBits&domain.DirUp != 0 {
		c.targetDynamicOffsetY = -displacement
	} else if dirBits&domain.DirDown != 0 {
		c.targetDynamicOffsetY = displacement
	}

	if dirBits&domain.DirLeft != 0 {
		c.targetDynamicOffsetX = -displacement
	} else if dirBits&domain.DirRight != 0 {
		c.targetDynamicOffsetX = displacement
	}

	if c.targetDynamicOffsetX != 0.0 && c.targetDynamicOffsetY != 0.0 {
		c.targetDynamicOffsetX *= domain.DiagonalMovementScale
		c.targetDynamicOffsetY *= domain.DiagonalMovementScale
	}
}

func (c *Camera) UpdateDynamicOffset(dirBits int, dt float64) {
	if c.followTarget == nil || c.dynamicPanSpeed <= 0.0 {
		return
	}

	c.SetDynamicOffsetTarget(dirBits)

	c.dynamicOffsetX = smoothDamp(
		c.dynamicOffsetX,
		c.targetDynamicOffsetX,
		&c.dynamicVelocityX,
		c.dynamicPanSmoothTime,
		c.dynamicPanSpeed,
		dt,
	)

	c.dynamicOffsetY = smoothDamp(
		c.dynamicOffsetY,
		c.targetDynamicOffsetY,
		&c.dynamicVelocityY,
		c.dynamicPanSmoothTime,
		c.dynamicPanSpeed,
		dt,
	)
}

func (c *Camera) ResetDynamicOffset() {
	if c == nil {
		return
	}
	c.dynamicOffsetX = 0.0
	c.dynamicOffsetY = 0.0
	c.targetDynamicOffsetX = 0.0
	c.targetDynamicOffsetY = 0.0
	c.dynamicVelocityX = 0.0
	c.dynamicVelocityY = 0.0
}

func calculateDelta(dirBits int, dt float64, speed float64) (dx, dy float64) {
	if speed <= 0.0 {
		return 0.0, 0.0
	}

	if dirBits&domain.DirUp != 0 {
		dy = -(dt * speed)
	} else if dirBits&domain.DirDown != 0 {
		dy = dt * speed
	}

	if dirBits&domain.DirLeft != 0 {
		dx = -(dt * speed)
	} else if dirBits&domain.DirRight != 0 {
		dx = dt * speed
	}

	if dx != 0.0 && dy != 0.0 {
		dx *= domain.DiagonalMovementScale
		dy *= domain.DiagonalMovementScale
	}

	return dx, dy
}

func smoothDamp(current, target float64, currentVelocity *float64, smoothTime, maxSpeed, dt float64) float64 {
	smoothTime = max(0.0001, smoothTime)
	omega := 2.0 / smoothTime

	x := omega * dt
	exp := 1.0 / (1.0 + x + 0.48*x*x + 0.235*x*x*x)

	change := current - target
	origTarget := target

	if maxSpeed > 0.0 {
		maxChange := maxSpeed * smoothTime
		change = min(max(change, -maxChange), maxChange)
	}

	target = current - change
	temp := (*currentVelocity + omega*change) * dt
	*currentVelocity = (*currentVelocity - omega*temp) * exp
	output := target + (change+temp)*exp

	if (origTarget-current > 0.0) == (output > origTarget) {
		output = origTarget
		*currentVelocity = 0.0
	}

	return output
}
