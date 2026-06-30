package domain

type Deck map[*Card]int

type AnteExclusion int

const (
	ExcludeBasicLand AnteExclusion = iota
	ExcludeVintageRestricted
)

// NonLandCards returns the distinct non-land cards in the deck. It is the pool
// a dungeon die draws from when granting a card "in play" for the next duel, so
// the grant is always a card the player actually runs.
func (d Deck) NonLandCards() []*Card {
	cards := make([]*Card, 0, len(d))
	for card := range d {
		if card.CardType == CardTypeLand {
			continue
		}
		cards = append(cards, card)
	}
	return cards
}

func (d Deck) ValidAnteCards(exclusions ...AnteExclusion) []*Card {
	excludeBasicLand := true
	excludeVintage := false
	for _, e := range exclusions {
		switch e {
		case ExcludeBasicLand:
			excludeBasicLand = true
		case ExcludeVintageRestricted:
			excludeVintage = true
		}
	}

	cards := []*Card{}
	for card := range d {
		if excludeBasicLand && card.CardType == CardTypeLand {
			continue
		}
		if excludeVintage && card.VintageRestricted {
			continue
		}

		cards = append(cards, card)
	}
	return cards
}
