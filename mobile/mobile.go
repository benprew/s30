package mobile

import (
	"github.com/benprew/s30/game"
	"github.com/hajimehoshi/ebiten/v2/mobile"
)

func init() {
	g, err := game.NewGame()
	if err != nil {
		panic(err)
	}
	mobile.SetGame(g)
}

// Dummy is required by gomobile.
func Dummy() {}
