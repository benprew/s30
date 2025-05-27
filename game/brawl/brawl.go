package brawl

import (
	"github.com/benprew/s30/mtg/core_engine"
)

type Brawl struct {
	game *core_engine.GameState
}

func NewBrawl(players []*core_engine.Player) *Brawl {
	g := core_engine.NewGame(players)
	return &Brawl{game: g}
}

func (b *Brawl) Update() error {
	// loop until game ends

	return nil
}
