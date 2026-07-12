package screens

import (
	"image"
	"strings"
	"testing"

	"github.com/benprew/s30/game/domain"
)

func TestWisemanStoryClickUsesViewport(t *testing.T) {
	s := &WisemanScreen{State: WisemanStateStory}
	var got image.Rectangle

	clicked := s.storyClicked(1280, 720, func(bounds image.Rectangle) bool {
		got = bounds
		return true
	})

	if !clicked {
		t.Fatal("expected story click to be handled")
	}
	if want := image.Rect(0, 0, 1280, 720); got != want {
		t.Fatalf("story click bounds = %v, want %v", got, want)
	}
}

// disableDeckQuestOffers makes the Wiseman never offer a deck-changing quest for
// the duration of a test, isolating the legacy boon/quest determineState paths
// from the (otherwise random) deck-quest offer.
func disableDeckQuestOffers(t *testing.T) {
	prev := deckQuestOfferChance
	deckQuestOfferChance = 0
	t.Cleanup(func() { deckQuestOfferChance = prev })
}

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
	disableDeckQuestOffers(t)
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
	disableDeckQuestOffers(t)
	targetCity := &domain.City{Name: "TargetVille"}
	originCity := &domain.City{
		Name:        "OriginVille",
		WisemanBoon: domain.BoonQuest,
		ProposedQuest: &domain.Quest{
			Type:          domain.QuestTypeDelivery,
			TargetCity:    targetCity,
			DaysRemaining: 25,
			Reward:        domain.QuestReward{Cards: 1, CardColor: domain.ColorAny},
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
	disableDeckQuestOffers(t)
	otherCity := &domain.City{
		Name:        "OtherTown",
		WisemanBoon: domain.BoonBonusLife,
	}
	player := &domain.Player{
		ActiveQuests: []*domain.Quest{{
			Type:          domain.QuestTypeDelivery,
			TargetCity:    &domain.City{Name: "TargetTown"},
			DaysRemaining: 10,
		}},
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

func TestQuestBoonSkippedWhenAtQuestCapacity(t *testing.T) {
	disableDeckQuestOffers(t)
	city := &domain.City{
		Name:        "SomeCity",
		WisemanBoon: domain.BoonQuest,
	}
	player := &domain.Player{}
	// Fill every quest slot so the city cannot offer its quest boon.
	for range domain.MaxActiveQuests {
		player.AddQuest(&domain.Quest{Type: domain.QuestTypeActionTracker})
	}
	s := &WisemanScreen{City: city, Player: player}

	s.determineState()

	if s.State != WisemanStateStory {
		t.Errorf("Expected story when quest boon city visited at capacity, got %d", s.State)
	}
}

func TestPrepareQuestOfferTextActionHasFlavor(t *testing.T) {
	s := &WisemanScreen{}
	q := &domain.Quest{
		Type:          domain.QuestTypeActionTracker,
		ID:            "cast_blue",
		Title:         "Spells of the Deep",
		Description:   "Cast 25 blue spells",
		DaysRemaining: 25,
		Reward:        domain.QuestReward{Gold: 100},
	}

	s.prepareQuestOfferText(q)

	joined := strings.Join(s.TextLines, "\n")
	if !strings.Contains(joined, "Cast 25 blue spells.") {
		t.Errorf("expected objective in text, got: %v", s.TextLines)
	}
	if !strings.Contains(joined, "Accept the Quest?") {
		t.Errorf("expected accept prompt, got: %v", s.TextLines)
	}
	if len(questFlavor[q.ID]) == 0 {
		t.Fatalf("test precondition: expected bespoke flavor for %q", q.ID)
	}
	for _, line := range questFlavor[q.ID] {
		if !strings.Contains(joined, line) {
			t.Errorf("expected flavor line %q in text, got: %v", line, s.TextLines)
		}
	}
}

func TestPrepareQuestOfferTextConstraintHasFlavor(t *testing.T) {
	s := &WisemanScreen{}
	q := &domain.Quest{
		Type:          domain.QuestTypeDeckConstraint,
		ID:            "mono_color_win",
		Title:         "Purity of Purpose",
		Description:   "Win a duel with a mono-color deck",
		DaysRemaining: 20,
		Reward:        domain.QuestReward{Gold: 100, Cards: 1, CardColor: domain.ColorAny},
	}

	s.prepareQuestOfferText(q)

	joined := strings.Join(s.TextLines, "\n")
	if !strings.Contains(joined, "Win a duel with a mono-color deck.") {
		t.Errorf("expected objective in text, got: %v", s.TextLines)
	}
	if !strings.Contains(joined, "Edit your deck before you duel.") {
		t.Errorf("expected deck-edit hint, got: %v", s.TextLines)
	}
	for _, line := range questFlavor[q.ID] {
		if !strings.Contains(joined, line) {
			t.Errorf("expected flavor line %q in text, got: %v", line, s.TextLines)
		}
	}
}

func TestQuestFlavorLinesFallBackToTitle(t *testing.T) {
	q := &domain.Quest{
		Type:  domain.QuestTypeActionTracker,
		ID:    "made_up_quest_with_no_flavor",
		Title: "An Untold Trial",
	}

	lines := questFlavorLines(q)

	if len(lines) == 0 {
		t.Fatal("expected fallback flavor lines, got none")
	}
	if !strings.Contains(strings.Join(lines, "\n"), q.Title) {
		t.Errorf("expected fallback to mention title %q, got: %v", q.Title, lines)
	}
}

func TestEveryQuestDefHasFlavor(t *testing.T) {
	for _, def := range domain.QuestDefList() {
		if len(questFlavor[def.ID]) == 0 {
			t.Errorf("quest %q has no flavor text", def.ID)
		}
	}
}

func TestQuestFlavorLinesAreCopied(t *testing.T) {
	q := &domain.Quest{Type: domain.QuestTypeActionTracker, ID: "cast_blue"}

	first := questFlavorLines(q)
	first[0] = "mutated"
	second := questFlavorLines(q)

	if second[0] == "mutated" {
		t.Error("questFlavorLines returned a slice aliasing the shared flavor map")
	}
}

func TestRandomRogueNameRespectsMaxLevel(t *testing.T) {
	for _, maxLevel := range []int{1, 3, 5, 8} {
		for range 50 {
			name := randomRogueName(maxLevel)
			rogue, ok := domain.Rogues[name]
			if !ok {
				t.Fatalf("randomRogueName returned unknown rogue %q", name)
			}
			if rogue.Level <= 0 || rogue.Level > maxLevel {
				t.Fatalf("randomRogueName(%d) returned %q with level %d, want 1..%d",
					maxLevel, name, rogue.Level, maxLevel)
			}
		}
	}
}

func TestRandomRogueNameFallsBackBelowLowestLevel(t *testing.T) {
	name := randomRogueName(0)
	if _, ok := domain.Rogues[name]; !ok {
		t.Fatalf("randomRogueName(0) returned unknown rogue %q", name)
	}
}

func TestAcceptDeliveryConsumesCityQuestBoon(t *testing.T) {
	city := &domain.City{Name: "QuestTown", WisemanBoon: domain.BoonQuest}
	proposed := &domain.Quest{Type: domain.QuestTypeDelivery, TargetCity: &domain.City{Name: "Target"}}
	city.ProposedQuest = proposed
	player := &domain.Player{}
	s := &WisemanScreen{City: city, Player: player, ProposedQuest: proposed}

	s.acceptQuest()

	if len(player.ActiveQuests) != 1 || player.ActiveQuests[0] != proposed {
		t.Fatalf("expected delivery quest to be added, have %d", len(player.ActiveQuests))
	}
	if !city.BoonGranted || city.ProposedQuest != nil {
		t.Error("accepting a city quest should mark its boon spent and clear the proposal")
	}
}

func TestPickBoonNoQuestAtCapacity(t *testing.T) {
	city := &domain.City{Name: "TestCity"}
	player := &domain.Player{}
	for range domain.MaxActiveQuests {
		player.AddQuest(&domain.Quest{Type: domain.QuestTypeActionTracker})
	}

	for range 100 {
		if pickBoon(city, player, nil) == domain.BoonQuest {
			t.Fatal("pickBoon should never return BoonQuest when at quest capacity")
		}
	}
}

func TestPickBoonAllowsQuestWithFreeSlot(t *testing.T) {
	city := &domain.City{Name: "TestCity"}
	player := &domain.Player{
		ActiveQuests: []*domain.Quest{{
			Type:          domain.QuestTypeDelivery,
			TargetCity:    &domain.City{Name: "Target"},
			DaysRemaining: 10,
		}},
	}

	offered := false
	for range 100 {
		if pickBoon(city, player, nil) == domain.BoonQuest {
			offered = true
			break
		}
	}
	if !offered {
		t.Error("pickBoon should be able to offer a quest when a slot is free")
	}
}
