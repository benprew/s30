package screens

import (
	"strings"
	"testing"

	"github.com/benprew/s30/game/domain"
)

func TestPaginateText(t *testing.T) {
	// Placeholder
}

func TestLoadStories(t *testing.T) {
	stories := loadStories()
	if len(stories) == 0 {
		t.Errorf("Expected stories, got none")
	}
	for i, story := range stories {
		if story == "" {
			t.Errorf("Story %d is empty", i)
		}
		if strings.Contains(story, "STARTBLOCK") || strings.Contains(story, "ENDBLOCK") {
			t.Errorf("Story %d contains delimiters", i)
		}
	}
}

func TestGrantBoonBonusLife(t *testing.T) {
	city := &domain.City{WisemanBoon: domain.BoonBonusLife}
	player := &domain.Player{}
	s := &WisemanScreen{City: city, Player: player}

	s.grantBoon()

	if player.BonusDuelLife != 2 {
		t.Errorf("Expected BonusDuelLife=2, got %d", player.BonusDuelLife)
	}
	if !city.BoonGranted {
		t.Error("Expected BoonGranted to be true")
	}
	if len(s.TextLines) == 0 {
		t.Error("Expected text lines to be set")
	}
}

func TestGrantBoonBonusCard(t *testing.T) {
	city := &domain.City{WisemanBoon: domain.BoonBonusCard}
	player := &domain.Player{}
	s := &WisemanScreen{City: city, Player: player}

	s.grantBoon()

	if len(player.BonusDuelCards) != 1 {
		t.Errorf("Expected 1 bonus card, got %d", len(player.BonusDuelCards))
	}
	if !city.BoonGranted {
		t.Error("Expected BoonGranted to be true")
	}
}

func TestGrantBoonEnemyDeckInfo(t *testing.T) {
	city := &domain.City{WisemanBoon: domain.BoonEnemyDeckInfo}
	player := &domain.Player{}
	s := &WisemanScreen{City: city, Player: player}

	s.grantBoon()

	if !city.BoonGranted {
		t.Error("Expected BoonGranted to be true")
	}
	found := false
	for _, line := range s.TextLines {
		if strings.Contains(line, "deck") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected deck info in text, got: %v", s.TextLines)
	}
}

func TestBoonGrantedShowsStory(t *testing.T) {
	city := &domain.City{
		WisemanBoon: domain.BoonBonusLife,
		BoonGranted: true,
	}
	player := &domain.Player{}
	s := &WisemanScreen{City: city, Player: player}

	s.determineState()

	if s.State != WisemanStateStory {
		t.Errorf("Expected story state after boon granted, got %d", s.State)
	}
	if player.BonusDuelLife != 0 {
		t.Errorf("Should not grant boon again, BonusDuelLife=%d", player.BonusDuelLife)
	}
}

func TestQuestBoonUsesProposedQuest(t *testing.T) {
	targetCity := &domain.City{Name: "TargetVille"}
	originCity := &domain.City{
		Name:        "OriginVille",
		WisemanBoon: domain.BoonQuest,
		ProposedQuest: &domain.Quest{
			Type:          domain.QuestTypeDelivery,
			TargetCity:    targetCity,
			OriginCity:    &domain.City{Name: "OriginVille"},
			DaysRemaining: 25,
			RewardType:    domain.RewardCard,
		},
	}
	player := &domain.Player{}
	s := &WisemanScreen{City: originCity, Player: player}

	s.determineState()

	if s.State != WisemanStateOffer {
		t.Errorf("Expected offer state, got %d", s.State)
	}
	if s.ProposedQuest != originCity.ProposedQuest {
		t.Error("Expected to reuse stored proposed quest")
	}
}

func TestBoonTypeIsQuest(t *testing.T) {
	if !domain.BoonQuest.IsQuest() {
		t.Error("BoonQuest.IsQuest() should return true")
	}
	if domain.BoonBonusLife.IsQuest() {
		t.Error("BoonBonusLife.IsQuest() should return false")
	}
	if domain.BoonNone.IsQuest() {
		t.Error("BoonNone.IsQuest() should return false")
	}
}

func TestActiveQuestUnrelatedCityGrantsBoon(t *testing.T) {
	questOrigin := &domain.City{Name: "QuestTown"}
	otherCity := &domain.City{
		Name:        "OtherTown",
		WisemanBoon: domain.BoonBonusLife,
	}
	player := &domain.Player{
		ActiveQuest: &domain.Quest{
			Type:          domain.QuestTypeDelivery,
			OriginCity:    questOrigin,
			TargetCity:    &domain.City{Name: "TargetTown"},
			DaysRemaining: 10,
		},
	}
	s := &WisemanScreen{City: otherCity, Player: player}

	granted := false
	for range 100 {
		otherCity.BoonGranted = false
		s.determineState()
		if otherCity.BoonGranted {
			granted = true
			break
		}
	}

	if !granted {
		t.Error("Expected boon to be granted at unrelated city while on quest")
	}
}

func TestActiveQuestOriginCityShowsProgress(t *testing.T) {
	originCity := &domain.City{
		Name:        "QuestTown",
		WisemanBoon: domain.BoonQuest,
	}
	player := &domain.Player{
		ActiveQuest: &domain.Quest{
			Type:          domain.QuestTypeDelivery,
			OriginCity:    originCity,
			TargetCity:    &domain.City{Name: "TargetTown"},
			DaysRemaining: 10,
		},
	}
	s := &WisemanScreen{City: originCity, Player: player}

	s.determineState()

	if s.State != WisemanStateActive {
		t.Errorf("Expected active state at quest origin, got %d", s.State)
	}
}

func TestQuestBoonSkippedWithActiveQuest(t *testing.T) {
	city := &domain.City{
		Name:        "SomeCity",
		WisemanBoon: domain.BoonQuest,
	}
	player := &domain.Player{
		ActiveQuest: &domain.Quest{
			Type:          domain.QuestTypeDelivery,
			OriginCity:    &domain.City{Name: "OtherCity"},
			TargetCity:    &domain.City{Name: "TargetCity"},
			DaysRemaining: 10,
		},
	}
	s := &WisemanScreen{City: city, Player: player}

	s.determineState()

	if s.State != WisemanStateStory {
		t.Errorf("Expected story when quest boon city visited with active quest, got %d", s.State)
	}
}

func TestPickBoonNoQuestWithActiveQuest(t *testing.T) {
	city := &domain.City{Name: "TestCity"}
	player := &domain.Player{
		ActiveQuest: &domain.Quest{
			Type:          domain.QuestTypeDelivery,
			OriginCity:    &domain.City{Name: "Other"},
			TargetCity:    &domain.City{Name: "Target"},
			DaysRemaining: 10,
		},
	}

	for range 100 {
		boon := pickBoon(city, player, nil)
		if boon == domain.BoonQuest {
			t.Fatal("pickBoon should never return BoonQuest when player has active quest")
		}
	}
}
