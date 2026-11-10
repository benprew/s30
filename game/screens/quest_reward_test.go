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
	owned := domain.NewOwnedCard(card)
	owned.ArtworkID = card.DefaultArtwork().ID
	domain.CacheCardImage(card.DefaultArtwork().ID, image.NewRGBA(image.Rect(0, 0, 245, 342)))
	t.Cleanup(domain.ClearCardImageCache)
	s := NewQuestRewardScreen([]domain.DeckQuestReward{{Quest: &domain.Quest{Title: "Reward"}, Cards: []*domain.OwnedCard{owned, owned, owned, owned}}}, nil, nil, nil)
	if len(s.rewards) != 2 {
		t.Fatalf("pages = %d, want 2", len(s.rewards))
	}
	for _, page := range s.rewards {
		for i := range page.cards {
			img, err := page.cardImage(i)
			if err != nil {
				t.Fatal(err)
			}
			if img.Bounds().Size() != image.Pt(183, 256) {
				t.Fatal("wrong reward card size")
			}
		}
	}
	if s.panelH < 450 {
		t.Fatal("reward panel is too short")
	}
}

func TestQuestRewardRefreshesArtAcrossPages(t *testing.T) {
	card := *domain.CARDS[0]
	card.Artworks = []domain.Artwork{{ID: "quest-default", CardName: card.Name()}, {ID: "quest-selected", CardName: card.Name()}}
	owned := &domain.OwnedCard{Card: &card, ArtworkID: "quest-selected"}
	domain.CacheCardImage(owned.ArtworkID, image.NewRGBA(image.Rect(0, 0, 245, 342)))
	t.Cleanup(domain.ClearCardImageCache)
	s := NewQuestRewardScreen([]domain.DeckQuestReward{{Quest: &domain.Quest{Title: "Reward"}, Cards: []*domain.OwnedCard{owned, owned, owned, owned}}}, nil, nil, nil)
	old, err := s.rewards[0].cardImage(0)
	if err != nil {
		t.Fatal(err)
	}
	domain.CacheCardImage(owned.ArtworkID, image.NewRGBA(image.Rect(0, 0, 245, 342)))
	want, _ := owned.CardImage(domain.CardViewFullMini)
	if want == old {
		t.Fatal("test did not replace the source image")
	}
	for _, page := range s.rewards {
		for i := range page.cards {
			img, err := page.cardImage(i)
			if err != nil {
				t.Fatal(err)
			}
			if img != want {
				t.Fatal("reward retained old art or lost the selected artwork")
			}
		}
	}
}
