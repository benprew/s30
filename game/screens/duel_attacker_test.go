package screens

import (
	"image"
	"testing"

	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/mtg/core"
)

func setupAttackerTest() (*DuelScreen, *core.Player, *core.Player, *core.Card) {
	player := &core.Player{
		ID:          0,
		LifeTotal:   20,
		ManaPool:    core.ManaPool{},
		Hand:        []*core.Card{},
		Library:     []*core.Card{},
		Battlefield: []*core.Card{},
		Graveyard:   []*core.Card{},
		Exile:       []*core.Card{},
		Turn:        &core.Turn{},
		InputChan:   make(chan core.PlayerAction, 100),
		WaitingChan: make(chan struct{}, 1),
	}
	opponent := &core.Player{
		ID:          1,
		LifeTotal:   20,
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

	gs := core.NewGame([]*core.Player{player, opponent}, false)

	domainCard := domain.FindCardByName("Llanowar Elves")
	creature := core.NewCardFromDomain(domainCard, 100, player)
	creature.Active = true
	creature.CurrentZone = core.ZoneBattlefield
	player.Battlefield = append(player.Battlefield, creature)

	s := &DuelScreen{
		gameState:        gs,
		self:             &duelPlayer{core: player},
		opponent:         &duelPlayer{core: opponent},
		pendingAttackers: make(map[core.EntityID]bool),
		cardActions:      make(map[core.EntityID][]core.PlayerAction),
		cardImgCache:     make(map[cardImgKey]cardImgEntry),
		cardPositions:    make(map[core.EntityID]image.Point),
	}

	player.Turn.Phase = core.PhaseCombat
	player.Turn.CombatStep = core.CombatStepDeclareAttackers
	gs.ActivePlayer = 0

	s.refreshCardActions()

	return s, player, opponent, creature
}

func TestPendingAttackers_ToggleOn(t *testing.T) {
	s, _, _, creature := setupAttackerTest()

	actions, ok := s.cardActions[creature.ID]
	if !ok || len(actions) == 0 {
		t.Fatal("expected DeclareAttacker action for creature")
	}
	if !hasActionType(actions, core.ActionDeclareAttacker) {
		t.Fatalf("expected action type %s in actions", core.ActionDeclareAttacker)
	}

	s.pendingAttackers[creature.ID] = true

	if !s.pendingAttackers[creature.ID] {
		t.Error("creature should be in pendingAttackers after toggle on")
	}
}

func TestPendingAttackers_ToggleOff(t *testing.T) {
	s, _, _, creature := setupAttackerTest()

	s.pendingAttackers[creature.ID] = true
	delete(s.pendingAttackers, creature.ID)

	if s.pendingAttackers[creature.ID] {
		t.Error("creature should not be in pendingAttackers after toggle off")
	}
}

func TestPendingAttackers_DoneButtonSendsActions(t *testing.T) {
	s, player, _, creature := setupAttackerTest()

	s.pendingAttackers[creature.ID] = true

	for id, actions := range s.cardActions {
		if !s.pendingAttackers[id] {
			continue
		}
		for _, action := range actions {
			if action.Type == core.ActionDeclareAttacker {
				select {
				case s.self.core.InputChan <- action:
				default:
					t.Fatal("failed to send action to InputChan")
				}
				break
			}
		}
	}
	s.pendingAttackers = make(map[core.EntityID]bool)
	select {
	case s.self.core.InputChan <- core.PlayerAction{Type: core.ActionPassPriority}:
	default:
		t.Fatal("failed to send PassPriority to InputChan")
	}

	var actions []core.PlayerAction
	for len(player.InputChan) > 0 {
		actions = append(actions, <-player.InputChan)
	}

	if len(actions) != 2 {
		t.Fatalf("expected 2 actions on InputChan, got %d", len(actions))
	}
	if actions[0].Type != core.ActionDeclareAttacker {
		t.Errorf("first action should be DeclareAttacker, got %s", actions[0].Type)
	}
	if actions[0].Card != creature {
		t.Error("DeclareAttacker action should reference the creature")
	}
	if actions[1].Type != core.ActionPassPriority {
		t.Errorf("second action should be PassPriority, got %s", actions[1].Type)
	}

	if len(s.pendingAttackers) != 0 {
		t.Error("pendingAttackers should be cleared after Done")
	}
}

func TestPendingAttackers_DoneWithNoPendingOnlySendsPass(t *testing.T) {
	s, player, _, _ := setupAttackerTest()

	for id, cardActions := range s.cardActions {
		if !s.pendingAttackers[id] {
			continue
		}
		for _, action := range cardActions {
			if action.Type == core.ActionDeclareAttacker {
				select {
				case s.self.core.InputChan <- action:
				default:
				}
				break
			}
		}
	}
	s.pendingAttackers = make(map[core.EntityID]bool)
	select {
	case s.self.core.InputChan <- core.PlayerAction{Type: core.ActionPassPriority}:
	default:
		t.Fatal("failed to send PassPriority")
	}

	var actions []core.PlayerAction
	for len(player.InputChan) > 0 {
		actions = append(actions, <-player.InputChan)
	}

	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actions))
	}
	if actions[0].Type != core.ActionPassPriority {
		t.Errorf("expected PassPriority, got %s", actions[0].Type)
	}
}

func TestPendingAttackers_ClearedWhenLeavingDeclareAttackers(t *testing.T) {
	s, player, _, creature := setupAttackerTest()

	s.pendingAttackers[creature.ID] = true

	player.Turn.CombatStep = core.CombatStepDeclareBlockers
	s.refreshCardActions()

	if len(s.pendingAttackers) != 0 {
		t.Error("pendingAttackers should be cleared when leaving declare attackers step")
	}
}

func TestPendingAttackers_NotClearedDuringDeclareAttackers(t *testing.T) {
	s, _, _, creature := setupAttackerTest()

	s.pendingAttackers[creature.ID] = true
	s.refreshCardActions()

	if !s.pendingAttackers[creature.ID] {
		t.Error("pendingAttackers should not be cleared while still in declare attackers step")
	}
}

func TestPendingAttackers_MultipleSentInOrder(t *testing.T) {
	s, player, _, _ := setupAttackerTest()

	domainCard := domain.FindCardByName("Llanowar Elves")
	creature2 := core.NewCardFromDomain(domainCard, 101, player)
	creature2.Active = true
	creature2.CurrentZone = core.ZoneBattlefield
	player.Battlefield = append(player.Battlefield, creature2)

	s.refreshCardActions()

	for _, card := range player.Battlefield {
		s.pendingAttackers[card.ID] = true
	}

	for id, cardActions := range s.cardActions {
		if !s.pendingAttackers[id] {
			continue
		}
		for _, action := range cardActions {
			if action.Type == core.ActionDeclareAttacker {
				select {
				case s.self.core.InputChan <- action:
				default:
				}
				break
			}
		}
	}
	s.pendingAttackers = make(map[core.EntityID]bool)
	select {
	case s.self.core.InputChan <- core.PlayerAction{Type: core.ActionPassPriority}:
	default:
	}

	var actions []core.PlayerAction
	for len(player.InputChan) > 0 {
		actions = append(actions, <-player.InputChan)
	}

	if len(actions) != 3 {
		t.Fatalf("expected 3 actions (2 attackers + pass), got %d", len(actions))
	}

	attackerCount := 0
	for _, a := range actions {
		if a.Type == core.ActionDeclareAttacker {
			attackerCount++
		}
	}
	if attackerCount != 2 {
		t.Errorf("expected 2 DeclareAttacker actions, got %d", attackerCount)
	}
	if actions[len(actions)-1].Type != core.ActionPassPriority {
		t.Errorf("last action should be PassPriority, got %s", actions[len(actions)-1].Type)
	}
}
