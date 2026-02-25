package ai

import (
	"math/rand"

	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/mtg/core"
)

func ChooseAction(actions []core.PlayerAction) core.PlayerAction {
	castActions := []core.PlayerAction{}
	landActions := []core.PlayerAction{}
	attackActions := []core.PlayerAction{}
	blockActions := []core.PlayerAction{}
	discardActions := []core.PlayerAction{}

	for _, a := range actions {
		switch a.Type {
		case core.ActionCastSpell:
			if a.Card.CardType != domain.CardTypeLand {
				castActions = append(castActions, a)
			}
		case core.ActionPlayLand:
			landActions = append(landActions, a)
		case core.ActionDeclareAttacker:
			attackActions = append(attackActions, a)
		case core.ActionDeclareBlocker:
			blockActions = append(blockActions, a)
		case core.ActionDiscard:
			discardActions = append(discardActions, a)
		}
	}

	if len(discardActions) > 0 {
		return discardActions[rand.Intn(len(discardActions))]
	}
	if len(castActions) > 0 {
		return castActions[rand.Intn(len(castActions))]
	}
	if len(landActions) > 0 {
		return landActions[rand.Intn(len(landActions))]
	}
	if len(attackActions) > 0 {
		return attackActions[rand.Intn(len(attackActions))]
	}
	if len(blockActions) > 0 {
		return blockActions[rand.Intn(len(blockActions))]
	}

	return core.PlayerAction{Type: core.ActionPassPriority}
}

func RunAI(game *core.GameState, player *core.Player, done <-chan struct{}) {
	for {
		select {
		case <-done:
			return
		case <-player.WaitingChan:
		}

		if player.HasLost || game.GetOpponent(player).HasLost {
			return
		}

		actions := game.AvailableActions(player)
		action := ChooseAction(actions)

		select {
		case player.InputChan <- action:
		case <-done:
			return
		}
	}
}
