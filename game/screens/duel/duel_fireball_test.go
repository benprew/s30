package duel

import (
	"slices"
	"testing"
	"uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/interactive"
)

func TestFireballTargetingFirstThenXChooser(t *testing.T) {
	human, fromTUI := newTestHuman()
	fireballID := uuid.New()
	targetOpponent := uuid.New()
	targetCreature := uuid.New()
	targetCreature2 := uuid.New()

	action := interactive.ActionOption{
		Type:         interactive.ActionCastSpell,
		CardID:       fireballID,
		CardName:     "Fireball",
		NeedsX:       true,
		MaxXValue:    4, // 5 total mana: 1 base {R} + 4 mana for (X + extra targets)
		NeedsTarget:  true,
		TargetType:   mage.TargetDamageAnyNumberOfTargets(),
		ValidTargets: []uuid.UUID{targetOpponent, targetCreature, targetCreature2},
	}

	s := &DuelScreen{
		human: human,
		cardActions: map[uuid.UUID][]interactive.ActionOption{
			fireballID: {action},
		},
	}

	// 1. Clicking Fireball in hand enters targeting mode FIRST, not X-choosing mode
	s.performCardAction(fireballID, "Fireball")

	if s.targetingCardID != fireballID {
		t.Fatalf("expected targetingCardID %v, got %v", fireballID, s.targetingCardID)
	}
	if s.isChoosingX() {
		t.Fatal("Fireball must not enter X-choosing mode before targets are chosen")
	}

	// 2. Target bounds should be min 1, and max affordable targets = MaxXValue + 1 = 5
	minT, maxT := s.targetBounds()
	if minT != 1 || maxT != 5 {
		t.Fatalf("targetBounds = (%d, %d), want (1, 5)", minT, maxT)
	}

	// 3. Select 2 targets (Opponent and Creature 1)
	s.selectTarget(targetOpponent)
	s.selectTarget(targetCreature)

	if len(s.selectedTargetIDs) != 2 {
		t.Fatalf("expected 2 selected targets, got %d", len(s.selectedTargetIDs))
	}

	// 4. Submitting targeting mode (e.g. clicking Done button) transitions to X-choosing mode
	if !s.submitTargetingAction() {
		t.Fatal("submitTargetingAction should succeed when 2 targets selected")
	}

	// Targeting mode should be exited or completed
	s.exitTargetingMode()

	// 5. Must now be in X-choosing mode with targets stored
	if !s.isChoosingX() {
		t.Fatal("expected isChoosingX() to be true after targeting complete")
	}
	if len(s.xChosenTargets) != 2 || !slices.Equal(s.xChosenTargets, []uuid.UUID{targetOpponent, targetCreature}) {
		t.Fatalf("xChosenTargets = %v, want [%v %v]", s.xChosenTargets, targetOpponent, targetCreature)
	}

	// 6. MaxXValue must be adjusted for additional target cost:
	// 2 targets = 1 target beyond the first = 1 extra mana.
	// MaxXValue = 4 - 1 = 3.
	if s.xMaxValue != 3 {
		t.Fatalf("xMaxValue = %d, want 3", s.xMaxValue)
	}
	if s.xCurrentValue != 3 {
		t.Fatalf("xCurrentValue = %d, want default to 3", s.xCurrentValue)
	}

	// 7. Adjust X to 2
	s.decrementX()
	if s.xCurrentValue != 2 {
		t.Fatalf("xCurrentValue after decrement = %d, want 2", s.xCurrentValue)
	}

	// 8. Confirm X
	s.selectXValue(s.xCurrentValue)

	if s.isChoosingX() {
		t.Fatal("expected isChoosingX() to be false after confirmation")
	}

	// 9. Verify PriorityAction sent to engine with both targets and XValue = 2
	select {
	case pa := <-fromTUI:
		if pa.CardID != fireballID {
			t.Fatalf("expected CardID %v, got %v", fireballID, pa.CardID)
		}
		if pa.XValue != 2 {
			t.Fatalf("expected XValue 2, got %d", pa.XValue)
		}
		if len(pa.Targets) != 2 || !slices.Equal(pa.Targets, []uuid.UUID{targetOpponent, targetCreature}) {
			t.Fatalf("expected Targets [%v %v], got %v", targetOpponent, targetCreature, pa.Targets)
		}
	default:
		t.Fatal("expected PriorityAction to be sent on X confirmation")
	}
}

func TestFireballTargetLimitByAvailableMana(t *testing.T) {
	human, _ := newTestHuman()
	fireballID := uuid.New()
	t1 := uuid.New()
	t2 := uuid.New()
	t3 := uuid.New()

	// 2 total mana: 1 base {R} + 1 mana available (MaxXValue = 1).
	// With 1 extra mana, at most 2 targets can be chosen (1 extra target costs {1}, X=0).
	// A 3rd target would cost {2} extra, which exceeds available mana.
	action := interactive.ActionOption{
		Type:         interactive.ActionCastSpell,
		CardID:       fireballID,
		CardName:     "Fireball",
		NeedsX:       true,
		MaxXValue:    1,
		NeedsTarget:  true,
		TargetType:   mage.TargetDamageAnyNumberOfTargets(),
		ValidTargets: []uuid.UUID{t1, t2, t3},
	}

	s := &DuelScreen{
		human: human,
		cardActions: map[uuid.UUID][]interactive.ActionOption{
			fireballID: {action},
		},
	}

	s.performCardAction(fireballID, "Fireball")
	minT, maxT := s.targetBounds()
	if minT != 1 || maxT != 2 {
		t.Fatalf("targetBounds with MaxX=1 should be (1, 2), got (%d, %d)", minT, maxT)
	}

	s.selectTarget(t1)
	s.selectTarget(t2)
	// Attempt to select 3rd target beyond max affordable
	s.selectTarget(t3)

	if len(s.selectedTargetIDs) > 2 {
		t.Fatalf("player should not be able to select 3 targets with only 2 mana available, got %d", len(s.selectedTargetIDs))
	}
}

func TestFireballSingleTarget(t *testing.T) {
	human, fromTUI := newTestHuman()
	fireballID := uuid.New()
	targetOpponent := uuid.New()

	action := interactive.ActionOption{
		Type:         interactive.ActionCastSpell,
		CardID:       fireballID,
		CardName:     "Fireball",
		NeedsX:       true,
		MaxXValue:    5,
		NeedsTarget:  true,
		TargetType:   mage.TargetDamageAnyNumberOfTargets(),
		ValidTargets: []uuid.UUID{targetOpponent},
	}

	s := &DuelScreen{
		human: human,
		cardActions: map[uuid.UUID][]interactive.ActionOption{
			fireballID: {action},
		},
	}

	s.performCardAction(fireballID, "Fireball")
	s.selectTarget(targetOpponent)

	if !s.submitTargetingAction() {
		t.Fatal("submitTargetingAction failed for single target")
	}
	s.exitTargetingMode()

	if !s.isChoosingX() {
		t.Fatal("expected isChoosingX to be true")
	}
	// For 1 target, no additional cost applies, MaxX remains 5
	if s.xMaxValue != 5 {
		t.Fatalf("expected xMaxValue 5, got %d", s.xMaxValue)
	}

	s.selectXValue(s.xCurrentValue)

	select {
	case pa := <-fromTUI:
		if pa.XValue != 5 || len(pa.Targets) != 1 || pa.Targets[0] != targetOpponent {
			t.Fatalf("expected X=5 and single target %v, got X=%d targets=%v", targetOpponent, pa.XValue, pa.Targets)
		}
	default:
		t.Fatal("expected priority action to be sent")
	}
}

func TestFireballCancelDuringTargeting(t *testing.T) {
	human, fromTUI := newTestHuman()
	fireballID := uuid.New()
	targetOpponent := uuid.New()

	action := interactive.ActionOption{
		Type:         interactive.ActionCastSpell,
		CardID:       fireballID,
		CardName:     "Fireball",
		NeedsX:       true,
		MaxXValue:    5,
		NeedsTarget:  true,
		TargetType:   mage.TargetDamageAnyNumberOfTargets(),
		ValidTargets: []uuid.UUID{targetOpponent},
	}

	s := &DuelScreen{
		human: human,
		cardActions: map[uuid.UUID][]interactive.ActionOption{
			fireballID: {action},
		},
	}

	s.performCardAction(fireballID, "Fireball")
	s.selectTarget(targetOpponent)

	// User cancels via escape
	s.handleEscape() // clears selected targets
	if len(s.selectedTargetIDs) != 0 {
		t.Fatalf("expected selected targets to be cleared on escape, got %d", len(s.selectedTargetIDs))
	}

	s.handleEscape() // exits targeting mode
	if s.targetingCardID != uuid.Nil() {
		t.Fatal("expected targeting mode to be exited on second escape")
	}
	if s.isChoosingX() {
		t.Fatal("X choosing mode should not be active after cancel")
	}

	select {
	case pa := <-fromTUI:
		t.Fatalf("unexpected priority action sent after cancel: %v", pa)
	default:
	}
}

func TestFireballCancelDuringXChooser(t *testing.T) {
	human, fromTUI := newTestHuman()
	fireballID := uuid.New()
	targetOpponent := uuid.New()

	action := interactive.ActionOption{
		Type:         interactive.ActionCastSpell,
		CardID:       fireballID,
		CardName:     "Fireball",
		NeedsX:       true,
		MaxXValue:    5,
		NeedsTarget:  true,
		TargetType:   mage.TargetDamageAnyNumberOfTargets(),
		ValidTargets: []uuid.UUID{targetOpponent},
	}

	s := &DuelScreen{
		human: human,
		cardActions: map[uuid.UUID][]interactive.ActionOption{
			fireballID: {action},
		},
	}

	s.performCardAction(fireballID, "Fireball")
	s.selectTarget(targetOpponent)
	s.submitTargetingAction()
	s.exitTargetingMode()

	if !s.isChoosingX() {
		t.Fatal("expected isChoosingX to be true")
	}

	// User cancels X chooser
	s.handleEscape()

	if s.isChoosingX() {
		t.Fatal("expected isChoosingX to be false after escape")
	}
	if len(s.xChosenTargets) != 0 {
		t.Fatalf("expected xChosenTargets to be cleared, got %v", s.xChosenTargets)
	}

	select {
	case pa := <-fromTUI:
		t.Fatalf("unexpected priority action sent after cancel: %v", pa)
	default:
	}
}
