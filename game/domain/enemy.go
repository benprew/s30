package domain

import (
	"image"
	"math"
	"math/rand"

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

	waitingTicks      int
	maxWaitTicks      int
	randomDirTicks    int
	maxRandomDirTicks int
	isWaiting         bool
	randomDirBits     int
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

// move returns direction bits with random movement and wait behavior
func (e *Enemy) move(playerX, playerY int) int {
	dx := float64(playerX - e.X)
	dy := float64(playerY - e.Y)
	distToPlayer := math.Sqrt(dx*dx + dy*dy)

	// If close, move directly toward player
	if distToPlayer <= 150 {
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

	// Check if enemy should start waiting
	if !e.isWaiting && distToPlayer > 200 && rand.Float64() < 0.02 {
		e.isWaiting = true
		e.waitingTicks = 0
		e.maxWaitTicks = 20 + rand.Intn(60)
		return 0
	}

	// If waiting, check if player gets close or wait time expires
	if e.isWaiting {
		e.waitingTicks++
		if distToPlayer < 150 || e.waitingTicks >= e.maxWaitTicks {
			e.isWaiting = false
		} else {
			return 0
		}
	}

	// Random movement with occasional direction changes
	e.randomDirTicks++
	if e.randomDirTicks >= e.maxRandomDirTicks || e.maxRandomDirTicks == 0 {
		e.randomDirTicks = 0
		e.maxRandomDirTicks = 30 + rand.Intn(60)

		dirbits := 0
		buffer := 10

		// Add randomness to movement
		if rand.Float64() < 0.7 {
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
		}

		// Add random perpendicular movement
		if rand.Float64() < 0.3 {
			if rand.Float64() < 0.5 {
				if math.Abs(dx) > math.Abs(dy) {
					if rand.Float64() < 0.5 {
						dirbits |= DirUp
					} else {
						dirbits |= DirDown
					}
				} else {
					if rand.Float64() < 0.5 {
						dirbits |= DirLeft
					} else {
						dirbits |= DirRight
					}
				}
			}
		}

		e.randomDirBits = dirbits
	}

	return e.randomDirBits
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
