package sprites

import "github.com/hajimehoshi/ebiten/v2"

type Land struct {
	Marsh     *ebiten.Image
	Water     *ebiten.Image
	Forest    *ebiten.Image
	Mountains *ebiten.Image
	Plains    *ebiten.Image
}

var Height = 11
var Width = 5
