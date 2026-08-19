package duel

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/benprew/mage-go/pkg/mage/interactive/ai"
	"github.com/google/uuid"
)

type autoplayStrategy struct {
	priority  interactive.PriorityAction
	attackers []uuid.UUID
	blockers  []mage.BlockAssignment
}

func (s *autoplayStrategy) PriorityAction(mage.Player, *mage.Game, int, bool) interactive.PriorityAction {
	return s.priority
}

func (s *autoplayStrategy) Attackers(mage.Player, *mage.Game) []uuid.UUID {
	return s.attackers
}

func (s *autoplayStrategy) Blockers(mage.Player, *mage.Game) []mage.BlockAssignment {
	return s.blockers
}

var _ ai.AIStrategy = (*autoplayStrategy)(nil)

func TestAutoPlayActionUsesStrategyForPrompt(t *testing.T) {
	human := interactive.NewHumanPlayer("You")
	opponent := mage.NewBasePlayer("Opponent")
	game := mage.NewGame(human, opponent)
	cardID := uuid.New()
	strategy := &autoplayStrategy{priority: interactive.PriorityAction{
		Type:   interactive.ActionCastSpell,
		CardID: cardID,
	}}
	s := &DuelScreen{
		human:        human,
		game:         game,
		autoStrategy: strategy,
		lastMsg: &interactive.GameMsg{
			Prompt: interactive.PromptMainPhaseAction,
		},
	}

	action, ok := s.autoPlayAction()
	if !ok {
		t.Fatal("autoPlayAction() did not return an action")
	}
	if action.Type != interactive.ActionCastSpell || action.CardID != cardID {
		t.Fatalf("autoPlayAction() = %#v, want strategy action", action)
	}
}

func TestAutoPlayActionDeclaresAttackers(t *testing.T) {
	human := interactive.NewHumanPlayer("You")
	game := mage.NewGame(human, mage.NewBasePlayer("Opponent"))
	attackerID := uuid.New()
	s := &DuelScreen{
		human:        human,
		game:         game,
		autoStrategy: &autoplayStrategy{attackers: []uuid.UUID{attackerID}},
		lastMsg: &interactive.GameMsg{
			Prompt: interactive.PromptDeclareAttackers,
		},
	}

	action, ok := s.autoPlayAction()
	if !ok || action.Type != interactive.ActionSelectAttackers || len(action.Attackers) != 1 || action.Attackers[0] != attackerID {
		t.Fatalf("autoPlayAction() = %#v, %v", action, ok)
	}
}

func TestAutoChoiceResponseSelectsRequestedCards(t *testing.T) {
	first := uuid.New()
	second := uuid.New()
	third := uuid.New()
	req := interactive.ChoiceRequest{
		Type:   interactive.ChoiceCardsFromHand,
		Amount: 2,
		Options: []interactive.ChoiceOption{
			{ID: first},
			{ID: second},
			{ID: third},
		},
	}

	response := autoChoiceResponse(req)
	if len(response.SelectedIDs) != 2 || response.SelectedIDs[0] != first || response.SelectedIDs[1] != second {
		t.Fatalf("autoChoiceResponse() selected %v, want first two cards", response.SelectedIDs)
	}
}
