package core_engine

import (
	"fmt"
	"slices"
)

type Zone int

const (
	ZoneLibrary Zone = iota
	ZoneHand
	ZoneBattlefield
	ZoneGraveyard
	ZoneExile
	ZoneStack
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
	card.CurrentZone = ZoneHand

	return nil
}

func (p *Player) ReceiveDamage(amount int) {
	p.LifeTotal -= amount
}

func (p *Player) Notify() {
	// TODO: Send notification to player
}

func (p *Player) getZone(zone Zone) *[]*Card {
	switch zone {
	case ZoneHand:
		return &p.Hand
	case ZoneLibrary:
		return &p.Library
	case ZoneBattlefield:
		return &p.Battlefield
	case ZoneGraveyard:
		return &p.Graveyard
	case ZoneExile:
		return &p.Exile
	default:
		return nil
	}
}

func (p *Player) MoveTo(card *Card, destZone Zone) error {
	if card.Owner != p {
		return fmt.Errorf("card %s is not owned by this player", card.CardName)
	}

	srcZone := p.getZone(card.CurrentZone)
	if srcZone == nil {
		return fmt.Errorf("invalid source zone %d", card.CurrentZone)
	}

	destSlice := p.getZone(destZone)
	if destSlice == nil {
		return fmt.Errorf("invalid destination zone %d", destZone)
	}

	found := false
	for i, c := range *srcZone {
		if c == card {
			*srcZone = slices.Delete(*srcZone, i, i+1)
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("card %s not found in zone %d", card.CardName, card.CurrentZone)
	}

	*destSlice = append(*destSlice, card)
	card.CurrentZone = destZone
	return nil
}
