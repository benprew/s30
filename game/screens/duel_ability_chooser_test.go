package screens

import (
	"image"
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"
	"github.com/google/uuid"
)

func TestPermRowFor_AnimatedLand_IsCreatureRow(t *testing.T) {
	perm := interactive.PermanentState{
		Name:       "Mishra's Factory",
		IsCreature: true,
		IsLand:     true,
	}
	if got := permRowFor(perm); got != permRowCreature {
		t.Errorf("animated land should be in creature row, got %v", got)
	}
}

func TestPermRowFor_Land_IsLandRow(t *testing.T) {
	perm := interactive.PermanentState{
		Name:   "Forest",
		IsLand: true,
	}
	if got := permRowFor(perm); got != permRowLand {
		t.Errorf("land should be in land row, got %v", got)
	}
}

func TestPerformCardAction_MultipleAbilities_EntersAbilityChoosingMode(t *testing.T) {
	permID := uuid.New()

	actions := []interactive.ActionOption{
		{
			Type:         interactive.ActionActivateAbility,
			Label:        "{T}: Add {C}",
			CardName:     "Mishra's Factory",
			PermanentID:  permID,
			AbilityIndex: 0,
		},
		{
			Type:         interactive.ActionActivateAbility,
			Label:        "{1}: This land becomes a 2/2 Assembly-Worker",
			CardName:     "Mishra's Factory",
			PermanentID:  permID,
			AbilityIndex: 1,
		},
		{
			Type:         interactive.ActionActivateAbility,
			Label:        "{T}: Target Assembly-Worker gets +1/+1",
			CardName:     "Mishra's Factory",
			PermanentID:  permID,
			AbilityIndex: 2,
			NeedsTarget:  true,
		},
	}

	s := &DuelScreen{
		cardActions:      map[uuid.UUID][]interactive.ActionOption{permID: actions},
		pendingAttackers: make(map[uuid.UUID]bool),
		pendingBlockers:  make(map[uuid.UUID]uuid.UUID),
		cardImgCache:     make(map[cardImgKey]cardImgEntry),
		cardPositions:    make(map[uuid.UUID]image.Point),
	}

	s.performCardAction(permID, "Mishra's Factory")

	if !s.isChoosingAbility() {
		t.Fatal("expected ability choosing mode to be active")
	}

	if len(s.abilityChoosingActions) != 3 {
		t.Fatalf("expected 3 actions, got %d", len(s.abilityChoosingActions))
	}
}

func TestSelectAbility_NoTarget_SendsAction(t *testing.T) {
	permID := uuid.New()
	fromTUI := make(chan interactive.PriorityAction, 1)

	actions := []interactive.ActionOption{
		{
			Type:         interactive.ActionActivateAbility,
			Label:        "{T}: Add {C}",
			CardName:     "Mishra's Factory",
			PermanentID:  permID,
			AbilityIndex: 0,
		},
		{
			Type:         interactive.ActionActivateAbility,
			Label:        "{1}: Become creature",
			CardName:     "Mishra's Factory",
			PermanentID:  permID,
			AbilityIndex: 1,
		},
	}

	human := interactive.NewHumanPlayerWithChannels("Test",
		make(chan interactive.GameMsg, 1),
		fromTUI,
		make(chan interactive.ChoiceRequest, 1),
		make(chan interactive.ChoiceResponse, 1),
	)

	s := &DuelScreen{
		cardActions:      map[uuid.UUID][]interactive.ActionOption{permID: actions},
		pendingAttackers: make(map[uuid.UUID]bool),
		pendingBlockers:  make(map[uuid.UUID]uuid.UUID),
		cardImgCache:     make(map[cardImgKey]cardImgEntry),
		cardPositions:    make(map[uuid.UUID]image.Point),
		human:            human,
	}

	s.abilityChoosingActions = actions
	s.selectAbility(0)

	if s.isChoosingAbility() {
		t.Fatal("expected ability choosing mode to be exited")
	}

	select {
	case pa := <-fromTUI:
		if pa.AbilityIndex != 0 {
			t.Fatalf("expected AbilityIndex 0, got %d", pa.AbilityIndex)
		}
		if pa.PermanentID != permID {
			t.Fatalf("expected PermanentID %v, got %v", permID, pa.PermanentID)
		}
	default:
		t.Fatal("expected action to be sent")
	}
}

func TestSelectAbility_WithTarget_EntersTargetingMode(t *testing.T) {
	permID := uuid.New()
	targetID := uuid.New()

	actions := []interactive.ActionOption{
		{
			Type:         interactive.ActionActivateAbility,
			Label:        "{T}: Add {C}",
			CardName:     "Mishra's Factory",
			PermanentID:  permID,
			AbilityIndex: 0,
		},
		{
			Type:         interactive.ActionActivateAbility,
			Label:        "{T}: Target gets +1/+1",
			CardName:     "Mishra's Factory",
			PermanentID:  permID,
			AbilityIndex: 1,
			NeedsTarget:  true,
			ValidTargets: []uuid.UUID{targetID},
		},
	}

	s := &DuelScreen{
		cardActions:      map[uuid.UUID][]interactive.ActionOption{permID: actions},
		pendingAttackers: make(map[uuid.UUID]bool),
		pendingBlockers:  make(map[uuid.UUID]uuid.UUID),
		cardImgCache:     make(map[cardImgKey]cardImgEntry),
		cardPositions:    make(map[uuid.UUID]image.Point),
	}

	s.abilityChoosingActions = actions
	s.selectAbility(1)

	if s.isChoosingAbility() {
		t.Fatal("expected ability choosing mode to be exited")
	}

	if s.targetingCardID != permID {
		t.Fatal("expected targeting mode to be active")
	}
}
