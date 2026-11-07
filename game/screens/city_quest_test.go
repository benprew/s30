package screens

import (
	"testing"

	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/world"
)

func TestCityQuestRepeatsAfterCooldownAndRedemption(t *testing.T) {
	disableDeckQuestOffers(t)
	for _, difficulty := range []domain.Difficulty{domain.DifficultyEasy, domain.DifficultyMedium, domain.DifficultyHard, domain.DifficultyExpert} {
		t.Run(domain.DifficultyToString(difficulty), func(t *testing.T) {
			city := &domain.City{Name: "Home", WisemanBoon: domain.BoonBonusLife, BoonGranted: true}
			player := &domain.Player{Days: 10}
			level := &world.Level{Player: player, Difficulty: difficulty, W: 2, H: 1,
				Tiles: [][]*world.Tile{{{City: city}, {City: &domain.City{Name: "Away"}}}}}
			quest := newDeliveryQuest(&domain.City{Name: "Away"}, domain.QuestReward{Gold: 10})
			city.ProposedQuest = quest
			s := &WisemanScreen{City: city, Player: player, Level: level}
			s.determineState()
			if s.ProposedQuest != quest {
				t.Fatal("a spent gift must not block the city quest")
			}
			s.acceptQuest()
			readyDay := 10 + 3 + int(difficulty)
			if city.QuestAvailableDay != readyDay || quest.OriginCity != city.Name || city.ProposedQuest != nil {
				t.Fatalf("acceptance did not record the origin and cooldown: city=%+v quest=%+v", city, quest)
			}
			s = &WisemanScreen{City: city, Player: player, Level: level}
			s.determineState()
			if s.State != WisemanStateStory || s.ProposedQuest != nil {
				t.Fatal("an outstanding quest must block another quest from its city")
			}

			player.Days = readyDay - 1
			player.RedeemFulfilledQuests(&domain.City{Name: "Away"})
			s.determineState()
			if s.State != WisemanStateStory || city.CanOfferQuest(player) {
				t.Fatal("early completion must not end the cooldown")
			}
			player.Days = readyDay
			s.determineState()
			if s.State != WisemanStateOffer || s.ProposedQuest == nil || s.ProposedQuest == quest {
				t.Fatal("expected a new city quest after redemption and cooldown")
			}
			offer := s.ProposedQuest
			reward := offer.Reward
			NewCityScreen(city, player, level)
			s = &WisemanScreen{City: city, Player: player, Level: level}
			s.determineState()
			if s.ProposedQuest != offer || s.ProposedQuest.Reward != reward {
				t.Fatal("returning to a city must preserve its unaccepted offer")
			}
		})
	}
}

func TestCityQuestButtonFollowsAvailability(t *testing.T) {
	city := &domain.City{Name: "Home", WisemanBoon: domain.BoonBonusLife, BoonGranted: true, QuestAvailableDay: 5}
	player := &domain.Player{Days: 4}
	for _, tc := range []struct {
		day  int
		want string
	}{
		{day: 4, want: "Talk to Wiseman"},
		{day: 5, want: "Begin Quest"},
	} {
		player.Days = tc.day
		s := NewCityScreen(city, player, nil)
		found := false
		for _, button := range s.Buttons {
			if button.ID == "quest" {
				found = true
				if button.ButtonText.Text != tc.want {
					t.Fatalf("day %d: quest button = %q, want %q", tc.day, button.ButtonText.Text, tc.want)
				}
			}
		}
		if !found {
			t.Fatal("city has no quest button")
		}
	}
}

func TestCityQuestAcceptanceAtCapacityPreservesOffer(t *testing.T) {
	quest := newDeliveryQuest(&domain.City{Name: "Away"}, domain.QuestReward{Gold: 10})
	city := &domain.City{Name: "Home", ProposedQuest: quest}
	player := &domain.Player{ActiveQuests: make([]*domain.Quest, domain.MaxActiveQuests)}
	s := &WisemanScreen{City: city, Player: player, ProposedQuest: quest}
	s.acceptQuest()
	if city.ProposedQuest != quest || city.QuestAvailableDay != 0 || quest.OriginCity != "" {
		t.Fatal("failed acceptance must preserve the offer without starting its cooldown")
	}
}

func TestCityQuestExpiryKeepsCooldown(t *testing.T) {
	city := &domain.City{Name: "Home", QuestAvailableDay: 5}
	player := &domain.Player{Days: 4, ActiveQuests: []*domain.Quest{{OriginCity: "Home", DaysRemaining: 0}}}
	if player.ExpireQuests() != 1 || city.CanOfferQuest(player) {
		t.Fatal("expiry must remove the quest without ending its city cooldown")
	}
	player.Days = 5
	if !city.CanOfferQuest(player) {
		t.Fatal("expiry must not permanently block city quests")
	}
}

func TestLegacyCityQuestBoonBecomesSeparateGift(t *testing.T) {
	for _, spent := range []bool{false, true} {
		city := &domain.City{Name: "Home", WisemanBoon: domain.BoonQuest, BoonGranted: spent}
		offer := newDeliveryQuest(&domain.City{Name: "Away"}, domain.QuestReward{Gold: 10})
		city.ProposedQuest = offer
		player := &domain.Player{}
		NewCityScreen(city, player, nil)
		if city.WisemanBoon == domain.BoonQuest || city.BoonGranted || city.ProposedQuest != offer || !city.CanOfferQuest(player) {
			t.Fatalf("legacy quest boon did not become an independent gift: %+v", city)
		}
	}
}
