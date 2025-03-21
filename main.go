package main

import (
	"log"

	"github.com/benprew/s30/game"
	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	ebiten.SetWindowTitle("Isometric (Ebitengine Demo)")
	ebiten.SetWindowSize(1280, 1024)
	ebiten.SetWindowResizable(true)

	g, err := game.NewGame()
	if err != nil {
		log.Fatal(err)
	}

	if err = ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
