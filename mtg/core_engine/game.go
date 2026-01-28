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
	Attackers         []*Card
	BlockerMap        map[*Card][]*Card
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
		Attackers:         []*Card{},
		BlockerMap:        make(map[*Card][]*Card),
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
			g.CheckStateBasedActions()
			action := g.WaitForPlayerInput(player)
			result, _ = g.ProcessAction(player, action)
		case NonActPlayerPriority:
			g.CheckStateBasedActions()
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

func (g *GameState) CheckStateBasedActions() {
	g.CleanupDeadCreatures()
	g.CheckWinConditions()
}

func (g *GameState) NextTurn() {
	player := g.Players[g.CurrentPlayer]
	g.ActivePlayer = g.CurrentPlayer
	player.Turn.Phase = PhaseUntap
	player.Turn.LandPlayed = false

	fmt.Printf("\n=== %s's Turn (Life: %d) ===\n", player.Name(), player.LifeTotal)

	for player.Turn.Phase != PhaseEnd {
		switch player.Turn.Phase {
		case PhaseUntap:
			g.UntapPhase(player)
		case PhaseUpkeep:
			g.UpkeepPhase(player)
		case PhaseDraw:
			g.DrawPhase(player)
		case PhaseMain1, PhaseMain2:
			g.MainPhase(player)
		case PhaseCombat:
			g.CombatPhase(player)
		}

		g.CheckWinConditions()
		if g.hasLoser() {
			return
		}
		player.Turn.NextPhase()
	}

	g.CleanupEndOfTurnEffects()
	g.printZoneStates()
	g.CurrentPlayer = (g.CurrentPlayer + 1) % len(g.Players)
}

func (g *GameState) CleanupEndOfTurnEffects() {
	for _, player := range g.Players {
		for _, card := range player.Battlefield {
			card.ClearEndOfTurnEffects()
		}
	}
}

func (g *GameState) UntapPhase(player *Player) {
	for i := range player.Battlefield {
		card := player.Battlefield[i]
		card.Tapped = false
		card.Active = true
	}
}

func (g *GameState) UpkeepPhase(player *Player) {
}

func (g *GameState) DrawPhase(player *Player) {
	if err := player.DrawCard(); err != nil {
		fmt.Printf("  %s cannot draw - loses the game!\n", player.Name())
		player.HasLost = true
		return
	}
	fmt.Printf("  %s draws a card\n", player.Name())
}

func (g *GameState) MainPhase(player *Player) {
	for {
		action := g.WaitForPlayerInput(player)
		switch action.Type {
		case ActionPassPriority:
			return
		case ActionPlayLand:
			g.PlayLand(player, action.Card)
			fmt.Printf("  %s plays %s\n", player.Name(), cardStr(action.Card))
		case ActionCastSpell:
			if err := g.TapLandsForMana(player, action.Card.ManaCost); err != nil {
				continue
			}
			if err := g.CastSpell(player, action.Card, action.Target); err != nil {
				continue
			}
			if action.Target != nil {
				fmt.Printf("  %s casts %s targeting %s\n", player.Name(), cardStr(action.Card), targetStr(action.Target))
			} else {
				fmt.Printf("  %s casts %s\n", player.Name(), cardStr(action.Card))
			}
			if item := g.Stack.Pop(); item != nil {
				g.Resolve(item)
			}
			g.CheckWinConditions()
			if g.hasLoser() {
				return
			}
		}
	}
}

func (g *GameState) CombatPhase(player *Player) {
	player.Turn.CombatStep = CombatStepBeginning

	player.Turn.CombatStep = CombatStepDeclareAttackers
	availableAttackers := g.AvailableAttackers(player)
	if len(availableAttackers) > 0 {
		fmt.Printf("  Combat: Available attackers: %s\n", g.cardListString(availableAttackers))
	}
	for {
		action := g.WaitForPlayerInput(player)
		if action.Type == ActionPassPriority {
			break
		}
		if action.Type == ActionDeclareAttacker {
			if err := g.DeclareAttacker(action.Card); err != nil {
				continue
			}
			g.Attackers = append(g.Attackers, action.Card)
			fmt.Printf("  %s attacks with %s (%d/%d)\n", player.Name(), cardStr(action.Card), action.Card.EffectivePower(), action.Card.EffectiveToughness())
		}
	}

	if len(g.Attackers) == 0 {
		player.Turn.CombatStep = CombatStepEndOfCombat
		return
	}

	player.Turn.CombatStep = CombatStepDeclareBlockers
	opponent := g.GetOpponent(player)
	for {
		action := g.WaitForPlayerInput(opponent)
		if action.Type == ActionPassPriority {
			break
		}
		if action.Type == ActionDeclareBlocker {
			if attacker, ok := action.Target.(*Card); ok {
				if err := g.DeclareBlocker(action.Card, attacker); err != nil {
					continue
				}
				fmt.Printf("  %s blocks %s with %s (%d/%d)\n", opponent.Name(), cardStr(attacker), cardStr(action.Card), action.Card.EffectivePower(), action.Card.EffectiveToughness())
			}
		}
	}

	player.Turn.CombatStep = CombatStepCombatDamage
	g.printCombatDamage(player, opponent)
	g.ResolveCombatDamage()
	g.CheckStateBasedActions()

	player.Turn.CombatStep = CombatStepEndOfCombat
	g.ClearCombatState()
}

func (g *GameState) TapLandsForMana(player *Player, cost string) error {
	required := player.ManaPool.ParseCost(cost)

	for color, needed := range required {
		if color == 'C' {
			continue
		}
		tapped := 0
		for _, card := range player.Battlefield {
			if tapped >= needed {
				break
			}
			if card.CardType == domain.CardTypeLand && !card.Tapped {
				for _, mana := range card.ManaProduction {
					if len(mana) == 1 && rune(mana[0]) == color {
						g.TapLandForMana(player, card)
						tapped++
						break
					}
				}
			}
		}
		if tapped < needed {
			return fmt.Errorf("not enough untapped lands for %c", color)
		}
	}

	colorlessNeeded := required['C']
	if colorlessNeeded > 0 {
		tapped := 0
		for _, card := range player.Battlefield {
			if tapped >= colorlessNeeded {
				break
			}
			if card.CardType == domain.CardTypeLand && !card.Tapped {
				g.TapLandForMana(player, card)
				tapped++
			}
		}
		if tapped < colorlessNeeded {
			return fmt.Errorf("not enough untapped lands for colorless")
		}
	}
	return nil
}

func (g *GameState) GetOpponent(player *Player) *Player {
	for _, p := range g.Players {
		if p != player {
			return p
		}
	}
	return nil
}

func (g *GameState) hasLoser() bool {
	for _, p := range g.Players {
		if p.HasLost {
			return true
		}
	}
	return false
}

func (g *GameState) printZoneStates() {
	fmt.Println("\n--- End of Turn Zone States ---")
	for _, p := range g.Players {
		fmt.Printf("%s (Life: %d):\n", p.Name(), p.LifeTotal)
		fmt.Printf("  Hand: %s\n", g.cardListString(p.Hand))
		fmt.Printf("  Battlefield: %s\n", g.cardListString(p.Battlefield))
		fmt.Printf("  Graveyard: %s\n", g.cardListString(p.Graveyard))
	}
}

func (g *GameState) printCombatDamage(attacker, defender *Player) {
	for _, card := range g.Attackers {
		blockers := g.BlockerMap[card]
		if len(blockers) == 0 {
			fmt.Printf("  %s deals %d damage to %s\n", cardStr(card), card.EffectivePower(), defender.Name())
		} else {
			for _, blocker := range blockers {
				fmt.Printf("  %s and %s trade blows (%d vs %d)\n", cardStr(card), cardStr(blocker), card.EffectivePower(), blocker.EffectivePower())
			}
		}
	}
}

func (g *GameState) cardListString(cards []*Card) string {
	if len(cards) == 0 {
		return "(empty)"
	}
	names := make([]string, len(cards))
	for i, c := range cards {
		names[i] = cardStr(c)
	}
	return fmt.Sprintf("%v", names)
}

func cardStr(c *Card) string {
	return fmt.Sprintf("%s#%d", c.Name(), c.ID)
}

func targetStr(t Targetable) string {
	if c, ok := t.(*Card); ok {
		return cardStr(c)
	}
	return t.Name()
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

func (g *GameState) hasTarget(card *Card) bool {
	if card.Targetable == "DamageableTarget" {
		return true
	}
	if card.TargetsCreaturesOnly() {
		return len(g.AvailableTargets(card)) > 0
	}
	return true
}

func (g *GameState) AvailableTargets(card *Card) []Targetable {
	targets := []Targetable{}

	if card.Targetable == "DamageableTarget" || card.Name() == "Lightning Bolt" {
		for _, player := range g.Players {
			if !player.IsDead() {
				targets = append(targets, player)
			}
		}
		for _, player := range g.Players {
			for _, c := range player.Battlefield {
				if c.CardType == domain.CardTypeCreature && !c.IsDead() {
					targets = append(targets, c)
				}
			}
		}
	}

	if card.TargetsCreaturesOnly() {
		for _, player := range g.Players {
			for _, c := range player.Battlefield {
				if c.CardType == domain.CardTypeCreature && !c.IsDead() {
					targets = append(targets, c)
				}
			}
		}
	}

	return targets
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
	activePlayer := g.Players[g.ActivePlayer]

	if player.Turn.Phase == PhaseMain1 || player.Turn.Phase == PhaseMain2 {
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
	} else {
		for _, card := range player.Hand {
			if g.CanCast(player, card) && card.CardType == domain.CardTypeInstant {
				actions = append(actions, PlayerAction{Type: ActionCastSpell, Card: card})
			}
		}
	}

	if player == activePlayer && player.Turn.Phase == PhaseCombat {
		if player.Turn.CombatStep == CombatStepDeclareAttackers {
			for _, card := range g.AvailableAttackers(player) {
				actions = append(actions, PlayerAction{Type: ActionDeclareAttacker, Card: card})
			}
		}
	}

	if player != activePlayer && activePlayer.Turn.Phase == PhaseCombat {
		if activePlayer.Turn.CombatStep == CombatStepDeclareBlockers {
			for _, card := range g.AvailableBlockers(player) {
				for _, attacker := range g.Attackers {
					actions = append(actions, PlayerAction{
						Type:   ActionDeclareBlocker,
						Card:   card,
						Target: attacker,
					})
				}
			}
		}
	}

	actions = append(actions, PlayerAction{Type: ActionPassPriority, Card: nil})

	return actions
}

func PlayGame(g *GameState, maxTurns int) []*Player {
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
