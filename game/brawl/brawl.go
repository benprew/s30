package brawl

import (
	"github.com/benprew/s30/mtg/core"
)

type Brawl struct {
	game *core.GameState
}

func NewBrawl(players []*core.Player) *Brawl {
	g := core.NewGame(players, false)
	return &Brawl{game: g}
}

func (b *Brawl) Update() error {
	// loop until game ends

	return nil
}
