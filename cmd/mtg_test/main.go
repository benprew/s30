package main

import (
	"fmt"

	_ "git.sr.ht/~cdcarter/mage-go/cards"
	mage "git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive/ai"
	"github.com/benprew/s30/logging"
)

func createPlayer(name string, cards []string) *ai.AIPlayer {
	player := ai.NewAIPlayer(name)
	for _, cardName := range cards {
		c, err := mage.CreateCard(cardName)
		if err != nil {
			fmt.Printf("Failed to create card %s: %v\n", cardName, err)
			continue
		}
		player.AddToLibrary(c)
	}
	player.ShuffleLibrary()
	return player
}

func main() {
	logging.Enable(logging.MTG)
	logging.Enable(logging.Duel)

	var deck []string
	for range 5 {
		deck = append(deck, "Mountain")
	}
	for range 5 {
		deck = append(deck, "Forest")
	}
	for range 2 {
		deck = append(deck, "Lightning Bolt")
	}
	for range 2 {
		deck = append(deck, "Giant Growth")
	}
	for range 2 {
		deck = append(deck, "Kird Ape")
	}
	for range 2 {
		deck = append(deck, "Sol Ring")
	}
	for range 2 {
		deck = append(deck, "War Mammoth")
	}
	for range 2 {
		deck = append(deck, "Web")
	}
	for range 2 {
		deck = append(deck, "Scryb Sprites")
	}

	ai1 := createPlayer("Player 1", deck)
	ai2 := createPlayer("Player 2", deck)

	g := mage.NewGame(ai1, ai2)

	for range 7 {
		ai1.DrawCard()
	}
	for range 7 {
		ai2.DrawCard()
	}

	fmt.Println("=== MTG Test Game ===")
	fmt.Printf("Players: 2, Starting life: %d\n\n", ai1.Life())

	g.OnPriority = func(g *mage.Game, playerIdx int, mainPhase bool) mage.PriorityAction {
		aiP := g.Players[playerIdx].(*ai.AIPlayer)
		action := aiP.GetPriorityAction(g, g.LandsPlayedThisTurn, mainPhase)
		return convertAction(action)
	}

	g.AfterPriorityAction = func(g *mage.Game, playerIdx int, action mage.PriorityAction) {
		p := g.Players[playerIdx]
		switch action.Type {
		case mage.PriorityPlayLand:
			perm := g.FindPermanent(action.CardID)
			name := "a land"
			if perm != nil {
				name = perm.Name()
			}
			fmt.Printf("  %s plays %s\n", p.Name(), name)
		case mage.PriorityCastSpell:
			obj := g.Stack.Peek()
			name := "a spell"
			if obj != nil && obj.Card != nil {
				name = obj.Card.Name()
			}
			fmt.Printf("  %s casts %s\n", p.Name(), name)
		case mage.PriorityActivateAbility:
			perm := g.FindPermanent(action.PermanentID)
			name := "permanent"
			if perm != nil {
				name = perm.Name()
			}
			fmt.Printf("  %s activates %s\n", p.Name(), name)
		}
	}

	for g.Turn <= 50 {
		fmt.Printf("── Turn %d: %s ──\n", g.Turn, g.ActivePlayerObj().Name())
		for _, step := range core.AllSteps() {
			g.RunStepWithPriority(step)
			if step == core.CombatDamage && len(g.Combat.Groups) > 0 {
				fmt.Printf("  Combat: %d attacker(s)\n", len(g.Combat.Groups))
			}
			if g.IsGameOver() {
				break
			}
		}
		if g.IsGameOver() {
			break
		}
		if len(g.ExtraTurns) > 0 {
			extraPlayerID := g.ExtraTurns[0]
			g.ExtraTurns = g.ExtraTurns[1:]
			for i, p := range g.Players {
				if p.PlayerID() == extraPlayerID {
					g.ActivePlayer = i
					break
				}
			}
		} else {
			g.ActivePlayer = (g.ActivePlayer + 1) % len(g.Players)
		}
		g.Turn++
	}

	fmt.Println("\n=== Game Over ===")
	for _, p := range g.Players {
		status := "alive"
		if !p.IsAlive() {
			status = "lost"
		}
		fmt.Printf("%s: Life %d (%s)\n", p.Name(), p.Life(), status)
	}
	fmt.Printf("Winner: %s\n", g.Winner())
}

func convertAction(a interactive.PriorityAction) mage.PriorityAction {
	switch a.Type {
	case interactive.ActionPlayLand:
		return mage.PriorityAction{Type: mage.PriorityPlayLand, CardID: a.CardID}
	case interactive.ActionCastSpell:
		return mage.PriorityAction{Type: mage.PriorityCastSpell, CardID: a.CardID, Targets: a.Targets, XValue: a.XValue}
	case interactive.ActionActivateAbility:
		return mage.PriorityAction{Type: mage.PriorityActivateAbility, PermanentID: a.PermanentID, AbilityIdx: a.AbilityIndex, Targets: a.Targets}
	default:
		return mage.PriorityAction{Type: mage.PriorityPass}
	}
}
