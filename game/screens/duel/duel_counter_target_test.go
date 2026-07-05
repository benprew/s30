package duel

import (
	"testing"

	mage "github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/google/uuid"
)

func newTestHuman() (*interactive.HumanPlayer, chan interactive.PriorityAction) {
	fromTUI := make(chan interactive.PriorityAction, 1)
	h := interactive.NewHumanPlayerWithChannels(
		"You",
		make(chan interactive.GameMsg, 1),
		fromTUI,
		make(chan interactive.ChoiceRequest, 1),
		make(chan interactive.ChoiceResponse, 1),
	)
	return h, fromTUI
}

func TestCounterAutoTargetsSingleOpponentSpell(t *testing.T) {
	human, fromTUI := newTestHuman()
	spellID := uuid.New()
	counterID := uuid.New()

	s := &DuelScreen{
		human: human,
		lastMsg: &interactive.GameMsg{
			State: &interactive.GameState{
				Opponent: interactive.PlayerState{Name: "Enemy"},
				StackItems: []interactive.StackItemState{
					{ID: spellID.String(), Name: "Bolt", Controller: "Enemy"},
				},
			},
		},
		cardActions: map[uuid.UUID][]interactive.ActionOption{
			counterID: {{
				Type:         interactive.ActionCastSpell,
				CardID:       counterID,
				CardName:     "Counterspell",
				NeedsTarget:  true,
				TargetType:   mage.TargetSpellOnStack(),
				ValidTargets: []uuid.UUID{spellID},
			}},
		},
	}

	s.performCardAction(counterID, "Counterspell")

	if s.targetingCardID != uuid.Nil {
		t.Fatal("should not enter targeting mode when only one opponent spell is on the stack")
	}

	select {
	case pa := <-fromTUI:
		if len(pa.Targets) != 1 || pa.Targets[0] != spellID {
			t.Fatalf("expected auto-target %v, got %v", spellID, pa.Targets)
		}
	default:
		t.Fatal("expected a cast action to be sent")
	}
}

func TestCounterPromptsWithMultipleOpponentSpells(t *testing.T) {
	human, _ := newTestHuman()
	spell1 := uuid.New()
	spell2 := uuid.New()
	counterID := uuid.New()

	s := &DuelScreen{
		human: human,
		lastMsg: &interactive.GameMsg{
			State: &interactive.GameState{
				Opponent: interactive.PlayerState{Name: "Enemy"},
				StackItems: []interactive.StackItemState{
					{ID: spell1.String(), Name: "Bolt", Controller: "Enemy"},
					{ID: spell2.String(), Name: "Growth", Controller: "Enemy"},
				},
			},
		},
		cardActions: map[uuid.UUID][]interactive.ActionOption{
			counterID: {{
				Type:         interactive.ActionCastSpell,
				CardID:       counterID,
				CardName:     "Counterspell",
				NeedsTarget:  true,
				TargetType:   mage.TargetSpellOnStack(),
				ValidTargets: []uuid.UUID{spell1, spell2},
			}},
		},
	}

	s.performCardAction(counterID, "Counterspell")

	if s.targetingCardID != counterID {
		t.Fatal("should enter targeting mode when multiple opponent spells are on the stack")
	}
}
