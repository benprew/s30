package entities

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

type Dimension struct {
	Height int
	Width  int
}

// An enemy is an opponent of the player.
// enemies walk around in the world and if they touch a player that starts a game
// of magic
type Enemy struct {
	character *Character
	name      CharacterName
	deck      int
}

func NewEnemy(name CharacterName) (Enemy, error) {
	c, err := NewCharacter(name)
	return Enemy{character: c, name: name}, err
}

func (e *Enemy) Draw(screen *ebiten.Image, options *ebiten.DrawImageOptions) {
	e.character.Draw(screen, options)
}

func (e *Enemy) Update(pLoc image.Point) error {
	dirBits := e.Move(pLoc.X, pLoc.Y)
	e.character.Update(dirBits)

	return nil
}

// Direction enemy should move, it will move towards player
func (e *Enemy) Move(playerX, playerY int) int {
	dirbits := 0
	buffer := 3 // pixel buffer so enemies don't tweak out
	if playerX > e.character.X+buffer {
		dirbits |= DirRight
	}
	if playerX < e.character.X-buffer {
		dirbits |= DirLeft
	}
	if playerY > e.character.Y+buffer {
		dirbits |= DirDown
	}
	if playerY < e.character.Y-buffer {
		dirbits |= DirUp
	}
	return dirbits
}

func (e *Enemy) SetLoc(loc image.Point) {
	e.character.X = loc.X
	e.character.Y = loc.Y
}

func (e *Enemy) Speed(speed int) {
	e.character.MoveSpeed = speed
}

func (e *Enemy) Loc() image.Point {
	return image.Point{X: e.character.X, Y: e.character.Y}
}

func (e *Enemy) Dims() Dimension {
	return Dimension{Height: e.character.Height, Width: e.character.Width}
}
