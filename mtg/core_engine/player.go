package core_engine

import (
	"fmt"
	"slices"
)

type PlayerAction struct {
	Type string
	Card *Card
}

type Player struct {
	ID          EntityID
	LifeTotal   int
	ManaPool    ManaPool
	Hand        []*Card
	Library     []*Card
	Battlefield []*Card
	Graveyard   []*Card
	Exile       []*Card
	Turn        *Turn
	HasLost     bool
	InputChan   chan PlayerAction
	IsAI        bool
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
		HasLost:     p.HasLost,
		InputChan:   p.InputChan,
		IsAI:        p.IsAI,
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

func (p *Player) ReceiveDamage(amount int) {
	p.LifeTotal -= amount
}

func (p *Player) Notify() {
	// TODO: Send notification to player
}

func (p *Player) RemoveFrom(c *Card, loc []*Card, locStr string) {
	for i, n := range loc {
		if n == c {
			f := loc[0:i]
			l := loc[i+1:]
			loc = f
			loc = append(loc, l...)
			break
		}
	}

	switch locStr {
	case "Hand":
		p.Hand = loc
	}
}

func (p *Player) AddTo(c *Card, loc string) {
	switch loc {
	case "Hand":
		p.Hand = append(p.Hand, c)
	case "Library":
		p.Library = append(p.Library, c)
	case "Battlefield":
		p.Battlefield = append(p.Battlefield, c)
	case "Graveyard":
		p.Graveyard = append(p.Graveyard, c)
	case "Exile":
		p.Exile = append(p.Exile, c)
	}
}

func moveCard(card *Card, source *[]*Card, dest *[]*Card) bool {
	for i, c := range *source {
		if c == card {
			// Add to destination
			*dest = append(*dest, c)
			// Remove from source
			*source = slices.Delete(*source, i, i+1)
			return true
		}
	}
	return false // card not found
}
