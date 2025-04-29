package rules

import "fmt"

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
		Hand:        append([]*Card{}, p.Hand...),
		Library:     append([]*Card{}, p.Library...),
		Battlefield: append([]*Card{}, p.Battlefield...),
		Graveyard:   append([]*Card{}, p.Graveyard...),
		Exile:       append([]*Card{}, p.Exile...),
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
