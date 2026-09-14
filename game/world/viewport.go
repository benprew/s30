package world

import "image"

type Viewport struct {
	X      int
	Y      int
	Width  int
	Height int
}

func (v Viewport) Center() image.Point {
	return image.Point{
		X: v.X + v.Width/2,
		Y: v.Y + v.Height/2,
	}
}
