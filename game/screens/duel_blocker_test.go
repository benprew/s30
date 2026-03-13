package screens

import (
	"image"
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"
	"github.com/google/uuid"
)

func setupBlockerTest() (*DuelScreen, interactive.PermanentState, interactive.PermanentState) {
	blockerID := uuid.New()
	attackerID := uuid.New()

	blocker := interactive.PermanentState{
		ID:         blockerID,
		Name:       "Llanowar Elves",
		Power:      1,
		Toughness:  1,
		IsCreature: true,
	}

	attacker := interactive.PermanentState{
		ID:         attackerID,
		Name:       "Llanowar Elves",
		Power:      1,
		Toughness:  1,
		IsCreature: true,
		Attacking:  true,
	}

	state := &interactive.GameState{
		Step:         stepDeclareBlockers,
		ActivePlayer: "Opponent",
		You: interactive.PlayerState{
			ID:          uuid.New(),
			Name:        "You",
			Life:        20,
			Battlefield: []interactive.PermanentState{blocker},
		},
		Opponent: interactive.PlayerState{
			ID:          uuid.New(),
			Name:        "Opponent",
			Life:        20,
			Battlefield: []interactive.PermanentState{attacker},
		},
	}

	msg := &interactive.GameMsg{
		State:  state,
		Prompt: interactive.PromptDeclareBlockers,
		Options: []interactive.ActionOption{
			{
				Type:        interactive.ActionSelectBlockers,
				Label:       "Llanowar Elves",
				CardName:    "Llanowar Elves",
				PermanentID: blockerID,
			},
		},
	}

	s := &DuelScreen{
		lastMsg:          msg,
		self:             &duelPlayer{name: "You"},
		opponent:         &duelPlayer{name: "Opponent"},
		pendingAttackers: make(map[uuid.UUID]bool),
		pendingBlockers:  make(map[uuid.UUID]uuid.UUID),
		cardActions:      make(map[uuid.UUID][]interactive.ActionOption),
		cardImgCache:     make(map[cardImgKey]cardImgEntry),
		cardPositions:    make(map[uuid.UUID]image.Point),
	}

	s.refreshCardActions()

	return s, blocker, attacker
}

func TestIsInDeclareBlockers(t *testing.T) {
	s, _, _ := setupBlockerTest()

	if !s.isInDeclareBlockers() {
		t.Error("expected isInDeclareBlockers to return true during declare blockers")
	}

	s.lastMsg.Prompt = interactive.PromptNone
	if s.isInDeclareBlockers() {
		t.Error("expected isInDeclareBlockers to return false outside declare blockers")
	}
}

func TestPendingBlockers_SelectBlocker(t *testing.T) {
	s, blocker, _ := setupBlockerTest()

	pos := s.getFieldCardPos(blocker, s.self, 0, false)
	mx := pos.X + fieldCardW/2
	my := pos.Y + fieldCardH/2

	s.handleBlockerClick(mx, my)

	if s.selectedBlocker != blocker.ID {
		t.Errorf("expected selectedBlocker to be %v, got %v", blocker.ID, s.selectedBlocker)
	}
}

func TestPendingBlockers_AssignBlocker(t *testing.T) {
	s, blocker, attacker := setupBlockerTest()

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
		t.Errorf("expected blocker assigned to attacker %v, got %v", attacker.ID, assignedTo)
	}
	if s.selectedBlocker != uuid.Nil {
		t.Error("expected selectedBlocker to be cleared after assignment")
	}
}

func TestPendingBlockers_RemoveBlocker(t *testing.T) {
	s, blocker, attacker := setupBlockerTest()

	s.pendingBlockers[blocker.ID] = attacker.ID

	pos := s.getFieldCardPos(blocker, s.self, 0, false)
	mx := pos.X + fieldCardW/2
	my := pos.Y + fieldCardH/2

	s.handleBlockerClick(mx, my)

	if _, ok := s.pendingBlockers[blocker.ID]; ok {
		t.Error("expected blocker to be removed from pendingBlockers after clicking it again")
	}
}

func TestPendingBlockers_ClearedWhenLeavingPhase(t *testing.T) {
	s, blocker, attacker := setupBlockerTest()

	s.pendingBlockers[blocker.ID] = attacker.ID
	s.selectedBlocker = blocker.ID

	s.lastMsg.Prompt = interactive.PromptNone
	s.lastMsg.State.Step = stepCombatDamage
	s.refreshCardActions()

	if len(s.pendingBlockers) != 0 {
		t.Error("pendingBlockers should be cleared when leaving declare blockers step")
	}
	if s.selectedBlocker != uuid.Nil {
		t.Error("selectedBlocker should be cleared when leaving declare blockers step")
	}
}
