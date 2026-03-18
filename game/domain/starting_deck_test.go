package domain

import "testing"

func TestShouldSkipCardVintageRestricted(t *testing.T) {
	dg := &DeckGenerator{difficulty: DifficultyEasy}

	restricted := &Card{VintageRestricted: true}
	if !dg.shouldSkipCard(restricted) {
		t.Error("Should skip VintageRestricted cards")
	}

	normal := &Card{}
	if dg.shouldSkipCard(normal) {
		t.Error("Should not skip normal cards")
	}
}

func TestShouldSkipCardVintageRestrictedAllDifficulties(t *testing.T) {
	difficulties := []Difficulty{DifficultyEasy, DifficultyMedium, DifficultyHard, DifficultyExpert}

	for _, diff := range difficulties {
		dg := &DeckGenerator{difficulty: diff}
		restricted := &Card{VintageRestricted: true}
		if !dg.shouldSkipCard(restricted) {
			t.Errorf("Should skip VintageRestricted cards on difficulty %d", diff)
		}
	}
}
