package domain

// the player character
type Player struct {
	Name        string
	Gold        int
	Food        int
	Life        int
	CardMap     map[int]int // list of cardIds and quantity the player owns
	WorldMagics []int       // list of world magics the player has
}

func NewPlayer(name string) *Player {
	return &Player{Name: name, Gold: 1200, Food: 30, Life: 8}
}
