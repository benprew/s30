package duel

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/google/uuid"
)

type exactTwoTarget struct {
	mage.Target
}

func (exactTwoTarget) Min() int { return 2 }
func (exactTwoTarget) Max() int { return 2 }

func TestTargetingExactTwoRequiresAndSubmitsTwoDistinctTargets(t *testing.T) {
	cardID := uuid.New()
	first := uuid.New()
	second := uuid.New()
	fromTUI := make(chan interactive.PriorityAction, 1)
	action := interactive.ActionOption{
		Type:         interactive.ActionCastSpell,
		CardID:       cardID,
		CardName:     "Ashes to Ashes",
		NeedsTarget:  true,
		TargetType:   exactTwoTarget{Target: mage.TargetCreature()},
		ValidTargets: []uuid.UUID{first, second},
	}
	human := interactive.NewHumanPlayerWithChannels("Test",
		make(chan interactive.GameMsg, 1),
		fromTUI,
		make(chan interactive.ChoiceRequest, 1),
		make(chan interactive.ChoiceResponse, 1),
	)
	s := &DuelScreen{human: human}

	s.enterTargetingMode(cardID, action.CardName, []interactive.ActionOption{action})
	s.selectTarget(first)
	if s.targetSelectionComplete() {
		t.Fatal("one target must not complete an exact-two selection")
	}
	if s.submitTargetingAction() {
		t.Fatal("incomplete target selection must not be submitted")
	}

	s.selectTarget(first)
	if len(s.selectedTargetIDs) != 0 {
		t.Fatal("selecting an already selected target should deselect it")
	}

	s.selectTarget(first)
	s.selectTarget(second)
	if !s.targetSelectionComplete() {
		t.Fatal("two targets should complete an exact-two selection")
	}
	if !s.submitTargetingAction() {
		t.Fatal("complete target selection should be submitted")
	}

	select {
	case got := <-fromTUI:
		if len(got.Targets) != 2 || got.Targets[0] != first || got.Targets[1] != second {
			t.Fatalf("submitted targets = %v, want [%v %v]", got.Targets, first, second)
		}
	default:
		t.Fatal("expected targeted action to be sent")
	}
}

func TestSingleTargetSelectionStillReplacesPreviousTarget(t *testing.T) {
	first := uuid.New()
	second := uuid.New()
	action := interactive.ActionOption{
		NeedsTarget:  true,
		TargetType:   mage.TargetCreature(),
		ValidTargets: []uuid.UUID{first, second},
	}
	s := &DuelScreen{}

	s.enterTargetingMode(uuid.New(), "Test", []interactive.ActionOption{action})
	s.selectTarget(first)
	s.selectTarget(second)

	if len(s.selectedTargetIDs) != 1 || s.selectedTargetIDs[0] != second {
		t.Fatalf("selected targets = %v, want [%v]", s.selectedTargetIDs, second)
	}
}

func TestTargetingPromptDescribesExactTargetCount(t *testing.T) {
	s := &DuelScreen{
		targetingAction: interactive.ActionOption{
			TargetType: exactTwoTarget{Target: mage.TargetCreature()},
		},
	}

	if got := s.targetingPrompt("Ashes to Ashes"); got != "Choose 2 targets for Ashes to Ashes" {
		t.Fatalf("targeting prompt = %q", got)
	}
}
