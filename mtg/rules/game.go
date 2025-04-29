package rules

import (
	"fmt"
	"slices"
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
			// Player lost
			// TODO: Implement game over logic
		}
		if len(player.Library) == 0 {
			// Player lost due to running out of cards
			// TODO: Implement game over logic
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

func (g *GameState) NextTurn() {
	player := g.Players[g.CurrentPlayer]
	player.Turn.NextPhase()
	if player.Turn.Phase == PhaseUntap {
		// Start of new turn
		// TODO: Implement untap logic
		// TODO: Implement upkeep logic
		// TODO: Implement draw logic
		player.Turn.LandPlayed = false
	}

	g.CurrentPlayer = (g.CurrentPlayer + 1) % len(g.Players)
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
	if card.CardType == "Land" && !card.IsTapped {
		return true
	}

	if card.CardType == "Creature" && !card.IsTapped && player.Turn.Phase == PhaseCombat {
		return true
	}
	return false
}

func (g *GameState) PlayCard(player *Player, card *Card) {
	if !g.CanCast(player, card) {
		return
	}

	// Pay the mana cost
	player.ManaPool.Pay(card.ManaCost)

	// Move the card from the player's hand to the battlefield
	for i, c := range player.Hand {
		if c == card {
			player.Hand = append(player.Hand[:i], player.Hand[i+1:]...)
			player.Battlefield = append(player.Battlefield, card)
			break
		}
	}
}

func (g *GameState) AvailableMana(player *Player, pPool ManaPool) (pool ManaPool) {
	for _, card := range player.Battlefield {
		if card.IsActive() || card.ManaProduction == nil {
			continue
		}
		for _, manaStr := range card.ManaProduction {
			manaRunes := []rune(manaStr)
			pool.AddMana(manaRunes)
		}
	}
	return pool
}

func (g *GameState) CanCast(player *Player, card *Card) bool {
	// Check if the card is in the player's hand
	cardInHand := slices.Contains(player.Hand, card)
	if !cardInHand {
		return false
	}

	pPool := ManaPool{}

	for color, amount := range player.ManaPool {
		pPool[color] = amount
	}

	pool := g.AvailableMana(player, pPool)

	return pool.CanPay(card.ManaCost)
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

func (g *GameState) PlayLand(player *Player, card *Card) error {
	if !g.CanPlayLand(player) {
		return fmt.Errorf("cannot play land this turn")
	}
	if card.CardType != "Land" {
		return fmt.Errorf("not a land card")
	}

	for i, c := range player.Hand {
		if c == card {
			player.Hand = slices.Delete(player.Hand, i, i+1)
			player.Battlefield = append(player.Battlefield, card)
			player.Turn.LandPlayed = true
			break
		}
	}
	return nil
}

func (g *GameState) CanPlayLand(player *Player) bool {
	if player.Turn.Phase != PhaseMain1 && player.Turn.Phase != PhaseMain2 {
		return false
	}
	if player.Turn.LandPlayed {
		return false
	}
	return true
}
