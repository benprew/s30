package screens

import (
	"image"
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"
	"github.com/google/uuid"
)

func setupAttackerTest() (*DuelScreen, interactive.PermanentState) {
	creatureID := uuid.New()
	creature := interactive.PermanentState{
		ID:         creatureID,
		Name:       "Llanowar Elves",
		Power:      1,
		Toughness:  1,
		IsCreature: true,
	}

	state := &interactive.GameState{
		Step:         stepDeclareAttackers,
		ActivePlayer: "You",
		You: interactive.PlayerState{
			ID:          uuid.New(),
			Name:        "You",
			Life:        20,
			Battlefield: []interactive.PermanentState{creature},
		},
		Opponent: interactive.PlayerState{
			ID:   uuid.New(),
			Name: "Opponent",
			Life: 20,
		},
	}

	msg := &interactive.GameMsg{
		State:  state,
		Prompt: interactive.PromptDeclareAttackers,
		Options: []interactive.ActionOption{
			{
				Type:        interactive.ActionSelectAttackers,
				Label:       "Llanowar Elves",
				CardName:    "Llanowar Elves",
				PermanentID: creatureID,
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

	return s, creature
}

func TestPendingAttackers_ToggleOn(t *testing.T) {
	s, creature := setupAttackerTest()

	actions, ok := s.cardActions[creature.ID]
	if !ok || len(actions) == 0 {
		t.Fatal("expected action for creature")
	}
	if !hasActionType(actions, interactive.ActionSelectAttackers) {
		t.Fatal("expected ActionSelectAttackers in actions")
	}

	s.pendingAttackers[creature.ID] = true
	if !s.pendingAttackers[creature.ID] {
		t.Error("creature should be in pendingAttackers after toggle on")
	}
}

func TestPendingAttackers_ToggleOff(t *testing.T) {
	s, creature := setupAttackerTest()

	s.pendingAttackers[creature.ID] = true
	delete(s.pendingAttackers, creature.ID)

	if s.pendingAttackers[creature.ID] {
		t.Error("creature should not be in pendingAttackers after toggle off")
	}
}

func TestPendingAttackers_ClearedWhenLeavingDeclareAttackers(t *testing.T) {
	s, creature := setupAttackerTest()

	s.pendingAttackers[creature.ID] = true

	s.lastMsg.State.Step = stepDeclareBlockers
	s.refreshCardActions()

	if len(s.pendingAttackers) != 0 {
		t.Error("pendingAttackers should be cleared when leaving declare attackers step")
	}
}

func TestPendingAttackers_NotClearedDuringDeclareAttackers(t *testing.T) {
	s, creature := setupAttackerTest()

	s.pendingAttackers[creature.ID] = true
	s.refreshCardActions()

	if !s.pendingAttackers[creature.ID] {
		t.Error("pendingAttackers should not be cleared while still in declare attackers step")
	}
}
