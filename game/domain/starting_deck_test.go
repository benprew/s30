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

// TestStartingDeckUsesOnlyBottomTiers verifies that non-land cards in a
// generated starting deck come from the rarely-played, almost-never-played,
// or meme tiers. Basic lands are exempt since they are not ranked.
func TestStartingDeckUsesOnlyBottomTiers(t *testing.T) {
	allowed := make(map[*Card]bool)
	for _, c := range CardsInTiers(TierRarelyPlayed, TierAlmostNeverPlayed, TierMeme) {
		allowed[c] = true
	}

	colors := []ColorMask{ColorWhite, ColorBlue, ColorBlack, ColorRed, ColorGreen}
	difficulties := []Difficulty{DifficultyEasy, DifficultyMedium, DifficultyHard, DifficultyExpert}

	for _, diff := range difficulties {
		for _, color := range colors {
			dg := DeckBuilder(diff, color, 42)
			deck := dg.CreateStartingDeck()
			for card := range deck {
				if card.CardType == CardTypeLand {
					continue
				}
				if !allowed[card] {
					t.Errorf("difficulty=%d color=%d: card %q not in bottom tiers",
						diff, color, card.CardName)
				}
			}
		}
	}
}
