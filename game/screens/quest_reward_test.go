package screens

import (
	"image"
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

func TestQuestRewardsPaginateMiniCards(t *testing.T) {
	card := domain.CARDS[0]
	domain.CacheCardImage(card.CardID(), image.NewRGBA(image.Rect(0, 0, 245, 342)))
	t.Cleanup(domain.ClearCardImageCache)
	s := NewQuestRewardScreen([]domain.DeckQuestReward{{Quest: &domain.Quest{Title: "Reward"}, Cards: []*domain.Card{card, card, card, card}}}, nil, nil, nil)
	if len(s.rewards) != 2 {
		t.Fatalf("pages = %d, want 2", len(s.rewards))
	}
	for _, page := range s.rewards {
		for _, img := range page.cardImgs {
			if img.Bounds().Size() != image.Pt(183, 256) {
				t.Fatal("wrong reward card size")
			}
		}
	}
	if s.panelH < 450 {
		t.Fatal("reward panel is too short")
	}
}
