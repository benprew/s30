package core

import (
	"fmt"
	"slices"

	"github.com/benprew/s30/mtg/effects"
)

type Action struct {
	ActionID int
	TargetID int
	Player   *Player
}

func (a *Action) Resolve() {
}

func (g *GameState) PlayLand(player *Player, card *Card) error {
	if err := player.MoveTo(card, ZoneBattlefield); err != nil {
		return err
	}
	player.Turn.LandPlayed = true
	card.Active = true
	return nil
}

func (g *GameState) CastSpell(player *Player, card *Card, target Targetable) error {
	if !g.CanCast(player, card) {
		return fmt.Errorf("cannot cast card")
	}

	if err := player.ManaPool.Pay(card.ManaCost); err != nil {
		return err
	}
	e := []effects.Event{}

	for _, a := range card.CardActions() {
		a.AddTarget(target)
		e = append(e, a)
	}

	g.Stack.Push(&StackItem{Events: e, Player: player, Card: card, Target: target})

	return nil
}

func (g *GameState) ActivateManaAbility(player *Player, card *Card) error {
	isOnBattlefield := slices.Contains(player.Battlefield, card)
	if !isOnBattlefield {
		return fmt.Errorf("card %s is not on the battlefield", card.CardName)
	}

	if card.Tapped {
		return fmt.Errorf("card %s is already tapped", card.CardName)
	}

	manaAbility := card.GetManaAbility()
	if manaAbility == nil {
		return fmt.Errorf("card %s has no mana ability", card.CardName)
	}

	card.Tapped = true

	for _, manaType := range manaAbility.ManaTypes {
		player.ManaPool.AddMana([]rune(manaType))
	}

	return nil
}
