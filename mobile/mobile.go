package mobile

import (
	"github.com/benprew/s30/game"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/mobile"
)

func init() {
	ebiten.SetTPS(10)
	mobile.SetGame(game.NewLoadingGame())
}

// Dummy is required by gomobile.
func Dummy() {}
