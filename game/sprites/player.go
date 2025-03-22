package sprites

import (
	_ "image/png"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	down = iota
	downLeft
	left
	leftUp
	up
	upRight
	right
	downRight
)

type PlayerSprite struct {
	Animations [][]*ebiten.Image
}

func LoadPlayer(sprWidth, sprHeight int, file []byte) (*PlayerSprite, error) {
	sheet, err := LoadSpriteSheet(sprWidth, sprHeight, file)
	return &PlayerSprite{sheet}, err
}
