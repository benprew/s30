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
	Character *Character
	CharacterInstance
	Engaged bool
}

func NewEnemy(name string) (Enemy, error) {
	c := Rogues[name]
	if err := c.LoadImages(); err != nil {
		panic(err)
	}
	e := Enemy{Character: c}
	return e, nil
}

func (c *Enemy) Draw(screen *ebiten.Image, options *ebiten.DrawImageOptions) {
	screen.DrawImage(c.Character.ShadowSprite[c.Direction][c.Frame], options)
	screen.DrawImage(c.Character.WalkingSprite[c.Direction][c.Frame], options)
}

func (e *Enemy) Update(pLoc image.Point) error {
	dirBits := e.move(pLoc.X, pLoc.Y)
	e.CharacterInstance.Update(dirBits)
	return nil
}

// move returns direction bits towards player
func (e *Enemy) move(playerX, playerY int) int {
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

func (e *Enemy) SetEngaged(v bool) {
	e.Engaged = v
}

func (e *Enemy) Loc() image.Point {
	return image.Point{X: e.X, Y: e.Y}
}

func (e *Enemy) Dims() image.Rectangle {
	return e.Character.WalkingSprite[0][0].Bounds()
}

func (e *Enemy) BribeAmount() int {
	return 20
}
