package ai

import (
	"math/rand"
	"sort"

	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/mtg/core"
)

func ChooseAction(actions []core.PlayerAction, game *core.GameState) core.PlayerAction {
	castActions := []core.PlayerAction{}
	landActions := []core.PlayerAction{}
	attackActions := []core.PlayerAction{}
	blockActions := []core.PlayerAction{}
	discardActions := []core.PlayerAction{}
	abilityActions := []core.PlayerAction{}

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
		case core.ActionActivateAbility:
			abilityActions = append(abilityActions, a)
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
	if len(abilityActions) > 0 {
		return abilityActions[rand.Intn(len(abilityActions))]
	}
	if len(attackActions) > 0 {
		return attackActions[rand.Intn(len(attackActions))]
	}
	if len(blockActions) > 0 {
		return chooseProfitableBlock(blockActions, game)
	}

	return core.PlayerAction{Type: core.ActionPassPriority}
}

// chooseProfitableBlock only assigns a blocker if the combined blockers
// (already assigned + available) can kill the attacker. Multiple blockers
// can gang up on a single attacker.
func chooseProfitableBlock(blockActions []core.PlayerAction, game *core.GameState) core.PlayerAction {
	type attackerInfo struct {
		attacker      *core.Card
		neededPower   int
		availablePool []core.PlayerAction
	}

	attackerMap := map[*core.Card]*attackerInfo{}
	for _, a := range blockActions {
		attacker := a.Target.(*core.Card)
		info, ok := attackerMap[attacker]
		if !ok {
			existingPower := 0
			for _, b := range game.BlockerMap[attacker] {
				existingPower += b.EffectivePower()
			}
			info = &attackerInfo{
				attacker:    attacker,
				neededPower: attacker.EffectiveToughness() - existingPower,
			}
			attackerMap[attacker] = info
		}
		info.availablePool = append(info.availablePool, a)
	}

	var candidates []attackerInfo
	for _, info := range attackerMap {
		if info.neededPower <= 0 {
			continue
		}
		totalAvailable := 0
		for _, a := range info.availablePool {
			totalAvailable += a.Card.EffectivePower()
		}
		if totalAvailable >= info.neededPower {
			candidates = append(candidates, *info)
		}
	}

	if len(candidates) == 0 {
		return core.PlayerAction{Type: core.ActionPassPriority}
	}

	sort.Slice(candidates, func(i, j int) bool {
		return len(candidates[i].availablePool) < len(candidates[j].availablePool)
	})

	best := candidates[0]
	sort.Slice(best.availablePool, func(i, j int) bool {
		return best.availablePool[i].Card.EffectivePower() > best.availablePool[j].Card.EffectivePower()
	})
	return best.availablePool[0]
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
		action := ChooseAction(actions, game)

		select {
		case player.InputChan <- action:
		case <-done:
			return
		}
	}
}
