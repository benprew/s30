package domain

import "image"

type Castle struct {
	Name      string
	Color     ColorMask
	RogueName string
	MapTile   image.Point
	Defeated  bool
}
