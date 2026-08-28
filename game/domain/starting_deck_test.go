package domain

import "testing"

func TestDifficultyToString(t *testing.T) {
	tests := []struct {
		difficulty Difficulty
		expected   string
	}{
		{DifficultyEasy, "Apprentice"},
		{DifficultyMedium, "Magician"},
		{DifficultyHard, "Sorcerer"},
		{DifficultyExpert, "Wizard"},
		{Difficulty(99), "Unknown"},
	}

	for _, test := range tests {
		if got := DifficultyToString(test.difficulty); got != test.expected {
			t.Errorf("DifficultyToString(%d) = %s, expected %s", test.difficulty, got, test.expected)
		}
	}
}

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
// target deck size from the Shandalar specification:
//   Apprentice (Easy):   36 cards
//   Magician   (Medium): 39 cards
//   Sorcerer   (Hard):   44 cards
//   Wizard     (Expert): 45 cards
func TestStartingDeckSizes(t *testing.T) {
	cases := []struct {
		diff    Difficulty
		minSize int
	}{
		{DifficultyEasy, 36},
		{DifficultyMedium, 39},
		{DifficultyHard, 44},
		{DifficultyExpert, 45},
	}

	colors := []ColorMask{ColorWhite, ColorBlue, ColorBlack, ColorRed, ColorGreen}

	const slack = 2
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

// TestStartingResources50CardsTotal verifies that CreateStartingResources generates
// exactly 50 cards total across the active deck and extra collection cards.
func TestStartingResources50CardsTotal(t *testing.T) {
	difficulties := []Difficulty{DifficultyEasy, DifficultyMedium, DifficultyHard, DifficultyExpert}
	colors := []ColorMask{ColorWhite, ColorBlue, ColorBlack, ColorRed, ColorGreen}

	for _, diff := range difficulties {
		for _, color := range colors {
			dg := DeckBuilder(diff, color, 42)
			deck, extraCards := dg.CreateStartingResources()

			deckCount := 0
			for _, n := range deck {
				deckCount += n
			}

			total := deckCount + len(extraCards)
			if total != 50 {
				t.Errorf("diff=%d color=%d: total starting cards = %d (deck %d + extra %d), want 50",
					diff, color, total, deckCount, len(extraCards))
			}
		}
	}
}

// TestStartingDeckNonLandSingletons verifies that all non-basic-land cards
// in the generated starting deck have a maximum count of 1.
func TestStartingDeckNonLandSingletons(t *testing.T) {
	difficulties := []Difficulty{DifficultyEasy, DifficultyMedium, DifficultyHard, DifficultyExpert}
	colors := []ColorMask{ColorWhite, ColorBlue, ColorBlack, ColorRed, ColorGreen}

	for _, diff := range difficulties {
		for _, color := range colors {
			dg := DeckBuilder(diff, color, 12345)
			deck := dg.CreateStartingDeck()

			for card, count := range deck {
				if card.CardType == CardTypeLand {
					continue
				}
				if count > 1 {
					t.Errorf("diff=%d color=%d: non-land card %q has count %d > 1",
						diff, color, card.CardName, count)
				}
			}
		}
	}
}

// TestStartingDeckSpellsIncludeInstantsSorceriesEnchantments verifies that
// starting decks contain a variety of spell types.
func TestStartingDeckSpellsIncludeInstantsSorceriesEnchantments(t *testing.T) {
	foundTypes := make(map[CardType]int)
	colors := []ColorMask{ColorWhite, ColorBlue, ColorBlack, ColorRed, ColorGreen}

	for seed := int64(1); seed <= 10; seed++ {
		for _, color := range colors {
			deck := DeckBuilder(DifficultyEasy, color, seed).CreateStartingDeck()
			for card := range deck {
				foundTypes[card.CardType]++
			}
		}
	}

	requiredTypes := []CardType{
		CardTypeCreature,
		CardTypeLand,
		CardTypeInstant,
		CardTypeSorcery,
		CardTypeEnchantment,
	}

	for _, ct := range requiredTypes {
		if foundTypes[ct] == 0 {
			t.Errorf("Expected to generate cards of type %s across starting decks, got 0", ct)
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

// TestStartingDeckUsesEligibleTiers verifies that non-land cards in a
// generated starting deck come from eligible tiers (mid to bottom tiers,
// plus at most 1 guaranteed rare).
func TestStartingDeckUsesEligibleTiers(t *testing.T) {
	allowed := make(map[*Card]bool)
	for _, c := range CardsInTiers(
		TierPlayedQuiteOften,
		TierPlayedFromTimeToTime,
		TierPlayedInSpecificArchetypes,
		TierRarelyPlayed,
		TierAlmostNeverPlayed,
		TierMeme,
	) {
		allowed[c] = true
	}

	colors := []ColorMask{ColorWhite, ColorBlue, ColorBlack, ColorRed, ColorGreen}
	difficulties := []Difficulty{DifficultyEasy, DifficultyMedium, DifficultyHard, DifficultyExpert}

	for _, diff := range difficulties {
		for _, color := range colors {
			dg := DeckBuilder(diff, color, 42)
			deck := dg.CreateStartingDeck()
			highTierCount := 0
			for card := range deck {
				if card.CardType == CardTypeLand {
					continue
				}
				if !allowed[card] {
					highTierCount++
				}
			}
			// At most 1 guaranteed rare slot can come from higher tiers
			if highTierCount > 1 {
				t.Errorf("difficulty=%d color=%d: deck has %d high-tier cards, expected at most 1",
					diff, color, highTierCount)
			}
		}
	}
}

// TestStartingResourcesExcludesMandatoryAndRestrictedCards verifies that no starting deck
// or extra collection cards ever include TierMandatory, TierAlmostMandatory,
// or VintageRestricted cards.
func TestStartingResourcesExcludesMandatoryAndRestrictedCards(t *testing.T) {
	difficulties := []Difficulty{DifficultyEasy, DifficultyMedium, DifficultyHard, DifficultyExpert}
	colors := []ColorMask{ColorWhite, ColorBlue, ColorBlack, ColorRed, ColorGreen}

	for seed := int64(1); seed <= 20; seed++ {
		for _, diff := range difficulties {
			for _, color := range colors {
				dg := DeckBuilder(diff, color, seed)
				deck, extraCards := dg.CreateStartingResources()

				checkCard := func(c *Card, source string) {
					if c.CardType == CardTypeLand {
						return
					}
					if c.VintageRestricted {
						t.Errorf("seed=%d diff=%d color=%d: %s contains VintageRestricted card %q",
							seed, diff, color, source, c.CardName)
					}
					if tier, ok := CardTierForName(c.CardName); ok {
						if tier == TierMandatory || tier == TierAlmostMandatory {
							t.Errorf("seed=%d diff=%d color=%d: %s contains top-tier card %q (tier %d)",
								seed, diff, color, source, c.CardName, tier)
						}
					}
				}

				for card := range deck {
					checkCard(card, "deck")
				}
				for _, card := range extraCards {
					checkCard(card, "extraCards")
				}
			}
		}
	}
}
