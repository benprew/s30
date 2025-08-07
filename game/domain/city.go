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

func (c *City) MkCards() {
	// Pick 5 random indexes from CARDS
	cardIndexes := make([]int, 5)
	for i := 0; i < 5; i++ {
		cardIndexes[i] = rand.Intn(len(CARDS))
	}
	c.CardsForSale = cardIndexes
}
