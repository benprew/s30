package screens

import (
	"testing"

	gameaudio "github.com/benprew/s30/game/audio"
	"github.com/benprew/s30/game/domain"
)

func TestQuestRewardContinuesForPointerClickOrKeyboard(t *testing.T) {
	tests := []struct {
		name    string
		clicked bool
		space   bool
		escape  bool
		want    bool
	}{
		{name: "no input"},
		{name: "pointer click", clicked: true, want: true},
		{name: "space", space: true, want: true},
		{name: "escape", escape: true, want: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := questRewardContinues(test.clicked, test.space, test.escape); got != test.want {
				t.Fatalf("questRewardContinues() = %t, want %t", got, test.want)
			}
		})
	}
}

func TestQuestRewardSound(t *testing.T) {
	if got := questRewardSound([]domain.DeckQuestReward{{Reward: domain.QuestReward{Gold: 100}}}); got != gameaudio.SFXReward {
		t.Fatalf("gold reward sound = %v, want reward", got)
	}
	if got := questRewardSound([]domain.DeckQuestReward{{Reward: domain.QuestReward{ManaLinks: 1}}}); got != gameaudio.SFXManalink {
		t.Fatalf("mana-link reward sound = %v, want manalink", got)
	}
}
