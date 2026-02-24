package core

import (
	"fmt"
	"slices"

	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/mtg/effects"
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

const (
	ActionPlayLand        = "PlayLand"
	ActionCastSpell       = "CastSpell"
	ActionPassPriority    = "PassPriority"
	ActionDeclareAttacker = "DeclareAttacker"
	ActionDeclareBlocker  = "DeclareBlocker"
	ActionDiscard         = "Discard"
)

type Targetable = effects.Targetable
type TargetType = effects.TargetType

const (
	TargetTypeCard   = effects.TargetTypeCard
	TargetTypePlayer = effects.TargetTypePlayer
)

type PlayerAction struct {
	Type   string
	Card   *Card
	Target Targetable
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
	WaitingChan chan struct{}
	IsAI        bool
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

func (p *Player) GainLife(amount int) {
	p.LifeTotal += amount
}

func (p *Player) Name() string {
	return fmt.Sprintf("Player %d", p.ID)
}

func (p *Player) EntityID() int {
	return int(p.ID)
}

func (p *Player) TargetType() TargetType {
	return TargetTypePlayer
}

func (p *Player) IsDead() bool {
	return p.LifeTotal <= 0
}

func (p *Player) AddPowerBoost(int) {
}

func (p *Player) AddToughnessBoost(int) {
}

func (p *Player) Notify() {
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

func (p *Player) ControlsLandType(landType string) bool {
	for _, card := range p.Battlefield {
		if card.CardType == domain.CardTypeLand {
			if card.Name() == landType {
				return true
			}
			for _, subtype := range card.Subtypes {
				if subtype == landType {
					return true
				}
			}
		}
	}
	return false
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

	if destZone == ZoneBattlefield && card.CardType == domain.CardTypeCreature {
		card.Active = false
	}

	return nil
}
