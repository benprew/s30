package duel

import (
	"testing"
	"uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/benprew/mage-go/pkg/mage/interactive/ai"
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

func TestAutoChoiceDefersMayToMageAI(t *testing.T) {
	responses := make(chan interactive.ChoiceResponse, 1)
	human := interactive.NewHumanPlayerWithChannels("You", make(chan interactive.GameMsg, 1), make(chan interactive.PriorityAction, 1), make(chan interactive.ChoiceRequest, 1), responses)
	game := mage.NewGame(human, mage.NewBasePlayer("Opponent"))
	for _, tc := range []struct {
		reason string
		want   bool
	}{
		{"draw a card", true}, {"pay {2} to mill yourself", false},
	} {
		s := &DuelScreen{human: human, game: game, choiceRequest: &interactive.ChoiceRequest{
			Type: interactive.ChoiceMay, Reason: tc.reason, Options: []interactive.ChoiceOption{{Label: "Yes"}, {Label: "No"}},
		}}
		s.respondToAutoChoice()
		if s.choiceRequest != nil {
			t.Fatal("choice was not sent")
		}
		response := <-responses
		if response.Accepted != tc.want {
			t.Errorf("%s: got %v, want %v", tc.reason, response.Accepted, tc.want)
		}
	}
}

func TestEnableAutoPlayUsesLondonPolicyAfterPriorMulligans(t *testing.T) {
	s, human := newMulliganScreen(t)
	human.SetHand(nil)
	for i := range 7 {
		card := mage.NewCreature("Bear", "{1}{G}", 2, 2)
		if i < 2 {
			card = mage.NewLand("Forest")
		}
		card.SetOwner(human.PlayerID())
		human.AddToHand(card)
	}
	s.mulliganCount = 2
	s.mulliganSelected[human.Hand()[0].ID()] = true
	s.EnableAutoPlay()
	if s.inMulligan || len(human.Hand()) != 5 {
		t.Fatalf("mulligan=%v hand=%d", s.inMulligan, len(human.Hand()))
	}
	lands := 0
	for _, card := range human.Hand() {
		if card.HasType(core.TypeLand) {
			lands++
		}
	}
	if lands != 2 {
		t.Fatalf("kept %d lands, want 2", lands)
	}
}
