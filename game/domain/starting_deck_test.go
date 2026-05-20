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

// TestStartingDeckSizes verifies each difficulty produces at least the
// minimum deck size from shandalar-faq.txt: Apprentice 30, Magician 35,
// Sorcerer 40, Wizard 40. The generator may leave slots empty if the
// weak-card pool runs out for a color/type, so the bound is a floor with
// some slack.
func TestStartingDeckSizes(t *testing.T) {
	cases := []struct {
		diff    Difficulty
		minSize int
	}{
		{DifficultyEasy, 30},
		{DifficultyMedium, 35},
		{DifficultyHard, 40},
		{DifficultyExpert, 40},
	}

	colors := []ColorMask{ColorWhite, ColorBlue, ColorBlack, ColorRed, ColorGreen}

	const slack = 3
	for _, tc := range cases {
		for _, color := range colors {
			deck := DeckBuilder(tc.diff, color, 42).CreateStartingDeck()
			total := 0
			for _, n := range deck {
				total += n
			}
			if total < tc.minSize-slack {
				t.Errorf("difficulty=%d color=%d: deck has %d cards, expected >=%d",
					tc.diff, color, total, tc.minSize-slack)
			}
		}
	}
}

// TestExpertDeckIsRainbow verifies the Wizard/Expert deck is a 5-color deck:
// basic lands span at least 3 colors and non-land cards span at least 3
// distinct color identities.
func TestExpertDeckIsRainbow(t *testing.T) {
	for _, primary := range []ColorMask{ColorWhite, ColorBlue, ColorBlack, ColorRed, ColorGreen} {
		deck := DeckBuilder(DifficultyExpert, primary, 42).CreateStartingDeck()

		landColors := map[ColorMask]bool{}
		spellColors := map[string]bool{}
		for card := range deck {
			if card.CardType == CardTypeLand {
				if c, ok := basicLands[card.CardName]; ok {
					landColors[c] = true
				}
				continue
			}
			for _, c := range card.ColorIdentity {
				spellColors[c] = true
			}
		}

		if len(landColors) < 3 {
			t.Errorf("primary=%d: expert deck has basics of only %d colors, want >=3",
				primary, len(landColors))
		}
		if len(spellColors) < 3 {
			t.Errorf("primary=%d: expert deck spells span only %d colors, want >=3",
				primary, len(spellColors))
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
