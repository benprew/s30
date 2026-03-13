package screens

import (
	"image"
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"
	"github.com/google/uuid"
)

func TestRefreshCardActions_PopulatesFromOptions(t *testing.T) {
	cardID := uuid.New()
	state := &interactive.GameState{
		Step:         "Precombat Main",
		ActivePlayer: "You",
		You: interactive.PlayerState{
			ID:   uuid.New(),
			Name: "You",
			Life: 20,
			Hand: []interactive.CardState{
				{ID: cardID, Name: "Forest", IsLand: true},
			},
		},
		Opponent: interactive.PlayerState{
			ID:   uuid.New(),
			Name: "Opponent",
			Life: 20,
		},
	}

	msg := &interactive.GameMsg{
		State:  state,
		Prompt: interactive.PromptMainPhaseAction,
		Options: []interactive.ActionOption{
			{
				Type:     interactive.ActionPlayLand,
				Label:    "Forest",
				CardName: "Forest",
				CardID:   cardID,
			},
			{
				Type:  interactive.ActionPass,
				Label: "Pass",
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

	if len(s.cardActions) == 0 {
		t.Error("refreshCardActions should populate card actions from lastMsg.Options")
	}

	actions, ok := s.cardActions[cardID]
	if !ok {
		t.Fatal("expected card actions for Forest")
	}
	if !hasActionType(actions, interactive.ActionPlayLand) {
		t.Error("expected PlayLand action for Forest")
	}
}

func TestRefreshCardActions_ClearsPendingAttackersOutsideCombat(t *testing.T) {
	creatureID := uuid.New()
	state := &interactive.GameState{
		Step:         "Precombat Main",
		ActivePlayer: "You",
		You: interactive.PlayerState{
			ID:   uuid.New(),
			Name: "You",
			Life: 20,
		},
	}

	msg := &interactive.GameMsg{
		State:  state,
		Prompt: interactive.PromptMainPhaseAction,
	}

	s := &DuelScreen{
		lastMsg:          msg,
		self:             &duelPlayer{name: "You"},
		opponent:         &duelPlayer{name: "Opponent"},
		pendingAttackers: map[uuid.UUID]bool{creatureID: true},
		pendingBlockers:  make(map[uuid.UUID]uuid.UUID),
		cardActions:      make(map[uuid.UUID][]interactive.ActionOption),
		cardImgCache:     make(map[cardImgKey]cardImgEntry),
		cardPositions:    make(map[uuid.UUID]image.Point),
	}

	s.refreshCardActions()

	if len(s.pendingAttackers) != 0 {
		t.Error("pendingAttackers should be cleared outside declare attackers step")
	}
}
