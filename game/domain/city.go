package domain

import (
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

type CityTier int

const (
	TierHamlet  CityTier = 1
	TierTown    CityTier = 2
	TierCapital CityTier = 3
)

func (ct CityTier) String() string {
	switch ct {
	case TierHamlet:
		return "Hamlet"
	case TierTown:
		return "Town"
	case TierCapital:
		return "Capital"
	default:
		return "Unknown"
	}
}

// represents the cities and villages on the map
type City struct {
	Tier               CityTier
	Name               string
	X                  int
	Y                  int
	Population         int
	BackgroundImage    *ebiten.Image
	CardsForSale       []*Card
	AmuletColor        ColorMask
	AssignedWorldMagic *WorldMagic
	QuestBanDays       int
	IsManaLinked       bool
}

// ==============================================================================
// 1.14 Why do the prices change for cards?
// ==============================================================================
//
//   It seems that each town or village has a different rate for cards.
//   There are 6 different types of cards: the 5 colors and colorless. Each
//   card has a fixed rate, for example, basic lands are 20 gold. The
//   easiest way to find out the rate at which a color is going for is to
//   attempt to sell a basic land when you are editing you deck at that
//   town/village. Here is a short table which I think works.
//
//     Rate	   Land Price Colorless Land
//     ----    ---------- --------------
//     50%     10         50
//     75%     15         75
//     100%    20         100
//     150%    25         125
//     200%    30         150
//
//   Usually Towns have a higher rate than villages. It also seems that the
//   buying price is exactly the same as the selling price.
//   I don't know if table is correct. It seems like the colorless land
//   prices are not right at all. Also, the card prices are rounded to the
//   nearest 5 gold.
//
//   Note: You can combine this knowledge with the Negative Card Bug. Go to
//   a village where the rate for that card is 50% and sell -50 of that
//   card. Then go to a town where the rate is 200% and sell those cards.
//   Instant Free Cash!
//
//   Note: Beth Moursund says that there should be a difference between the
//   buying price and the selling price, but I've never seen a difference.

func (c *City) FoodCost() int {
	return int(c.Tier) * 10
}

func (c *City) HasWorldMagic() bool {
	return c.AssignedWorldMagic != nil
}

func (c *City) GetWorldMagic() *WorldMagic {
	return c.AssignedWorldMagic
}

func MkCards() []*Card {
	cards := make([]*Card, 0, 5)
	for i := 0; i < 5; i++ {
		card := CARDS[rand.Intn(len(CARDS))]
		if card.VintageRestricted {
			i--
			continue
		}
		cards = append(cards, card)
	}
	return cards
}
