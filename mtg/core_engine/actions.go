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

func (g *GameState) CastSpell(player *Player, card *Card, target Targetable) error {
    fmt.Println("Casting spell")
    if !g.CanCast(player, card) {
        return fmt.Errorf("cannot cast card")
    }

    // Pay the mana cost
    player.ManaPool.Pay(card.ManaCost)
    e := []Event{}

    for _, a := range card.Actions {
        a.AddTarget(target)
        e = append(e, a)
    }

    fmt.Println("pushing stack")
    g.Stack.Push(&StackItem{Events: e, Player: player, Card: card})

    return nil
}

// func CastFromHand(player *Player, card *Card) {
//  // Move the card from the player's hand to the battlefield
//  for i, c := range player.Hand {
//      if c == card {
//          player.Hand = slices.Delete(player.Hand, i, i+1)
//          player.Battlefield = append(player.Battlefield, card)
//          break
//      }
//  }
// }
