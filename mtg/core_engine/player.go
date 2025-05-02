package core_engine

import (
	"fmt"
	"slices"
)

type Player struct {
	LifeTotal   int
	ManaPool    ManaPool
	Hand        []*Card
	Library     []*Card
	Battlefield []*Card
	Graveyard   []*Card
	Exile       []*Card
	Turn        *Turn
}

func (p *Player) DeepCopy() *Player {
	newPlayer := &Player{
		LifeTotal:   p.LifeTotal,
		Hand:        slices.Clone(p.Hand),
		Library:     slices.Clone(p.Library),
		Battlefield: slices.Clone(p.Battlefield),
		Graveyard:   slices.Clone(p.Graveyard),
		Exile:       slices.Clone(p.Exile),
		Turn:        p.Turn, // Assuming Turn doesn't need to be deep copied
	}

	newManaPool := make(ManaPool, len(p.ManaPool))
	copy(newManaPool, p.ManaPool)
	newPlayer.ManaPool = newManaPool

	return newPlayer
}

func (p *Player) DrawCard() error {
	if len(p.Library) == 0 {
		return fmt.Errorf("no cards in library")
	}

	card := p.Library[0]
	p.Library = p.Library[1:]
	p.Hand = append(p.Hand, card)

	return nil
}
