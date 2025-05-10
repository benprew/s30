package core_engine

import (
	"fmt"
	"slices"
)

// Action functions assume the player can perform them
// validating actions happens earlier
func (g *GameState) PlayLand(player *Player, card *Card) error {
	for i, c := range player.Hand {
		if c == card {
			player.Hand = slices.Delete(player.Hand, i, i+1)
			player.Battlefield = append(player.Battlefield, card)
			player.Turn.LandPlayed = true
			c.Active = true
			break
		}
	}
	return nil
}

func (g *GameState) CastCard(player *Player, card *Card) error {
	if !g.CanCast(player, card) {
		return fmt.Errorf("cannot cast card")
	}

	// Pay the mana cost
	player.ManaPool.Pay(card.ManaCost)

	// Move the card from the player's hand to the battlefield
	for i, c := range player.Hand {
		if c == card {
			player.Hand = slices.Delete(player.Hand, i, i+1)
			player.Battlefield = append(player.Battlefield, card)
			break
		}
	}
	return nil
}
