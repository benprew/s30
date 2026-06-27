package domain

import (
	"fmt"
	"math/bits"
)

// ValidateDeckConstraint reports whether the given (un-padded, as-built) deck
// satisfies a deck-constraint quest, plus a short reason when it does not.
//
// The deck passed in must be the player's constructed deck
// (CardCollection.GetDeck), NOT GetDuelDeck — the latter pads to MinDeckSize
// with random basics, which would inflate the fat-deck and color-light checks.
// These quests measure the player's build intent.
//
// ConstraintNoAttacking is not a deck property; it always passes the deck check
// and is enforced at duel end (zero attackers declared by the player).
func ValidateDeckConstraint(q *Quest, deck Deck) (bool, string) {
	switch q.Constraint {
	case ConstraintMonoColor:
		return validateMonoColor(q, deck)
	case ConstraintFatDeck:
		return validateFatDeck(q, deck)
	case ConstraintLowCurve:
		return validateLowCurve(q, deck)
	case ConstraintColorLight:
		return validateColorLight(q, deck)
	case ConstraintNoAttacking, ConstraintNone:
		return true, ""
	default:
		return true, ""
	}
}

func validateMonoColor(q *Quest, deck Deck) (bool, string) {
	var union ColorMask
	for card := range deck {
		if card.IsLand() {
			continue
		}
		union |= card.ColorMask()
	}
	if bits.OnesCount(uint(union)) > 1 {
		return false, "All non-land cards must share a single color"
	}
	if q.Color != ColorColorless && union != ColorColorless && union != q.Color {
		return false, "Deck must be mono-" + ColorMaskToString(q.Color)
	}
	return true, ""
}

func validateFatDeck(q *Quest, deck Deck) (bool, string) {
	total := 0
	for _, count := range deck {
		total += count
	}
	if total <= q.ConstraintN {
		return false, fmt.Sprintf("Deck must be more than %d cards", q.ConstraintN)
	}
	return true, ""
}

func validateLowCurve(q *Quest, deck Deck) (bool, string) {
	for card := range deck {
		if card.CardType != CardTypeCreature {
			continue
		}
		if card.ManaValue() > q.ConstraintN {
			return false, fmt.Sprintf("Every creature must have mana value %d or less", q.ConstraintN)
		}
	}
	return true, ""
}

func validateColorLight(q *Quest, deck Deck) (bool, string) {
	for card := range deck {
		if card.ColorMask()&q.Color != 0 {
			return false, "Deck must use no " + ColorMaskToString(q.Color) + " cards"
		}
	}
	return true, ""
}
