package duel

import (
	"testing"
	"uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/interactive"
)

func TestDetonateTargetingFiltersArtifactsByChosenX(t *testing.T) {
	ornithopterID := uuid.New()
	tomeID := uuid.New()

	state := &interactive.GameState{
		Opponent: interactive.PlayerState{
			Name: "Enemy",
			Battlefield: []interactive.PermanentState{
				{ID: ornithopterID, Name: "Ornithopter"},
				{ID: tomeID, Name: "Jayemdae Tome"},
			},
		},
	}

	action := interactive.ActionOption{
		Type:         interactive.ActionCastSpell,
		CardID:       uuid.New(),
		CardName:     "Detonate",
		NeedsX:       true,
		MaxXValue:    6,
		NeedsTarget:  true,
		TargetType:   mage.TargetArtifactWithManaValueX(),
		ValidTargets: []uuid.UUID{ornithopterID, tomeID},
	}

	human, fromTUI := newTestHuman()
	s := &DuelScreen{human: human, lastMsg: &interactive.GameMsg{State: state}}

	s.enterXChoosingMode([]interactive.ActionOption{action})
	s.selectXValue(4)
	if _, ok := s.targetingActions[tomeID]; !ok {
		t.Fatal("expected Jayemdae Tome (CMC 4) to be targetable when X = 4")
	}
	if _, ok := s.targetingActions[ornithopterID]; ok {
		t.Fatal("did not expect Ornithopter (CMC 0) to be targetable when X = 4")
	}

	s.selectTarget(tomeID)
	if !s.submitTargetingAction() {
		t.Fatal("expected submitTargetingAction to succeed")
	}
	select {
	case pa := <-fromTUI:
		if pa.XValue != 4 {
			t.Fatalf("expected XValue 4, got %d", pa.XValue)
		}
		if len(pa.Targets) != 1 || pa.Targets[0] != tomeID {
			t.Fatalf("expected target %v, got %v", tomeID, pa.Targets)
		}
	default:
		t.Fatal("expected priority action to be sent")
	}

	s.exitTargetingMode()
	s.enterXChoosingMode([]interactive.ActionOption{action})
	s.selectXValue(0)
	if _, ok := s.targetingActions[ornithopterID]; !ok {
		t.Fatal("expected Ornithopter (CMC 0) to be targetable when X = 0")
	}
	if _, ok := s.targetingActions[tomeID]; ok {
		t.Fatal("did not expect Jayemdae Tome (CMC 4) to be targetable when X = 0")
	}

	s.exitTargetingMode()
	s.enterXChoosingMode([]interactive.ActionOption{action})
	s.selectXValue(2)
	if len(s.targetingActions) != 0 {
		t.Fatalf("expected no valid targets for X = 2, got %d", len(s.targetingActions))
	}
}

func TestStandardXSpellRetainsValidTargets(t *testing.T) {
	creatureID := uuid.New()
	action := interactive.ActionOption{
		Type:         interactive.ActionCastSpell,
		CardID:       uuid.New(),
		CardName:     "Fireball",
		NeedsX:       true,
		MaxXValue:    5,
		NeedsTarget:  true,
		TargetType:   mage.TargetCreature(),
		ValidTargets: []uuid.UUID{creatureID},
	}

	s := &DuelScreen{}
	s.enterXChoosingMode([]interactive.ActionOption{action})
	s.selectXValue(5)
	if _, ok := s.targetingActions[creatureID]; !ok {
		t.Fatal("expected creature to be targetable for standard X spell")
	}
}
