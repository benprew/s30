package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/mtg/core_engine"
)

func createPlayers() []*core_engine.Player {
	players := []*core_engine.Player{}
	entityID := core_engine.EntityID(1)

	for i := range 2 {
		player := &core_engine.Player{
			ID:          core_engine.EntityID(i + 1),
			LifeTotal:   9,
			ManaPool:    core_engine.ManaPool{},
			Hand:        []*core_engine.Card{},
			Library:     []*core_engine.Card{},
			Battlefield: []*core_engine.Card{},
			Graveyard:   []*core_engine.Card{},
			Exile:       []*core_engine.Card{},
			Turn:        &core_engine.Turn{},
			InputChan:   make(chan core_engine.PlayerAction, 100),
			IsAI:        true,
		}

		addCard := func(name string) {
			domainCard := domain.FindCardByName(name)
			if domainCard != nil {
				coreCard := core_engine.NewCardFromDomain(domainCard, entityID, player)
				player.Library = append(player.Library, coreCard)
				entityID++
			}
		}

		for range 5 {
			addCard("Mountain")
		}
		for range 5 {
			addCard("Forest")
		}
		for range 3 {
			addCard("Lightning Bolt")
		}
		for range 3 {
			addCard("Giant Growth")
		}
		for range 10 {
			addCard("Kird Ape")
		}

		rand.Shuffle(len(player.Library), func(i, j int) {
			player.Library[i], player.Library[j] = player.Library[j], player.Library[i]
		})

		players = append(players, player)
	}

	return players
}

func runAI(game *core_engine.GameState, player *core_engine.Player, done chan struct{}) {
	for {
		select {
		case <-done:
			return
		default:
		}

		if player.HasLost || game.GetOpponent(player).HasLost {
			return
		}

		activePlayer := game.Players[game.ActivePlayer]
		isActivePlayer := player == activePlayer
		isDefending := game.GetOpponent(player) == activePlayer &&
			activePlayer.Turn.Phase == core_engine.PhaseCombat &&
			activePlayer.Turn.CombatStep == core_engine.CombatStepDeclareBlockers

		if !isActivePlayer && !isDefending {
			time.Sleep(10 * time.Millisecond)
			continue
		}

		actions := game.AvailableActions(player)
		action := chooseAction(game, player, actions)

		select {
		case player.InputChan <- action:
		case <-done:
			return
		}

		time.Sleep(10 * time.Millisecond)
	}
}

func chooseAction(game *core_engine.GameState, player *core_engine.Player, actions []core_engine.PlayerAction) core_engine.PlayerAction {
	castActions := []core_engine.PlayerAction{}
	landActions := []core_engine.PlayerAction{}
	attackActions := []core_engine.PlayerAction{}
	blockActions := []core_engine.PlayerAction{}

	for _, a := range actions {
		switch a.Type {
		case core_engine.ActionCastSpell:
			if a.Card.CardType != domain.CardTypeLand {
				targets := game.AvailableTargets(a.Card)
				if len(targets) > 0 {
					a.Target = targets[rand.Intn(len(targets))]
				}
				castActions = append(castActions, a)
			}
		case core_engine.ActionPlayLand:
			landActions = append(landActions, a)
		case core_engine.ActionDeclareAttacker:
			attackActions = append(attackActions, a)
		case core_engine.ActionDeclareBlocker:
			blockActions = append(blockActions, a)
		}
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

	return core_engine.PlayerAction{Type: core_engine.ActionPassPriority}
}

func main() {
	players := createPlayers()
	game := core_engine.NewGame(players)
	game.StartGame()

	fmt.Println("=== MTG Test Game ===")
	fmt.Printf("Players: %d, Starting life: %d\n\n", len(players), players[0].LifeTotal)

	done := make(chan struct{})
	for _, p := range players {
		go runAI(game, p, done)
	}

	winners := core_engine.PlayGame(game, 10)
	close(done)

	fmt.Println("\n=== Game Over ===")
	for _, p := range players {
		status := "alive"
		if p.HasLost {
			status = "lost"
		}
		fmt.Printf("%s: Life %d (%s)\n", p.Name(), p.LifeTotal, status)
	}

	if len(winners) == 1 {
		fmt.Printf("Winner: %s\n", winners[0].Name())
	} else if len(winners) == 0 {
		fmt.Println("No winner (draw or max turns reached)")
	}
}
