package domain

type Deck map[*Card]int

type AnteExclusion int

const (
	ExcludeBasicLand       AnteExclusion = iota
	ExcludeVintageRestricted
)

func (d Deck) ValidAnteCards(exclusions ...AnteExclusion) []*Card {
	excludeBasicLand := false
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
