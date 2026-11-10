package duel

import (
	"testing"
	"uuid"

	"github.com/benprew/mage-go/pkg/mage/interactive"
)

func TestSuggestedDamageAssignmentKillsMostBlockers(t *testing.T) {
	blockerA := interactive.PermanentState{ID: uuid.New(), Toughness: 3, Damage: 1}
	blockerB := interactive.PermanentState{ID: uuid.New(), Toughness: 2}
	blockerC := interactive.PermanentState{ID: uuid.New(), Toughness: 4}

	got := suggestedDamageAssignment([]interactive.PermanentState{blockerA, blockerB, blockerC}, 5)

	if got[blockerA.ID] != 2 || got[blockerB.ID] != 2 || got[blockerC.ID] != 1 {
		t.Fatalf("assignment = %v, want 2/2/1", got)
	}
}

func TestSuggestedDamageAssignmentIncludesZeroDamageBlockers(t *testing.T) {
	pumped := interactive.PermanentState{ID: uuid.New(), Toughness: 3}
	unpumped := interactive.PermanentState{ID: uuid.New(), Toughness: 1}

	got := suggestedDamageAssignment([]interactive.PermanentState{pumped, unpumped}, 2)

	if _, ok := got[unpumped.ID]; !ok {
		t.Fatalf("assignment = %v, want unpumped blocker present so it has damage controls", got)
	}
	if got[pumped.ID] != 2 || got[unpumped.ID] != 0 {
		t.Fatalf("assignment = %v, want pumped=2 unpumped=0", got)
	}
}

func TestDamageAssignmentControlsMoveDamageBetweenBlockers(t *testing.T) {
	blockerA := uuid.New()
	blockerB := uuid.New()
	s := &DuelScreen{
		damageAssignment: map[uuid.UUID]int{
			blockerA: 2,
			blockerB: 1,
		},
	}

	s.increaseAssignedDamage(blockerB)
	if s.damageAssignment[blockerA] != 1 || s.damageAssignment[blockerB] != 2 {
		t.Fatalf("after increase = %v, want A=1 B=2", s.damageAssignment)
	}

	s.decreaseAssignedDamage(blockerB)
	if s.damageAssignment[blockerA] != 2 || s.damageAssignment[blockerB] != 1 {
		t.Fatalf("after decrease = %v, want A=2 B=1", s.damageAssignment)
	}
}

func TestSubmitDamageAssignmentSendsAction(t *testing.T) {
	fromTUI := make(chan interactive.PriorityAction, 1)
	human := interactive.NewHumanPlayerWithChannels("You", make(chan interactive.GameMsg, 1), fromTUI, nil, nil)
	blockerA := uuid.New()
	blockerB := uuid.New()
	s := &DuelScreen{
		human: human,
		lastMsg: &interactive.GameMsg{
			Prompt: interactive.PromptAssignCombatDamage,
		},
		damageAssignment: map[uuid.UUID]int{
			blockerA: 2,
			blockerB: 1,
		},
	}

	s.submitPendingAndPass()

	action := <-fromTUI
	if action.Type != interactive.ActionAssignCombatDamage {
		t.Fatalf("action.Type = %v, want ActionAssignCombatDamage", action.Type)
	}
	if action.Damage[blockerA] != 2 || action.Damage[blockerB] != 1 {
		t.Fatalf("damage = %v, want A=2 B=1", action.Damage)
	}
}

func TestDamageAssignmentOrderPutsLethalBlockersFirst(t *testing.T) {
	blockerA := interactive.PermanentState{ID: uuid.New(), Toughness: 3}
	blockerB := interactive.PermanentState{ID: uuid.New(), Toughness: 2}
	blockerC := interactive.PermanentState{ID: uuid.New(), Toughness: 4}
	s := &DuelScreen{
		lastMsg: &interactive.GameMsg{
			State: &interactive.GameState{
				Opponent: interactive.PlayerState{
					Battlefield: []interactive.PermanentState{blockerA, blockerB, blockerC},
				},
			},
			Options: []interactive.ActionOption{{
				ValidTargets: []uuid.UUID{blockerA.ID, blockerB.ID, blockerC.ID},
			}},
		},
		damageAssignment: map[uuid.UUID]int{
			blockerA.ID: 1,
			blockerB.ID: 2,
			blockerC.ID: 2,
		},
	}

	order := s.damageAssignmentOrder()
	if order[0] != blockerB.ID {
		t.Fatalf("order[0] = %v, want lethal blocker %v first", order[0], blockerB.ID)
	}
}

func TestSuggestedDividedDamageAssignmentRequiresAtLeastOnePerTarget(t *testing.T) {
	t1 := uuid.New()
	t2 := uuid.New()
	s := &DuelScreen{}

	got := suggestedDividedDamageAssignment([]uuid.UUID{t1, t2}, 4, s)
	if got[t1] < 1 || got[t2] < 1 {
		t.Fatalf("expected each target to have at least 1 damage, got %v", got)
	}
	if got[t1]+got[t2] != 4 {
		t.Fatalf("expected sum 4, got %v", got)
	}
}

func TestDividedDamageControlsCannotDropBelowOne(t *testing.T) {
	t1 := uuid.New()
	t2 := uuid.New()
	s := &DuelScreen{
		lastMsg: &interactive.GameMsg{
			Prompt: interactive.PromptAssignDamage,
		},
		damageAssignment: map[uuid.UUID]int{
			t1: 3,
			t2: 1,
		},
	}

	// Decreasing t2 when it has 1 should be a no-op under CR 601.2d
	s.decreaseAssignedDamage(t2)
	if s.damageAssignment[t2] != 1 || s.damageAssignment[t1] != 3 {
		t.Fatalf("target dropped below 1: %v", s.damageAssignment)
	}

	// Decreasing t1 from 3 to 2 should increase t2 from 1 to 2
	s.decreaseAssignedDamage(t1)
	if s.damageAssignment[t1] != 2 || s.damageAssignment[t2] != 2 {
		t.Fatalf("expected 2/2, got %v", s.damageAssignment)
	}
}

func TestSubmitDividedDamageAssignmentSendsAction(t *testing.T) {
	fromTUI := make(chan interactive.PriorityAction, 1)
	human := interactive.NewHumanPlayerWithChannels("You", make(chan interactive.GameMsg, 1), fromTUI, nil, nil)
	t1 := uuid.New()
	t2 := uuid.New()
	s := &DuelScreen{
		human: human,
		lastMsg: &interactive.GameMsg{
			Prompt: interactive.PromptAssignDamage,
		},
		damageAssignment: map[uuid.UUID]int{
			t1: 2,
			t2: 2,
		},
	}

	s.submitPendingAndPass()

	action := <-fromTUI
	if action.Type != interactive.ActionAssignDamage {
		t.Fatalf("action.Type = %v, want ActionAssignDamage", action.Type)
	}
	if action.Damage[t1] != 2 || action.Damage[t2] != 2 {
		t.Fatalf("damage = %v, want t1=2 t2=2", action.Damage)
	}
}
