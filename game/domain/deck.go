package domain

type Deck map[*Card]int

func (d Deck) ValidAnteCards(excludeBasicLand bool) []*Card {
	cards := []*Card{}
	for card := range d {
		if excludeBasicLand && card.CardType == CardTypeLand {
			continue
		}

		cards = append(cards, card)
	}
	return cards
}
