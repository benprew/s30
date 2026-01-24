package core_engine

import (
	"fmt"
	"slices"
)

// An event is either
type Action struct {
	ActionID int     // One of the available actions (deal damage, draw card)
	TargetID int     // either cardid or playerid, can be null (-1)
	Player   *Player // owner of the action
}

func (a *Action) Resolve() {

}

// Action functions assume the player can perform them
// validating actions happens earlier
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

	// Pay the mana cost
	if err := player.ManaPool.Pay(card.ManaCost); err != nil {
		return err
	}
	e := []Event{}

	for _, a := range card.CardActions() {
		a.AddTarget(target)
		e = append(e, a)
	}

	g.Stack.Push(&StackItem{Events: e, Player: player, Card: card})

	return nil
}

// TapLandForMana taps a land card on the battlefield and adds its mana production to the player's mana pool.
func (g *GameState) TapLandForMana(player *Player, card *Card) error {
	// Check if the card is on the player's battlefield
	isOnBattlefield := slices.Contains(player.Battlefield, card)

	if !isOnBattlefield {
		return fmt.Errorf("card %s is not on the battlefield", card.CardName)
	}

	// Check if the card is already tapped
	if card.Tapped {
		return fmt.Errorf("card %s is already tapped", card.CardName)
	}

	// Tap the card
	card.Tapped = true

	// Add mana production to the player's mana pool
	for _, manaType := range card.ManaProduction {
		player.ManaPool.AddMana([]rune(manaType))
	}

	return nil
}
