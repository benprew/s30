package domain

import (
	"testing"
)

func TestCityCanOfferQuest(t *testing.T) {
	for _, tc := range []struct {
		name   string
		city   City
		player Player
		want   bool
	}{
		{name: "new city", city: City{Name: "Home"}, want: true},
		{name: "spent gift", city: City{Name: "Home", WisemanBoon: BoonBonusLife, BoonGranted: true}, want: true},
		{name: "cooldown", city: City{Name: "Home", QuestAvailableDay: 5}, player: Player{Days: 4}},
		{name: "cooldown complete", city: City{Name: "Home", QuestAvailableDay: 5}, player: Player{Days: 5}, want: true},
		{name: "active quest", city: City{Name: "Home"}, player: Player{ActiveQuests: []*Quest{{OriginCity: "Home"}}}},
		{name: "unclaimed reward", city: City{Name: "Home"}, player: Player{ActiveQuests: []*Quest{{OriginCity: "Home", IsCompleted: true}}}},
		{name: "other city quest", city: City{Name: "Home"}, player: Player{ActiveQuests: []*Quest{{OriginCity: "Away"}}}, want: true},
		{name: "full slots", city: City{Name: "Home"}, player: Player{ActiveQuests: make([]*Quest, MaxActiveQuests)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.city.CanOfferQuest(&tc.player); got != tc.want {
				t.Fatalf("CanOfferQuest = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCityTierEnum(t *testing.T) {
	tests := []struct {
		tier     CityTier
		expected string
	}{
		{TierHamlet, "Hamlet"},
		{TierTown, "Town"},
		{TierCapital, "Capital"},
		{CityTier(99), "Unknown"},
	}

	for _, test := range tests {
		result := test.tier.String()
		if result != test.expected {
			t.Errorf("Expected %s, got %s", test.expected, result)
		}
	}
}

func TestCityTierValues(t *testing.T) {
	if int(TierHamlet) != 1 {
		t.Errorf("TierHamlet should be 1, got %d", int(TierHamlet))
	}
	if int(TierTown) != 2 {
		t.Errorf("TierTown should be 2, got %d", int(TierTown))
	}
	if int(TierCapital) != 3 {
		t.Errorf("TierCapital should be 3, got %d", int(TierCapital))
	}
}

func TestCityFoodCost(t *testing.T) {
	tests := []struct {
		tier         CityTier
		expectedCost int
	}{
		{TierHamlet, 10},
		{TierTown, 20},
		{TierCapital, 30},
	}

	for _, test := range tests {
		city := City{Tier: test.tier}
		cost := city.FoodCost()
		if cost != test.expectedCost {
			t.Errorf("Expected food cost %d for tier %s, got %d", test.expectedCost, test.tier.String(), cost)
		}
	}
}

func TestCityWorldMagic(t *testing.T) {
	magic := &WorldMagic{Name: "Test Magic", Cost: 100, Description: "Test"}

	city := City{Tier: TierCapital}

	if city.HasWorldMagic() {
		t.Error("City should not have world magic initially")
	}

	if city.GetWorldMagic() != nil {
		t.Error("GetWorldMagic should return nil initially")
	}

	city.AssignedWorldMagic = magic

	if !city.HasWorldMagic() {
		t.Error("City should have world magic after assignment")
	}

	if city.GetWorldMagic() != magic {
		t.Error("GetWorldMagic should return the assigned magic")
	}
}
