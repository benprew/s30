package core_engine

import (
	"fmt"
	"slices"
	"time"
)

type GameState struct {
	Players           []*Player
	CurrentPlayer     int
	ActivePlayer      int
	CurrentState      string
	ConsecutivePasses int
	Stack             Stack
}

func NewGame(players []*Player) *GameState {
	return &GameState{
		Players:           players,
		CurrentPlayer:     0,
		ActivePlayer:      0,
		CurrentState:      "WaitingForAction",
		ConsecutivePasses: 0,
		Stack:             NewStack(),
	}
}

func (g *GameState) CheckWinConditions() {
	for _, player := range g.Players {
		if player.LifeTotal <= 0 {
			player.HasLost = true
		}
	}
}

func (g *GameState) StartGame() {
	for i, player := range g.Players {
		for range 7 {
			player.DrawCard()
		}

		fmt.Println(i)
		fmt.Println(player.Hand)
		fmt.Println(player.Library)
	}
}

// When game receives player input, it should be the total input needed for the spell. Ie. for bolt
// it should receive that it's casting LB and have already chosen a target. No decisions should be
// remaining to be made. The UI should handle all the decisions before it's passed off to the game
func (g *GameState) WaitForPlayerInput(player *Player) (action PlayerAction) {
	fmt.Printf("Waiting for player %d input on %s\n", player.ID, player.Turn.Phase)

	AITurnTimeout := 5 * time.Second

	// Wait for input from the specific player who has priority
	if player.IsAI {
		select {
		case action = <-player.InputChan:
		case <-time.After(AITurnTimeout):
		}
	} else {
		action = <-player.InputChan
	}

	return action
}

func (g *GameState) Resolve(item *StackItem) error {
	if item == nil {
		return nil
	}
	p := item.Player
	c := item.Card
	if c.CardType != CardTypeInstant && c.CardType != CardTypeSorcery {
		p.RemoveFrom(c, p.Hand, "Hand")
		p.AddTo(c, "Battlefield")
	} else {
		p.RemoveFrom(c, p.Hand, "Hand")
		p.AddTo(c, "Graveyard")
	}
	return nil
}

func (g *GameState) ProcessAction(player *Player, action PlayerAction) (StackResult, *StackItem) {
	switch action.Type {
	case "CastSpell":
		// TODO: spells will have decisions that need to be made when cast
		// the player will need to make those decisions (ie Lightning Bolt target)
		g.Stack.Next(EventPlayerAddsAction, &StackItem{Player: player, Events: action.Card.Actions})
		g.CastCreature(player, action.Card)
	case "PlayLand":
		g.PlayLand(player, action.Card)
	case "PassPriority":
		return g.Stack.Next(EventPlayerPassesPriority, nil)
	default:
		fmt.Println("Unknown action type:", action.Type)
	}
	return -1, nil
}

func (g *GameState) RunStack() {
	player := g.Players[g.CurrentPlayer]
	player2 := g.Players[g.CurrentPlayer+1%2]
	cnt := 0
	result, item := g.Stack.Next(-1, nil)
	for g.Stack.CurrentState != StateEmpty {
		if cnt > 500 {
			break
		}
		fmt.Println("Stack State: ", g.Stack.CurrentState)
		switch result {
		case ActPlayerPriority:
			fmt.Println("Waiting for player1")
			action := g.WaitForPlayerInput(player)
			fmt.Println("Got Action", action)
			result, _ = g.ProcessAction(player, action)
		case NonActPlayerPriority:
			fmt.Println("Waiting for player2")
			action := g.WaitForPlayerInput(player2)
			result, _ = g.ProcessAction(player2, action)
		case Resolve:
			// TODO: resolve cards
			g.Resolve(item)
			result, item = g.Stack.Next(-1, nil)
		default:
			result, item = g.Stack.Next(-1, nil)
		}
		cnt++
	}

}

func (g *GameState) NextTurn() {
	player := g.Players[g.CurrentPlayer]
	for player.Turn.Phase != PhaseEnd {
		fmt.Println("Phase:", player.Turn.Phase)
		cards, err := g.CardsWithActions(player)
		if err != nil {
			fmt.Println("Error getting cards with actions:", err)
			return
		}
		switch player.Turn.Phase {
		case PhaseUntap:
			g.UntapPhase(player, cards)
		case PhaseUpkeep:
			g.UpkeepPhase(player, cards)
			g.RunStack()
		case PhaseDraw:
			g.DrawPhase(player, cards)
			g.RunStack()
		case PhaseMain1:
			g.MainPhase(player, cards)
			g.RunStack()
		case PhaseCombat:
			g.RunStack()
		case PhaseMain2:
			g.RunStack()
		case PhaseEnd:
			g.RunStack()
		}
		g.CheckWinConditions()

		if player.HasLost {
			return
		}
		player.Turn.NextPhase()
	}

	g.CurrentPlayer = (g.CurrentPlayer + 1) % len(g.Players)
}

func (g *GameState) UntapPhase(player *Player, cards []*Card) {
	for _, card := range player.Battlefield {
		card.Tapped = false
	}
}

func (g *GameState) UpkeepPhase(player *Player, cards []*Card) {
}

func (g *GameState) DrawPhase(player *Player, cards []*Card) {
	if err := player.DrawCard(); err != nil {
		player.HasLost = true
	}
}

func (g *GameState) MainPhase(player *Player, cards []*Card) {

}

func (g *GameState) CardsWithActions(player *Player) ([]*Card, error) {
	cards := []*Card{}
	for _, card := range player.Hand {
		if g.CanCast(player, card) {
			cards = append(cards, card)
		}
	}
	for _, card := range player.Battlefield {
		if g.CanTap(player, card) {
			cards = append(cards, card)
		}
	}
	return cards, nil
}

func (g *GameState) CanTap(player *Player, card *Card) bool {
	if card.CardType == CardTypeLand && !card.Tapped {
		return true
	}

	// Creatures can tap for abilities at instant speed, but attacking/blocking tap
	// happens during combat. This function seems to be for tapping for mana or abilities.
	// Assuming tapping for mana/abilities is possible if not tapped.
	// If tapping is only for attacking/blocking, the combat phase check is relevant.
	// Let's keep the combat phase check for now based on the original code's structure.
	if card.CardType == CardTypeCreature && !card.Tapped && player.Turn.Phase == PhaseCombat {
		// This condition seems specific to attacking/blocking.
		// If creatures can tap for mana/abilities outside combat, this logic needs refinement.
		return true
	}
	return false
}

func (g *GameState) CanCast(player *Player, card *Card) bool {
	// Check if the card is in the player's hand
	cardInHand := slices.Contains(player.Hand, card)
	if !cardInHand {
		return false
	}

	pPool := make(ManaPool, len(player.ManaPool))
	copy(pPool, player.ManaPool)
	pool := g.AvailableMana(player, pPool)

	return pool.CanPay(card.ManaCost) && g.hasTarget(card)
}

// If card doesn't have a targetable type, it has a target, otherwise check if
// there are valid targets
func (g *GameState) hasTarget(card *Card) bool {
	// this is always true since either player is damageable and if you're still
	// playing the game you have a target
	if card.Targetable == "DamageableTarget" {
		return true
	}
	return true
}

func (g *GameState) CanPlayLand(player *Player, card *Card) bool {
	if player.Turn.Phase != PhaseMain1 && player.Turn.Phase != PhaseMain2 {
		return false
	}
	if player.Turn.LandPlayed {
		return false
	}
	if card.CardType != CardTypeLand {
		return false
	}

	return true
}

// PlayGame simulates the game turn by turn until only one player remains who hasn't lost.
// It returns the slice of players who have not lost when the game ends (ideally one winner).
func PlayGame(g *GameState) []*Player {
	fmt.Println("Starting game simulation...")
	maxTurns := 500
	for totalTurns := range maxTurns {
		// Check win conditions at the start of the loop, in case the game
		// state is already resolved before the first turn.
		g.CheckWinConditions()

		nonLosingPlayers := []*Player{}
		for _, player := range g.Players {
			if !player.HasLost {
				nonLosingPlayers = append(nonLosingPlayers, player)
			}
		}

		// If 1 or fewer players haven't lost, the game is over.
		if len(nonLosingPlayers) <= 1 {
			fmt.Println("Game Over.")
			if len(nonLosingPlayers) == 1 {
				// Assuming Player has a Name field for printing the winner
				// If not, you might need to adjust this line.
				// fmt.Printf("Winner: %s\n", nonLosingPlayers[0].Name)
				fmt.Printf("Winner found.\n")
			} else {
				fmt.Println("Draw or no winner.")
			}
			return nonLosingPlayers
		}

		// Call the method to advance the turn for the current player.
		g.NextTurn()

		// Optional: Add a safety break for infinite loops in case win condition is never met
		// For example, after a certain number of turns:
		if totalTurns > maxTurns {
			fmt.Println("Game ended due to turn limit.")
			return nonLosingPlayers
		}
	}
	return nil
}
