package domain

import (
	"math/rand"
)

// DeckPrimaryColor returns the color that appears most often across the deck's
// cards, weighted by how many copies of each card the deck runs. A deck with no
// colored cards returns ColorColorless.
func DeckPrimaryColor(deck Deck) ColorMask {
	counts := map[ColorMask]int{}
	for card, n := range deck {
		for _, s := range card.ColorIdentity {
			if m, ok := colorStringToMask[s]; ok {
				counts[m] += n
			}
		}
	}

	best := ColorColorless
	bestCount := 0
	for _, m := range []ColorMask{ColorWhite, ColorBlue, ColorBlack, ColorRed, ColorGreen} {
		if counts[m] > bestCount {
			bestCount = counts[m]
			best = m
		}
	}
	return best
}

// RandomBasicLand returns a basic land card of a random color, or nil if the
// card database contains none.
func RandomBasicLand() *Card {
	names := make([]string, 0, len(basicLands))
	for name := range basicLands {
		names = append(names, name)
	}
	rand.Shuffle(len(names), func(i, j int) { names[i], names[j] = names[j], names[i] })
	for _, name := range names {
		if card := FindCardByName(name); card != nil {
			return card
		}
	}
	return nil
}

// RewardChoices returns the cards a player may pick from after winning a duel:
// the opponent's ante card, a random low-to-medium power card in the player
// deck's primary color, and a random basic land. Entries that can't be produced
// (no opponent ante, a colorless deck, or an empty pool) are omitted, so callers
// should handle a slice shorter than three.
func RewardChoices(playerDeck Deck, opponentCard *Card) []*Card {
	choices := []*Card{}

	if opponentCard != nil {
		choices = append(choices, opponentCard)
	}

	if color := DeckPrimaryColor(playerDeck); color != ColorColorless {
		if cards := RandomLowMidCardsForColor(color, 1); len(cards) > 0 {
			choices = append(choices, cards[0])
		}
	}

	if land := RandomBasicLand(); land != nil {
		choices = append(choices, land)
	}

	return choices
}
