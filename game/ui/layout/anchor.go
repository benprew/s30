package layout

import (
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	frameOffsetX = 100
	frameOffsetY = 75
	frameWidth   = 820
	frameHeight  = 425
)

// Anchor represents a position on the screen relative to edges/corners/center
type Anchor int

const (
	// Corners
	TopLeft Anchor = iota
	TopRight
	BottomLeft
	BottomRight

	// Edges
	TopCenter
	BottomCenter
	LeftCenter
	RightCenter

	// Center
	Center

	// Absolute positioning (uses X, Y directly without anchoring)
	Absolute

	WFTopLeft
	WFTopRight
	WFBottomLeft
	WFBottomRight
	WFCenter
)

type Position struct {
	Anchor   Anchor
	OffsetX  int
	OffsetY  int
	PaddingX int
	PaddingY int
}

// Resolve calculates the absolute X, Y coordinates based on anchor and screen dimensions
func (p Position) Resolve(screenWidth, screenHeight int) (int, int) {
	var x, y int

	switch p.Anchor {
	case WFTopLeft:
		x, y = frameOffsetX, frameOffsetY
	case WFTopRight:
		x, y = frameOffsetX+frameWidth, frameOffsetY
	case WFBottomLeft:
		x, y = frameOffsetX, frameOffsetY+frameHeight
	case WFBottomRight:
		x, y = frameOffsetX+frameWidth, frameOffsetY+frameHeight
	case WFCenter:
		x, y = frameOffsetX+(frameWidth/2), frameOffsetY+(frameHeight/2)
	case TopLeft:
		x, y = 0, 0
	case TopRight:
		x, y = screenWidth, 0
	case BottomLeft:
		x, y = 0, screenHeight
	case BottomRight:
		x, y = screenWidth, screenHeight
	case TopCenter:
		x, y = screenWidth/2, 0
	case BottomCenter:
		x, y = screenWidth/2, screenHeight
	case LeftCenter:
		x, y = 0, screenHeight/2
	case RightCenter:
		x, y = screenWidth, screenHeight/2
	case Center:
		x, y = screenWidth/2, screenHeight/2
	case Absolute:
		return p.OffsetX, p.OffsetY
	}

	return x + p.OffsetX + p.PaddingX, y + p.OffsetY + p.PaddingY
}

// ResolveWithSize calculates position and optionally adjusts for element size
// This is useful for centering elements or aligning them relative to their size
func (p Position) ResolveWithSize(screenWidth, screenHeight, elemWidth, elemHeight int, centerElement bool) (int, int) {
	x, y := p.Resolve(screenWidth, screenHeight)

	if centerElement {
		// Adjust position so the element's center is at the anchor point
		x -= elemWidth / 2
		y -= elemHeight / 2
	}

	return x, y
}

// GetBounds is a helper to get screen dimensions from an ebiten.Image
func GetBounds(screen *ebiten.Image) (width, height int) {
	bounds := screen.Bounds()
	return bounds.Dx(), bounds.Dy()
}
