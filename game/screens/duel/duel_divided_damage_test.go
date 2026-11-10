package duel

import (
	"slices"
	"testing"
	"uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/interactive"
)

func TestPyrotechnicsMultiTargetingAllowsUpTo4Targets(t *testing.T) {
	human, fromTUI := newTestHuman()
	pyroID := uuid.New()
	tOpponent := uuid.New()
	tCreature1 := uuid.New()
	tCreature2 := uuid.New()
	tCreature3 := uuid.New()

	action := interactive.ActionOption{
		Type:         interactive.ActionCastSpell,
		CardID:       pyroID,
		CardName:     "Pyrotechnics",
		NeedsTarget:  true,
		TargetType:   mage.TargetDamageDivided(4),
		ValidTargets: []uuid.UUID{tOpponent, tCreature1, tCreature2, tCreature3},
	}

	s := &DuelScreen{
		human: human,
		cardActions: map[uuid.UUID][]interactive.ActionOption{
			pyroID: {action},
		},
	}

	// 1. Enter targeting mode
	s.performCardAction(pyroID, "Pyrotechnics")
	if s.targetingCardID != pyroID {
		t.Fatalf("expected targetingCardID %v, got %v", pyroID, s.targetingCardID)
	}

	// 2. Bounds should allow 1 to 4 targets
	minT, maxT := s.targetBounds()
	if minT != 1 || maxT != 4 {
		t.Fatalf("targetBounds = (%d, %d), want (1, 4)", minT, maxT)
	}

	// 3. Select 3 targets
	s.selectTarget(tOpponent)
	s.selectTarget(tCreature1)
	s.selectTarget(tCreature2)

	if len(s.selectedTargetIDs) != 3 {
		t.Fatalf("expected 3 selected targets, got %d", len(s.selectedTargetIDs))
	}

	// 4. Submit targeting action
	if !s.submitTargetingAction() {
		t.Fatal("expected submitTargetingAction to succeed")
	}

	select {
	case pa := <-fromTUI:
		if pa.Type != interactive.ActionCastSpell {
			t.Fatalf("pa.Type = %v, want ActionCastSpell", pa.Type)
		}
		if len(pa.Targets) != 3 || !slices.Contains(pa.Targets, tOpponent) || !slices.Contains(pa.Targets, tCreature1) || !slices.Contains(pa.Targets, tCreature2) {
			t.Fatalf("unexpected targets: %v", pa.Targets)
		}
	default:
		t.Fatal("expected priority action sent")
	}
}

func TestDividedDamagePromptAndControls(t *testing.T) {
	human, fromTUI := newTestHuman()
	tOpponent := uuid.New()
	tCreature := uuid.New()

	s := &DuelScreen{
		human: human,
		opponent: &duelPlayer{
			name: "Opponent",
		},
		self: &duelPlayer{
			name: "You",
		},
	}

	msg := &interactive.GameMsg{
		Prompt: interactive.PromptAssignDamage,
		Options: []interactive.ActionOption{{
			CardName:     "Pyrotechnics",
			MaxXValue:    4,
			ValidTargets: []uuid.UUID{tOpponent, tCreature},
		}},
		State: &interactive.GameState{
			Opponent: interactive.PlayerState{
				ID: tOpponent,
				Battlefield: []interactive.PermanentState{
					{ID: tCreature, Name: "Grizzly Bears", Toughness: 2},
				},
			},
		},
	}

	s.lastMsg = msg
	s.refreshDamageAssignmentPrompt(msg)

	if s.damageTotal != 4 {
		t.Fatalf("damageTotal = %d, want 4", s.damageTotal)
	}

	// Under CR 601.2d, each must receive at least 1, total 4
	if s.damageAssignment[tOpponent] < 1 || s.damageAssignment[tCreature] < 1 {
		t.Fatalf("each target must receive at least 1 damage: %v", s.damageAssignment)
	}
	if s.damageAssignment[tOpponent]+s.damageAssignment[tCreature] != 4 {
		t.Fatalf("sum of damage must be 4: %v", s.damageAssignment)
	}

	// Increase creature damage (shift from opponent to creature)
	creatureBefore := s.damageAssignment[tCreature]
	oppBefore := s.damageAssignment[tOpponent]
	if oppBefore > 1 {
		s.increaseAssignedDamage(tCreature)
		if s.damageAssignment[tCreature] != creatureBefore+1 || s.damageAssignment[tOpponent] != oppBefore-1 {
			t.Fatalf("damage after increase: %v", s.damageAssignment)
		}
	}

	// Player clicks Done
	s.submitPendingAndPass()

	select {
	case pa := <-fromTUI:
		if pa.Type != interactive.ActionAssignDamage {
			t.Fatalf("pa.Type = %v, want ActionAssignDamage", pa.Type)
		}
		sum := 0
		for id, amt := range pa.Damage {
			if amt < 1 {
				t.Fatalf("target %v received %d < 1 damage", id, amt)
			}
			sum += amt
		}
		if sum != 4 {
			t.Fatalf("total damage assigned = %d, want 4", sum)
		}
	default:
		t.Fatal("expected priority action sent")
	}
}

func TestDividedDamagePlayerDamageControlBoundsClick(t *testing.T) {
	tOpponent := uuid.New()
	tCreature := uuid.New()

	s := &DuelScreen{
		damageAssignment: map[uuid.UUID]int{
			tOpponent: 3,
			tCreature: 1,
		},
		opponent: &duelPlayer{
			name: "Opponent",
		},
		self: &duelPlayer{
			name: "You",
		},
		lastMsg: &interactive.GameMsg{
			Prompt: interactive.PromptAssignDamage,
			State: &interactive.GameState{
				Opponent: interactive.PlayerState{
					ID: tOpponent,
				},
			},
		},
	}

	pos := s.playerDamageControlPos(s.opponent)
	minus, plus := damageControlBounds(pos)

	// Click minus on opponent
	clickedID, delta := s.damageControlAt(minus.Min.X+2, minus.Min.Y+2)
	if clickedID != tOpponent || delta != -1 {
		t.Fatalf("expected opponent minus, got id=%v delta=%d", clickedID, delta)
	}

	// Click plus on opponent
	clickedID, delta = s.damageControlAt(plus.Min.X+2, plus.Min.Y+2)
	if clickedID != tOpponent || delta != 1 {
		t.Fatalf("expected opponent plus, got id=%v delta=%d", clickedID, delta)
	}
}
