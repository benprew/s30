package main

import (
	"fmt"
	"math/rand"

	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/mtg/ai"
	"github.com/benprew/s30/mtg/core"
)

func createPlayers() []*core.Player {
	players := []*core.Player{}
	entityID := core.EntityID(1)

	for i := range 2 {
		player := &core.Player{
			ID:          core.EntityID(i + 1),
			LifeTotal:   12,
			ManaPool:    core.ManaPool{},
			Hand:        []*core.Card{},
			Library:     []*core.Card{},
			Battlefield: []*core.Card{},
			Graveyard:   []*core.Card{},
			Exile:       []*core.Card{},
			Turn:        &core.Turn{},
			InputChan:   make(chan core.PlayerAction, 100),
			WaitingChan: make(chan struct{}, 1),
			IsAI:        true,
		}

		addCard := func(name string) {
			domainCard := domain.FindCardByName(name)
			if domainCard != nil {
				coreCard := core.NewCardFromDomain(domainCard, entityID, player)
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
		for range 2 {
			addCard("Lightning Bolt")
		}
		for range 2 {
			addCard("Giant Growth")
		}
		for range 2 {
			addCard("Kird Ape")
		}
		for range 2 {
			addCard("Sol Ring")
		}
		for range 2 {
			addCard("War Mammoth")
		}
		for range 2 {
			addCard("Dragon Whelp")
		}
		for range 2 {
			addCard("Scryb Sprites")
		}

		rand.Shuffle(len(player.Library), func(i, j int) {
			player.Library[i], player.Library[j] = player.Library[j], player.Library[i]
		})

		players = append(players, player)
	}

	return players
}

func runAI(game *core.GameState, player *core.Player, done chan struct{}) {
	for {
		select {
		case <-done:
			return
		case <-player.WaitingChan:
		}

		if player.HasLost || game.GetOpponent(player).HasLost {
			return
		}

		activePlayer := game.Players[game.ActivePlayer]

		fmt.Printf("  [AI %s] Getting actions, phase=%v, combat_step=%v\n",
			player.Name(), activePlayer.Turn.Phase, activePlayer.Turn.CombatStep)

		actions := game.AvailableActions(player)
		fmt.Printf("  [AI %s] Available actions: %d\n", player.Name(), len(actions))
		for i, a := range actions {
			if a.Card != nil {
				fmt.Printf("    [%d] %v - %s\n", i, a.Type, a.Card.Name())
			} else {
				fmt.Printf("    [%d] %v\n", i, a.Type)
			}
		}

		action := ai.ChooseAction(actions)
		if action.Type == core.ActionCastSpell && action.Card != nil {
			if action.Target != nil {
				fmt.Printf("  [AI %s] Chose action: %v - %s#%d -> %s#%d\n",
					player.Name(), action.Type, action.Card.Name(), action.Card.ID,
					action.Target.Name(), action.Target.EntityID())
			} else {
				fmt.Printf("  [AI %s] Chose action: %v - %s#%d\n",
					player.Name(), action.Type, action.Card.Name(), action.Card.ID)
			}
		} else {
			fmt.Printf("  [AI %s] Chose action: %v\n", player.Name(), action.Type)
		}

		select {
		case player.InputChan <- action:
			fmt.Printf("  [AI %s] Sent action to channel\n", player.Name())
		case <-done:
			return
		}
	}
}

func main() {
	players := createPlayers()
	game := core.NewGame(players)
	game.StartGame()

	fmt.Println("=== MTG Test Game ===")
	fmt.Printf("Players: %d, Starting life: %d\n\n", len(players), players[0].LifeTotal)

	done := make(chan struct{})
	for _, p := range players {
		go runAI(game, p, done)
	}

	winners := core.PlayGame(game, 15)
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
