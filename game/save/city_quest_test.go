package save

import (
	"testing"

	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/world"
)

func TestSavePreservesCityQuestAvailability(t *testing.T) {
	city := &domain.City{Name: "Home", QuestAvailableDay: 15, WisemanBoon: domain.BoonBonusLife, BoonGranted: true}
	other := &domain.City{Name: "Away", ProposedQuest: &domain.Quest{
		Type: domain.QuestTypeDelivery, Title: "Take letter to Home", TargetCity: city,
		DaysRemaining: 20, Reward: domain.QuestReward{ManaLinks: 1},
	}}
	player := &domain.Player{Days: 12, ActiveQuests: []*domain.Quest{{
		Type: domain.QuestTypeDefeatEnemy, OriginCity: "Home", EnemyName: "Goblin", DaysRemaining: 10,
	}}}
	level := &world.Level{Player: player, Difficulty: domain.DifficultyHard, W: 2, H: 1,
		Tiles: [][]*world.Tile{{{City: city}, {City: other}}}}
	data, err := serializeSave(level)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := deserializeSave(data)
	if err != nil {
		t.Fatal(err)
	}
	player = loaded.World.Player
	city = loaded.World.Tiles[0][0].City
	other = loaded.World.Tiles[0][1].City
	if city.QuestAvailableDay != 15 || !city.BoonGranted || city.CanOfferQuest(player) {
		t.Fatal("save lost the city cooldown or gift state")
	}
	player.Days = 15
	if player.ActiveQuests[0].OriginCity != "Home" || city.CanOfferQuest(player) {
		t.Fatal("save lost the outstanding quest's city")
	}
	player.ActiveQuests = nil
	if !city.CanOfferQuest(player) {
		t.Fatal("city must offer a quest after its saved cooldown and outstanding quest end")
	}
	if other.ProposedQuest == nil || other.ProposedQuest.Reward.ManaLinks != 1 || other.ProposedQuest.TargetCity.Name != "Home" {
		t.Fatal("save lost the unaccepted quest offer")
	}
}
