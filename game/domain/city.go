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
}

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
	// Pick 5 random indexes from CARDS
	cards := make([]*Card, 0, 5)
	for i := 0; i < 5; i++ {
		cards = append(cards, CARDS[rand.Intn(len(CARDS))])
	}
	return cards
}
