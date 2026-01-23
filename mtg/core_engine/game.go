package core_engine

import (
	"fmt"
	"slices"
	"time"

	"github.com/benprew/s30/game/domain"
)

type GameState struct {
	Players           []*Player
	CurrentPlayer     int
	ActivePlayer      int
	CurrentState      string
	ConsecutivePasses int
	Stack             Stack
	CardMap           map[EntityID]*Card
}

func NewGame(players []*Player) *GameState {
	return &GameState{
		Players:           players,
		CurrentPlayer:     0,
		ActivePlayer:      0,
		CurrentState:      "WaitingForAction",
		ConsecutivePasses: 0,
		Stack:             NewStack(),
		CardMap:           make(map[EntityID]*Card),
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
	eID := EntityID(1)
	for _, player := range g.Players {
		for j := range player.Library {
			player.Library[j].ID = eID
			g.CardMap[eID] = player.Library[j]
			eID++
		}
	}
	for _, player := range g.Players {
		for range 7 {
			player.DrawCard()
		}
	}
}

// When game receives player input, it should be the total input needed for the spell. Ie. for bolt
// it should receive that it's casting LB and have already chosen a target. No decisions should be
// remaining to be made. The UI should handle all the decisions before it's passed off to the game
func (g *GameState) WaitForPlayerInput(player *Player) (action PlayerAction) {
	AITurnTimeout := 5 * time.Second

	if player.IsAI {
		select {
		case action = <-player.InputChan:
		case <-time.After(AITurnTimeout):
			action = PlayerAction{Type: ActionPassPriority}
		}
	} else {
		action = <-player.InputChan
	}

	return action
}

func (g *GameState) Resolve(item *StackItem) error {
	if item == nil {
		return fmt.Errorf("Resolve: Item is nil")
	}

	c := item.Card
	if c == nil {
		return fmt.Errorf("Resolve: nil card")
	}
	events := item.Events

	for _, e := range events {
		e.Resolve()
		if e.Target() != nil && e.Target().TargetType() == TargetTypeCard {
			tgt := e.Target().(*Card)
			if tgt.IsDead() {
				if err := tgt.Owner.MoveTo(tgt, ZoneGraveyard); err != nil {
					return fmt.Errorf("unable to move card to graveyard: %w", err)
				}
			}
		}
	}

	if c.CardType != domain.CardTypeInstant && c.CardType != domain.CardTypeSorcery {
		if err := c.Owner.MoveTo(c, ZoneBattlefield); err != nil {
			return fmt.Errorf("unable to move card to battlefield: %w", err)
		}
	} else {
		if err := c.Owner.MoveTo(c, ZoneGraveyard); err != nil {
			return fmt.Errorf("unable to move card to graveyard: %w", err)
		}
	}
	return nil
}

type FindArgs struct {
	ID   EntityID
	Name string
}

func (g *GameState) FindCard(args FindArgs, zone []*Card) *Card {
	if args.ID > 0 && zone == nil {
		return g.CardMap[args.ID]
	}
	for _, c := range zone {
		if c.Name() == args.Name || c.ID == args.ID {
			return c
		}
	}
	return nil
}

func (g *GameState) ProcessAction(player *Player, action PlayerAction) (StackResult, *StackItem) {
	switch action.Type {
	case ActionCastSpell:
		// TODO: spells will have decisions that need to be made when cast
		// the player will need to make those decisions (ie Lightning Bolt target)
		g.Stack.Next(EventPlayerAddsAction, &StackItem{Player: player, Events: action.Card.Actions})
		g.CastSpell(player, action.Card, nil) // TODO: fix this, add target where appropriate
	case ActionPlayLand:
		g.PlayLand(player, action.Card)
	case ActionPassPriority:
		return g.Stack.Next(EventPlayerPassesPriority, nil)
	}
	return -1, nil
}

func (g *GameState) RunStack() {
	player := g.Players[g.CurrentPlayer]
	player2 := g.Players[(g.CurrentPlayer+1)%len(g.Players)]
	cnt := 0
	result, item := g.Stack.Next(-1, nil)
	for g.Stack.CurrentState != StateEmpty {
		if cnt > 500 {
			break
		}
		switch result {
		case ActPlayerPriority:
			action := g.WaitForPlayerInput(player)
			result, _ = g.ProcessAction(player, action)
		case NonActPlayerPriority:
			action := g.WaitForPlayerInput(player2)
			result, _ = g.ProcessAction(player2, action)
		case Resolve:
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
		cards, err := g.CardsWithActions(player)
		if err != nil {
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
	if card.CardType == domain.CardTypeLand && !card.Tapped {
		return true
	}

	// Creatures can tap for abilities at instant speed, but attacking/blocking tap
	// happens during combat. This function seems to be for tapping for mana or abilities.
	// Assuming tapping for mana/abilities is possible if not tapped.
	// If tapping is only for attacking/blocking, the combat phase check is relevant.
	// Let's keep the combat phase check for now based on the original code's structure.
	if card.CardType == domain.CardTypeCreature && !card.Tapped && player.Turn.Phase == PhaseCombat {
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
	if card.CardType != domain.CardTypeLand {
		return false
	}

	return true
}

func (g *GameState) AvailableActions(player *Player) []PlayerAction {
	actions := []PlayerAction{}

	for _, card := range player.Hand {
		if g.CanPlayLand(player, card) {
			actions = append(actions, PlayerAction{Type: ActionPlayLand, Card: card})
		}
	}

	for _, card := range player.Hand {
		if g.CanCast(player, card) {
			actions = append(actions, PlayerAction{Type: ActionCastSpell, Card: card})
		}
	}

	actions = append(actions, PlayerAction{Type: ActionPassPriority, Card: nil})

	return actions
}

func PlayGame(g *GameState) []*Player {
	maxTurns := 500
	for totalTurns := range maxTurns {
		g.CheckWinConditions()

		nonLosingPlayers := []*Player{}
		for _, player := range g.Players {
			if !player.HasLost {
				nonLosingPlayers = append(nonLosingPlayers, player)
			}
		}

		if len(nonLosingPlayers) <= 1 {
			return nonLosingPlayers
		}

		g.NextTurn()

		if totalTurns > maxTurns {
			return nonLosingPlayers
		}
	}
	return nil
}
