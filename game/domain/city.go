package domain

import (
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

// represents the cities and villages on the map
type City struct {
	Tier            int // 1,2 or 3
	Name            string
	X               int
	Y               int
	Population      int
	BackgroundImage *ebiten.Image
	CardsForSale    []int
}

func (c *City) FoodCost() int {
	return c.Tier * 10
}

func MkCards() []int {
	// Pick 5 random indexes from CARDS
	cards := make([]int, 0, 5)
	for i := 0; i < 5; i++ {
		cards = append(cards, rand.Intn(len(CARDS)))
	}
	return cards
}
