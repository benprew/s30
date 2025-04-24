package entities

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

// the player character
type Player struct {
	character *Character
	name      CharacterName
}

func NewPlayer(name CharacterName) (*Player, error) {
	c, err := NewCharacter(name)
	return &Player{
		character: c,
		name:      name,
	}, err
}

func (p *Player) Draw(screen *ebiten.Image, options *ebiten.DrawImageOptions) {
	p.character.Draw(screen, options)
}

func (p *Player) Update(screenW, screenH, levelW, levelH int) error {
	dirBits := p.Move()
	p.character.Update(dirBits)

	// Clamp player position to world boundaries
	if p.character.X < screenW/2 {
		p.character.X = screenW / 2
	}
	if p.character.X > levelW-screenW/2 {
		p.character.X = levelW - screenW/2
	}
	if p.character.Y < screenH/2 {
		p.character.Y = screenH / 2
	} else if p.character.Y > levelH-screenH/2 {
		p.character.Y = levelH - screenH/2
	}

	return nil
}

func (p *Player) SetLoc(loc image.Point) {
	p.character.X = loc.X
	p.character.Y = loc.Y
}

func (p *Player) Move() (dirBits int) {
	if ebiten.IsKeyPressed(ebiten.KeyLeft) || ebiten.IsKeyPressed(ebiten.KeyA) {
		dirBits |= DirLeft
	}
	if ebiten.IsKeyPressed(ebiten.KeyRight) || ebiten.IsKeyPressed(ebiten.KeyD) {
		dirBits |= DirRight
	}
	if ebiten.IsKeyPressed(ebiten.KeyDown) || ebiten.IsKeyPressed(ebiten.KeyS) {
		dirBits |= DirDown
	}
	if ebiten.IsKeyPressed(ebiten.KeyUp) || ebiten.IsKeyPressed(ebiten.KeyW) {
		dirBits |= DirUp
	}
	return dirBits
}

func (p *Player) Loc() image.Point {
	return image.Point{X: p.character.X, Y: p.character.Y}
}
