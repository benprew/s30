package domain

import "github.com/hajimehoshi/ebiten/v2"

// represents the cities and villages on the map
type City struct {
	Tier            int // 1,2 or 3
	Name            string
	X               int
	Y               int
	Population      int
	BackgroundImage *ebiten.Image
}

func (c *City) FoodCost() int {
	return c.Tier * 10
}
