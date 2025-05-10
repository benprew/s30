package core_engine

import (
	"fmt"
	"slices"
	"time"
)

type GameState struct {
	Players       []*Player
	CurrentPlayer int
}

func NewGame(players []*Player) *GameState {
	return &GameState{
		Players:       players,
		CurrentPlayer: 0,
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

func (g *GameState) DeterminePriority() *Player {
	return g.Players[g.CurrentPlayer]
}

func (g *GameState) WaitForPlayerInput() {
	// Determine which player has priority
	playerWithPriority := g.DeterminePriority()

	fmt.Printf("Waiting for player input on %s\n", playerWithPriority.Turn.Phase)

	AITurnTimeout := 5 * time.Second

	// Wait for input from the specific player who has priority
	if playerWithPriority.IsAI {
		select {
		case action := <-playerWithPriority.InputChan:
			g.ProcessAction(playerWithPriority, action)
		case <-time.After(AITurnTimeout):
		}
	} else {
		action := <-playerWithPriority.InputChan
		g.ProcessAction(playerWithPriority, action)
	}
	// Process game rules after action
	// g.ApplyStateBasedActions()
}

func (g *GameState) ProcessAction(player *Player, action PlayerAction) {
	// Neither the player nor the AI should be able to create invalid actions

	// Process the action taken by the player
	switch action.Type {
	case "CastSpell":
		g.CastCard(player, action.Card)
	case "PlayLand":
		g.PlayLand(player, action.Card)
	case "PassPriority":
		// Pass priority to the next player
	default:
		fmt.Println("Unknown action type:", action.Type)
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
			g.WaitForPlayerInput()
		case PhaseDraw:
			g.DrawPhase(player, cards)
			g.WaitForPlayerInput()
		case PhaseMain1:
			g.MainPhase(player, cards)
			g.WaitForPlayerInput()
		case PhaseCombat:
			g.WaitForPlayerInput()
		case PhaseMain2:
			g.WaitForPlayerInput()
		case PhaseEnd:
			g.WaitForPlayerInput()
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

	return pool.CanPay(card.ManaCost)
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
