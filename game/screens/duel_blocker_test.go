package screens

import (
	"image"
	"testing"

	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/mtg/core"
)

func setupBlockerTest() (*DuelScreen, *core.Player, *core.Player, *core.Card, *core.Card) {
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
	blocker := core.NewCardFromDomain(domainCard, 100, player)
	blocker.Active = true
	blocker.CurrentZone = core.ZoneBattlefield
	player.Battlefield = append(player.Battlefield, blocker)

	attacker := core.NewCardFromDomain(domainCard, 200, opponent)
	attacker.Active = true
	attacker.CurrentZone = core.ZoneBattlefield
	opponent.Battlefield = append(opponent.Battlefield, attacker)
	gs.Attackers = append(gs.Attackers, attacker)

	s := &DuelScreen{
		gameState:        gs,
		self:             &duelPlayer{core: player},
		opponent:         &duelPlayer{core: opponent},
		pendingAttackers: make(map[core.EntityID]bool),
		pendingBlockers:  make(map[core.EntityID]core.EntityID),
		cardActions:      make(map[core.EntityID][]core.PlayerAction),
		cardImgCache:     make(map[cardImgKey]cardImgEntry),
		cardPositions:    make(map[core.EntityID]image.Point),
	}

	opponent.Turn.Phase = core.PhaseCombat
	opponent.Turn.CombatStep = core.CombatStepDeclareBlockers
	gs.ActivePlayer = 1

	s.refreshCardActions()

	return s, player, opponent, blocker, attacker
}

func TestIsInDeclareBlockers(t *testing.T) {
	s, _, opponent, _, _ := setupBlockerTest()

	if !s.isInDeclareBlockers() {
		t.Error("expected isInDeclareBlockers to return true during declare blockers")
	}

	opponent.Turn.CombatStep = core.CombatStepCombatDamage
	if s.isInDeclareBlockers() {
		t.Error("expected isInDeclareBlockers to return false outside declare blockers")
	}
}

func TestPendingBlockers_SelectBlocker(t *testing.T) {
	s, _, _, blocker, _ := setupBlockerTest()

	pos := s.getFieldCardPos(blocker, s.self, 0, false)
	mx := pos.X + fieldCardW/2
	my := pos.Y + fieldCardH/2

	s.handleBlockerClick(mx, my)

	if s.selectedBlocker != blocker.ID {
		t.Errorf("expected selectedBlocker to be %d, got %d", blocker.ID, s.selectedBlocker)
	}
}

func TestPendingBlockers_AssignBlocker(t *testing.T) {
	s, _, _, blocker, attacker := setupBlockerTest()

	s.selectedBlocker = blocker.ID

	pos := s.getFieldCardPos(attacker, s.opponent, 0, false)
	mx := pos.X + fieldCardW/2
	my := pos.Y + fieldCardH/2

	s.handleBlockerClick(mx, my)

	assignedTo, ok := s.pendingBlockers[blocker.ID]
	if !ok {
		t.Fatal("expected blocker to be in pendingBlockers")
	}
	if assignedTo != attacker.ID {
		t.Errorf("expected blocker assigned to attacker %d, got %d", attacker.ID, assignedTo)
	}
	if s.selectedBlocker != 0 {
		t.Error("expected selectedBlocker to be cleared after assignment")
	}
}

func TestPendingBlockers_RemoveBlocker(t *testing.T) {
	s, _, _, blocker, attacker := setupBlockerTest()

	s.pendingBlockers[blocker.ID] = attacker.ID

	pos := s.getFieldCardPos(blocker, s.self, 0, false)
	mx := pos.X + fieldCardW/2
	my := pos.Y + fieldCardH/2

	s.handleBlockerClick(mx, my)

	if _, ok := s.pendingBlockers[blocker.ID]; ok {
		t.Error("expected blocker to be removed from pendingBlockers after clicking it again")
	}
}

func TestPendingBlockers_DoneButtonSendsActions(t *testing.T) {
	s, player, _, blocker, attacker := setupBlockerTest()

	s.pendingBlockers[blocker.ID] = attacker.ID

	for blockerID, attackerID := range s.pendingBlockers {
		b := s.findBattlefieldCard(s.self, blockerID)
		a := s.findBattlefieldCard(s.opponent, attackerID)
		if b != nil && a != nil {
			select {
			case s.self.core.InputChan <- core.PlayerAction{
				Type:   core.ActionDeclareBlocker,
				Card:   b,
				Target: a,
			}:
			default:
				t.Fatal("failed to send DeclareBlocker action")
			}
		}
	}
	s.pendingBlockers = make(map[core.EntityID]core.EntityID)
	s.selectedBlocker = 0
	select {
	case s.self.core.InputChan <- core.PlayerAction{Type: core.ActionPassPriority}:
	default:
		t.Fatal("failed to send PassPriority")
	}

	var actions []core.PlayerAction
	for len(player.InputChan) > 0 {
		actions = append(actions, <-player.InputChan)
	}

	if len(actions) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(actions))
	}
	if actions[0].Type != core.ActionDeclareBlocker {
		t.Errorf("first action should be DeclareBlocker, got %s", actions[0].Type)
	}
	if actions[0].Card != blocker {
		t.Error("DeclareBlocker action should reference the blocker")
	}
	targetCard, ok := actions[0].Target.(*core.Card)
	if !ok {
		t.Fatal("DeclareBlocker target should be a *Card")
	}
	if targetCard != attacker {
		t.Error("DeclareBlocker target should reference the attacker")
	}
	if actions[1].Type != core.ActionPassPriority {
		t.Errorf("second action should be PassPriority, got %s", actions[1].Type)
	}
}

func TestPendingBlockers_ClearedWhenLeavingPhase(t *testing.T) {
	s, _, opponent, blocker, attacker := setupBlockerTest()

	s.pendingBlockers[blocker.ID] = attacker.ID
	s.selectedBlocker = blocker.ID

	opponent.Turn.CombatStep = core.CombatStepCombatDamage
	s.refreshCardActions()

	if len(s.pendingBlockers) != 0 {
		t.Error("pendingBlockers should be cleared when leaving declare blockers step")
	}
	if s.selectedBlocker != 0 {
		t.Error("selectedBlocker should be cleared when leaving declare blockers step")
	}
}
