package main

import (
	"fmt"
	"math/rand"

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

		for range 10 {
			addCard("Mountain")
		}
		for range 10 {
			addCard("Lightning Bolt")
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

func runPhase(game *core_engine.GameState, player *core_engine.Player) {
	fmt.Printf("  Phase: %s\n", player.Turn.Phase)

	switch player.Turn.Phase {
	case core_engine.PhaseDraw:
		if err := player.DrawCard(); err != nil {
			fmt.Println("    Unable to draw card")
			player.HasLost = true
			return
		}
		fmt.Println("    Drew a card")
	case core_engine.PhaseMain1, core_engine.PhaseMain2:
		runMainPhase(game, player)
	}
}

func runMainPhase(game *core_engine.GameState, player *core_engine.Player) {
	for {
		allActions := game.AvailableActions(player)
		actions := []core_engine.PlayerAction{}
		for _, a := range allActions {
			if a.Type == core_engine.ActionCastSpell && a.Card.CardType == domain.CardTypeLand {
				continue
			}
			actions = append(actions, a)
		}

		if len(actions) <= 1 {
			return
		}

		actionStrs := []string{}
		for _, a := range actions {
			if a.Card != nil {
				actionStrs = append(actionStrs, fmt.Sprintf("%s %s", a.Type, cardName(a.Card)))
			} else {
				actionStrs = append(actionStrs, a.Type)
			}
		}
		fmt.Printf("    Available actions: %v\n", actionStrs)

		var action core_engine.PlayerAction
		castActions := []core_engine.PlayerAction{}
		landActions := []core_engine.PlayerAction{}
		for _, a := range actions {
			switch a.Type {
			case core_engine.ActionCastSpell:
				castActions = append(castActions, a)
			case core_engine.ActionPlayLand:
				landActions = append(landActions, a)
			}
		}

		if len(castActions) > 0 {
			action = castActions[rand.Intn(len(castActions))]
		} else if len(landActions) > 0 {
			action = landActions[rand.Intn(len(landActions))]
		} else {
			action = core_engine.PlayerAction{Type: core_engine.ActionPassPriority}
		}

		switch action.Type {
		case core_engine.ActionPassPriority:
			fmt.Println("    Passing priority")
			return
		case core_engine.ActionPlayLand:
			fmt.Printf("    Playing %s\n", cardName(action.Card))
			game.PlayLand(player, action.Card)
		case core_engine.ActionCastSpell:
			if err := tapLandsForMana(game, player, action.Card.ManaCost); err != nil {
				fmt.Printf("    Cannot tap lands for %s: %v\n", cardName(action.Card), err)
				continue
			}

			var target core_engine.Targetable
			targets := game.AvailableTargets(action.Card)
			if len(targets) > 0 {
				target = targets[rand.Intn(len(targets))]
			}

			if target != nil {
				fmt.Printf("    Casting %s targeting %s\n", cardName(action.Card), targetName(target))
			} else {
				fmt.Printf("    Casting %s\n", cardName(action.Card))
			}

			if err := game.CastSpell(player, action.Card, target); err != nil {
				fmt.Printf("    Failed to cast: %v\n", err)
				continue
			}
			if item := game.Stack.Pop(); item != nil {
				game.Resolve(item)
				if target != nil {
					fmt.Printf("    %s now at %d life\n", targetName(target), getLife(target))
				}
			}
			game.CheckWinConditions()
		}
	}
}

func tapLandsForMana(game *core_engine.GameState, player *core_engine.Player, cost string) error {
	required := player.ManaPool.ParseCost(cost)
	redNeeded := required['R']

	tapped := 0
	for _, card := range player.Battlefield {
		if tapped >= redNeeded {
			break
		}
		if card.CardType == domain.CardTypeLand && !card.Tapped {
			game.TapLandForMana(player, card)
			tapped++
		}
	}

	if tapped < redNeeded {
		return fmt.Errorf("not enough untapped lands")
	}
	return nil
}

func getOpponent(game *core_engine.GameState, player *core_engine.Player) *core_engine.Player {
	for _, p := range game.Players {
		if p != player {
			return p
		}
	}
	return nil
}

func getLife(target core_engine.Targetable) int {
	if target.TargetType() == core_engine.TargetTypePlayer {
		if p, ok := target.(*core_engine.Player); ok {
			return p.LifeTotal
		}
	}
	if target.TargetType() == core_engine.TargetTypeCard {
		if c, ok := target.(*core_engine.Card); ok {
			return c.Toughness - c.DamageTaken
		}
	}
	return 0
}

func printZones(player *core_engine.Player) {
	fmt.Printf("  Hand: %s\n", cardNames(player.Hand))
	fmt.Printf("  Battlefield: %s\n", cardNames(player.Battlefield))
	fmt.Printf("  Graveyard: %s\n", cardNames(player.Graveyard))
}

func cardName(c *core_engine.Card) string {
	return fmt.Sprintf("%s#%d", c.Name(), c.ID)
}

func targetName(t core_engine.Targetable) string {
	if c, ok := t.(*core_engine.Card); ok {
		return cardName(c)
	}
	return t.Name()
}

func cardNames(cards []*core_engine.Card) string {
	if len(cards) == 0 {
		return "(empty)"
	}
	names := []string{}
	for _, c := range cards {
		names = append(names, cardName(c))
	}
	return fmt.Sprintf("%v", names)
}

func playGame(game *core_engine.GameState) {
	turnCounts := make(map[*core_engine.Player]int)
	maxTurns := 5

	for {
		player := game.Players[game.CurrentPlayer]
		turnCounts[player]++

		if turnCounts[player] > maxTurns {
			fmt.Printf("Turn limit exceeded for %s\n", player.Name())
			return
		}

		fmt.Printf("Turn %d - %s (Life: %d)\n", turnCounts[player], player.Name(), player.LifeTotal)

		player.Turn.Phase = core_engine.PhaseUntap
		player.Turn.LandPlayed = false

		phases := []core_engine.Phase{
			core_engine.PhaseUntap,
			core_engine.PhaseUpkeep,
			core_engine.PhaseDraw,
			core_engine.PhaseMain1,
			core_engine.PhaseCombat,
			core_engine.PhaseMain2,
			core_engine.PhaseEnd,
		}

		for _, card := range player.Battlefield {
			card.Tapped = false
		}

		for _, phase := range phases {
			player.Turn.Phase = phase
			runPhase(game, player)

			if player.HasLost {
				fmt.Printf("%s has lost!\n", player.Name())
				return
			}

			for _, p := range game.Players {
				if p.HasLost {
					fmt.Printf("%s has lost!\n", p.Name())
					return
				}
			}
		}

		for _, p := range game.Players {
			fmt.Printf("%s zones:\n", p.Name())
			printZones(p)
		}

		game.CurrentPlayer = (game.CurrentPlayer + 1) % len(game.Players)
	}
}

func main() {
	players := createPlayers()
	game := core_engine.NewGame(players)
	game.StartGame()

	fmt.Println("=== MTG Test Game ===")
	fmt.Printf("Players: %d, Starting life: %d\n\n", len(players), players[0].LifeTotal)

	for _, p := range players {
		fmt.Printf("%s draws 7 cards\n", p.Name())
	}
	fmt.Println()

	playGame(game)

	fmt.Println("\n=== Game Over ===")
	for _, p := range players {
		status := "alive"
		if p.HasLost {
			status = "lost"
		}
		fmt.Printf("%s: Life %d (%s)\n", p.Name(), p.LifeTotal, status)
	}
}
