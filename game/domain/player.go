package domain

// the player character
type Player struct {
	Name        string
	Gold        int
	Food        int
	WorldMagics []int
}

func NewPlayer(name string) *Player {
	return &Player{Name: name, Gold: 120, Food: 30}
}
