package domain

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

type Dimension struct {
	Height int
	Width  int
}

// Enemy is a game opponent, an instantiation of a Rogue and Character.
type Enemy struct {
	Character
	Name    CharacterName
	DeckIdx int
	Engaged bool
	Rogue   *Rogue
}

func NewEnemy(name CharacterName) (Enemy, error) {
	c, err := NewCharacter(name)
	if err != nil {
		return Enemy{}, err
	}
	e := Enemy{Character: *c, Name: name}
	// Try to attach rogue data by sprite filename
	spriteFn := string(name) + ".spr.png"
	if RoguesBySprite != nil {
		if r, ok := RoguesBySprite[spriteFn]; ok {
			e.Rogue = r
		}
	}
	return e, nil
}

func (e *Enemy) Draw(screen *ebiten.Image, options *ebiten.DrawImageOptions) {
	e.Character.Draw(screen, options)
}

func (e *Enemy) Update(pLoc image.Point) error {
	dirBits := e.Move(pLoc.X, pLoc.Y)
	e.Character.Update(dirBits)
	return nil
}

// Move returns direction bits towards player
func (e *Enemy) Move(playerX, playerY int) int {
	dirbits := 0
	buffer := 10
	if playerX > e.X+buffer {
		dirbits |= DirRight
	}
	if playerX < e.X-buffer {
		dirbits |= DirLeft
	}
	if playerY > e.Y+buffer {
		dirbits |= DirDown
	}
	if playerY < e.Y-buffer {
		dirbits |= DirUp
	}
	return dirbits
}

func (e *Enemy) SetLoc(loc image.Point) {
	e.X = loc.X
	e.Y = loc.Y
}

func (e *Enemy) Speed(speed int) {
	e.MoveSpeed = speed
}

func (e *Enemy) SetEngaged(v bool) {
	e.Engaged = v
}

func (e *Enemy) Loc() image.Point {
	return image.Point{X: e.X, Y: e.Y}
}

func (e *Enemy) Dims() Dimension {
	return Dimension{Height: e.Height, Width: e.Width}
}

func (e *Enemy) NameVal() CharacterName {
	return e.Name
}
