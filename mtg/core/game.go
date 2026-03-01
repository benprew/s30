package core

import (
	"fmt"
	"slices"
	"time"

	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/mtg/effects"
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
	DebugPrint        bool
	FirstTurn         bool
}

func NewGame(players []*Player, debug bool) *GameState {
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
		DebugPrint:        debug,
		FirstTurn:         true,
	}
}

func (g *GameState) debugf(format string, args ...any) {
	if g.DebugPrint {
		fmt.Printf(format, args...)
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
		player.ID = eID
		eID++
	}
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

	activePlayer := g.Players[g.ActivePlayer]
	g.debugf("  [DEBUG] WaitForPlayerInput: waiting for %s, phase=%v, combat_step=%v, is_active=%v\n",
		player.Name(), activePlayer.Turn.Phase, activePlayer.Turn.CombatStep, player == activePlayer)

	if player.WaitingChan != nil {
		select {
		case player.WaitingChan <- struct{}{}:
		default:
		}
	}

	if player.IsAI {
		select {
		case action = <-player.InputChan:
			g.debugf("  [DEBUG] WaitForPlayerInput: %s sent action %v\n", player.Name(), action.Type)
		case <-time.After(AITurnTimeout):
			g.debugf("  [DEBUG] WaitForPlayerInput: %s TIMEOUT, defaulting to pass\n", player.Name())
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

	if c.IsAura() {
		targetCreature, ok := item.Target.(*Card)
		if !ok || targetCreature.CurrentZone != ZoneBattlefield {
			if err := c.Owner.MoveTo(c, ZoneGraveyard); err != nil {
				return fmt.Errorf("unable to move aura to graveyard: %w", err)
			}
			return nil
		}
		if err := c.Owner.MoveTo(c, ZoneBattlefield); err != nil {
			return fmt.Errorf("unable to move aura to battlefield: %w", err)
		}
		c.AttachedTo = targetCreature
		targetCreature.Attachments = append(targetCreature.Attachments, c)
		return nil
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
		if err := g.TapManaSourcesFor(player, action.Card.ManaCost); err != nil {
			g.debugf("error tapping mana: %v\n", err)
			return ActPlayerPriority, nil
		}
		if err := g.CastSpell(player, action.Card, action.Target); err != nil {
			g.debugf("error casting spell: %v\n", err)
			return ActPlayerPriority, nil
		}
		if action.Target != nil {
			g.debugf("  %s casts %s targeting %s\n", player.Name(), cardStr(action.Card), targetStr(action.Target))
		} else {
			g.debugf("  %s casts %s\n", player.Name(), cardStr(action.Card))
		}
		g.Stack.ConsecutivePasses = 0
		return ActPlayerPriority, nil
	case ActionPlayLand:
		g.PlayLand(player, action.Card)
		g.debugf("  %s plays %s\n", player.Name(), cardStr(action.Card))
		return ActPlayerPriority, nil
	case ActionPassPriority:
		res, item := g.Stack.Next(EventPlayerPassesPriority, nil)
		g.debugf("after pass: res: %d, item: %v\n", res, item)
		return res, item
	}
	g.debugf("  [WARN] ProcessAction: ignoring unhandled action type %q from %s\n", action.Type, player.Name())
	return ActPlayerPriority, nil
}

func (g *GameState) RunStack() bool {
	for _, p := range g.Players {
		if len(p.InputChan) > 0 {
			g.debugf("  [WARN] RunStack: %s has %d pending actions in channel\n", p.Name(), len(p.InputChan))
			for len(p.InputChan) > 0 {
				<-p.InputChan
			}
		}
	}

	player := g.Players[g.CurrentPlayer]
	player2 := g.Players[(g.CurrentPlayer+1)%len(g.Players)]
	cnt := 0
	g.Stack.CurrentState = StateStartStack
	result, item := g.Stack.Next(-1, nil)
	for g.Stack.CurrentState != StateEmpty {
		g.CheckStateBasedActions()
		if g.hasLoser() {
			return false
		}
		if cnt > 100 {
			break
		}
		g.debugf("RunStack: res: %d, item: %v\n", result, item)
		switch result {
		case ActPlayerPriority:
			g.CheckStateBasedActions()
			action := g.WaitForPlayerInput(player)
			result, item = g.ProcessAction(player, action)
		case NonActPlayerPriority:
			g.CheckStateBasedActions()
			action := g.WaitForPlayerInput(player2)
			result, item = g.ProcessAction(player2, action)
		case Resolve:
			g.Resolve(item)
			result, item = g.Stack.Next(-1, nil)
		default:
			result, item = g.Stack.Next(-1, nil)
		}
		cnt++
	}
	for _, p := range g.Players {
		for len(p.InputChan) > 0 {
			<-p.InputChan
		}
	}
	return true
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

	g.debugf("\n=== %s's Turn (Life: %d) ===\n", player.Name(), player.LifeTotal)

	for player.Turn.Phase != PhaseEndTurn {
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
		case PhaseEnd:
			g.EndPhase(player)
		case PhaseCleanup:
			g.CleanupPhase(player)
		}

		for _, p := range g.Players {
			p.ManaPool.Drain()
		}

		g.CheckWinConditions()
		if g.hasLoser() {
			return
		}

		player.Turn.NextPhase()
	}

	g.printZoneStates()
	g.FirstTurn = false
	g.CurrentPlayer = (g.CurrentPlayer + 1) % len(g.Players)
}

func (g *GameState) CleanupPhase(player *Player) {
	// discard down to "max hand size", defaults to 7
	maxHandSize := 7
	if len(player.Hand) > maxHandSize {
		player.Turn.Discarding = true
		for len(player.Hand) > maxHandSize {
			action := g.WaitForPlayerInput(player)
			if action.Type == ActionDiscard && action.Card != nil {
				player.MoveTo(action.Card, ZoneGraveyard)
				g.debugf("  %s discards %s\n", player.Name(), action.Card.Name())
			}
		}
		player.Turn.Discarding = false
	}

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
	// TODO Process upkeep triggers
	g.debugf("UpkeepPhase: running stack\n")
	g.RunStack()
}

func (g *GameState) DrawPhase(player *Player) {
	if g.FirstTurn && g.CurrentPlayer == 0 {
		g.debugf("  %s skips draw on first turn\n", player.Name())
		return
	}
	if err := player.DrawCard(); err != nil {
		g.debugf("  %s cannot draw - loses the game!\n", player.Name())
		player.HasLost = true
		return
	}
	if g.DebugPrint {
		g.debugf("  %s draws a card. Hand: ", player.Name())
		for i, c := range player.Hand {
			if i > 0 {
				g.debugf(", ")
			}
			g.debugf("%s", c.Name())
		}
		g.debugf("\n")
	}
}

func (g *GameState) MainPhase(player *Player) {
	g.RunStack()
}

func (g *GameState) CombatPhase(player *Player) {
	opponent := g.GetOpponent(player)

	player.Turn.CombatStep = CombatStepBeginning
	g.RunStack()
	if g.hasLoser() {
		return
	}

	player.Turn.CombatStep = CombatStepDeclareAttackers
	availableAttackers := g.AvailableAttackers(player)
	if len(availableAttackers) > 0 {
		g.debugf("  Combat: Available attackers: %s\n", g.cardListString(availableAttackers))
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
			g.debugf("  %s attacks with %s (%d/%d)\n", player.Name(), cardStr(action.Card), action.Card.EffectivePower(), action.Card.EffectiveToughness())
		}
	}
	g.RunStack()
	if g.hasLoser() {
		return
	}

	if len(g.Attackers) > 0 {
		player.Turn.CombatStep = CombatStepDeclareBlockers
		for {
			action := g.WaitForPlayerInput(opponent)
			if action.Type == ActionPassPriority {
				break
			}
			if action.Type == ActionDeclareBlocker {
				if attacker, ok := action.Target.(*Card); ok {
					if err := g.DeclareBlocker(action.Card, attacker); err != nil {
						g.debugf("  error declaring blocker: %v\n", err)
						continue
					}
					g.debugf("  %s blocks %s with %s (%d/%d)\n", opponent.Name(), cardStr(attacker), cardStr(action.Card), action.Card.EffectivePower(), action.Card.EffectiveToughness())
				}
			}
		}
		g.RunStack()
		if g.hasLoser() {
			return
		}

		if g.combatHasFirstStrike() {
			player.Turn.CombatStep = CombatStepFirstStrikeDamage
			g.ResolveFirstStrikeDamage()
			g.CheckStateBasedActions()
			g.RunStack()
			if g.hasLoser() {
				return
			}
		}

		player.Turn.CombatStep = CombatStepCombatDamage
		g.printCombatDamage(player, opponent)
		g.ResolveCombatDamage()
		g.CheckStateBasedActions()
		g.RunStack()
		if g.hasLoser() {
			return
		}
	}

	player.Turn.CombatStep = CombatStepEndOfCombat
	g.RunStack()
	g.ClearCombatState()

	player.Turn.CombatStep = CombatStepNone
}

func (g *GameState) EndPhase(player *Player) {
	g.RunStack()
}

func (g *GameState) TapManaSourcesFor(player *Player, cost string) error {
	required := player.ManaPool.ParseCost(cost)
	g.debugf("TapManaSourcesFor: required: %v - pool: %c\n", required, player.ManaPool)

	g.debugf("avail mana: %c\n", g.AvailableMana(player, player.ManaPool))
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
						if err := g.ActivateManaAbility(player, card); err != nil {
							panic(err)
						}
						g.debugf("Tapping %s for color\n", card.CardName)
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

	g.debugf("avail mana: %c\n", g.AvailableMana(player, player.ManaPool))

	if required['C'] > 0 {
		for _, card := range player.Battlefield {
			if player.ManaPool.CanPay(cost) {
				break
			}
			if !card.Tapped && (card.CardType == domain.CardTypeLand || len(card.ManaProduction) > 0) {
				g.debugf("tapping %s\n", card.CardName)
				if err := g.ActivateManaAbility(player, card); err != nil {
					panic(err)
				}
				g.debugf("cost: %s pool: %c, required: %v\n", cost, player.ManaPool, player.ManaPool.ParseCost(cost))
			}
		}
		if !player.ManaPool.CanPay(cost) {
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
	g.debugf("\n--- End of Turn Zone States ---\n")
	for _, p := range g.Players {
		g.debugf("%s (Life: %d):\n", p.Name(), p.LifeTotal)
		g.debugf("  Hand: %s\n", g.cardListString(p.Hand))
		g.debugf("  Battlefield: %s\n", g.cardListString(p.Battlefield))
		g.debugf("  Graveyard: %s\n", g.cardListString(p.Graveyard))
	}
}

func (g *GameState) printCombatDamage(attacker, defender *Player) {
	for _, card := range g.Attackers {
		blockers := g.BlockerMap[card]
		if len(blockers) == 0 {
			g.debugf("  %s deals %d damage to %s\n", cardStr(card), card.EffectivePower(), defender.Name())
		} else {
			for _, blocker := range blockers {
				g.debugf("  %s and %s trade blows (%d vs %d)\n", cardStr(card), cardStr(blocker), card.EffectivePower(), blocker.EffectivePower())
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
	if card.CardType == domain.CardTypeLand {
		return false
	}

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
	if !g.cardRequiresTarget(card) {
		return true
	}
	return len(g.AvailableTargets(card)) > 0
}

func (g *GameState) cardRequiresTarget(card *Card) bool {
	if card.GetTargetSpec() == nil {
		return false
	}
	switch card.CardType {
	case domain.CardTypeInstant, domain.CardTypeSorcery:
		return true
	case domain.CardTypeEnchantment:
		return card.IsAura()
	default:
		return false
	}
}

func (g *GameState) castActionsForCard(card *Card) []PlayerAction {
	if !g.cardRequiresTarget(card) {
		return []PlayerAction{{Type: ActionCastSpell, Card: card}}
	}
	targets := g.AvailableTargets(card)
	actions := make([]PlayerAction, len(targets))
	for i, target := range targets {
		actions[i] = PlayerAction{Type: ActionCastSpell, Card: card, Target: target}
	}
	return actions
}

func (g *GameState) AvailableTargets(card *Card) []Targetable {
	spec := card.GetTargetSpec()
	if spec == nil {
		return []Targetable{}
	}

	creature_targets := []Targetable{}
	land_targets := []Targetable{}
	player_targets := []Targetable{}
	for _, player := range g.Players {
		player_targets = append(player_targets, player) // dead players are still valid targets
	}

	for _, player := range g.Players {
		for _, c := range player.Battlefield {
			switch c.CardType {
			case domain.CardTypeCreature:
				creature_targets = append(creature_targets, c)
			case domain.CardTypeLand:
				land_targets = append(land_targets, c)
			}
		}
	}

	switch spec.Type {
	case "creature":
		return creature_targets
	case "land":
		return land_targets
	case "player":
		return player_targets
	default: // any
		return append(creature_targets, player_targets...)
	}
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

	if player.Turn.Discarding {
		for _, card := range player.Hand {
			actions = append(actions, PlayerAction{Type: ActionDiscard, Card: card})
		}
		return actions
	}

	stackActive := !g.Stack.IsEmpty()

	hand := make([]*Card, len(player.Hand))
	copy(hand, player.Hand)

	if !stackActive && (player.Turn.Phase == PhaseMain1 || player.Turn.Phase == PhaseMain2) {
		for _, card := range hand {
			if card == nil {
				continue
			}
			if g.CanPlayLand(player, card) {
				actions = append(actions, PlayerAction{Type: ActionPlayLand, Card: card})
			}
		}
		for _, card := range hand {
			if card == nil {
				continue
			}
			if g.CanCast(player, card) {
				actions = append(actions, g.castActionsForCard(card)...)
			}
		}
	} else {
		for _, card := range hand {
			if card == nil {
				continue
			}
			if g.CanCast(player, card) && (card.CardType == domain.CardTypeInstant || card.HasKeyword(effects.KeywordFlash)) {
				actions = append(actions, g.castActionsForCard(card)...)
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
					if attacker.HasKeyword(effects.KeywordFlying) &&
						!card.HasKeyword(effects.KeywordFlying) &&
						!card.HasKeyword(effects.KeywordReach) {
						continue
					}
					if landType := attacker.LandwalkType(); landType != "" && player.ControlsLandType(landType) {
						continue
					}
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
