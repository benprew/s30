package brawl

import (
	mage "git.sr.ht/~cdcarter/mage-go/pkg/mage"
)

type Brawl struct {
	game *mage.Game
}

func NewBrawl(playerA, playerB mage.Player) *Brawl {
	g := mage.NewGame(playerA, playerB)
	return &Brawl{game: g}
}

func (b *Brawl) Update() error {
	return nil
}
