package duel

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/interactive"
)

// The engine reads a number choice as SelectedIndex + minimum, so the picked
// option must travel as an index. Sending it as an ID leaves the index at 0,
// and Tetravus never removes a counter no matter which number is clicked.
func TestRespondToChoiceNumberSendsSelectedIndex(t *testing.T) {
	responses := make(chan interactive.ChoiceResponse, 1)
	human := interactive.NewHumanPlayerWithChannels(
		"You",
		make(chan interactive.GameMsg, 1),
		make(chan interactive.PriorityAction, 1),
		make(chan interactive.ChoiceRequest, 1),
		responses,
	)
	s := &DuelScreen{
		human: human,
		choiceRequest: &interactive.ChoiceRequest{
			Type:   interactive.ChoiceNumber,
			Reason: "Remove how many +1/+1 counters from Tetravus?",
			Options: []interactive.ChoiceOption{
				{Label: "0"}, {Label: "1"}, {Label: "2"}, {Label: "3"},
			},
		},
	}

	s.respondToChoice(2)

	select {
	case resp := <-responses:
		if resp.SelectedIndex != 2 {
			t.Fatalf("SelectedIndex = %d, want 2", resp.SelectedIndex)
		}
	default:
		t.Fatal("respondToChoice sent no response")
	}
}
