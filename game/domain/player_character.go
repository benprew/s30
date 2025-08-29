package domain

import (
	"image"

	"github.com/benprew/s30/game/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

// Player is a game character with player-specific fields.
type Player struct {
	Character
	Name    string
	Gold    int
	Food    int
	Life    int
	CardMap map[int]int
}

func NewPlayer(name CharacterName) (*Player, error) {
	c, err := NewCharacter(name)
	if err != nil {
		return nil, err
	}
	return &Player{
		Character: *c,
		Name:      string(name),
		Gold:      1200,
		Food:      30,
		Life:      8,
		CardMap:   make(map[int]int),
	}, nil
}

func (p *Player) Draw(screen *ebiten.Image, options *ebiten.DrawImageOptions) {
	p.Character.Draw(screen, options)
}

func (p *Player) Update(screenW, screenH, levelW, levelH int) error {
	dirBits := p.Move(screenW, screenH)
	p.Character.Update(dirBits)

	if p.X < screenW/2 {
		p.X = screenW / 2
	}
	if p.X > levelW-screenW/2 {
		p.X = levelW - screenW/2
	}
	if p.Y < screenH/2 {
		p.Y = screenH / 2
	} else if p.Y > levelH-screenH/2 {
		p.Y = levelH - screenH/2
	}

	return nil
}

func (p *Player) SetLoc(loc image.Point) {
	p.X = loc.X
	p.Y = loc.Y
}

func (p *Player) Move(screenW, screenH int) (dirBits int) {
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

	var cursorX, cursorY = ui.TouchPosition()
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		cursorX, cursorY = ebiten.CursorPosition()
	}

	if cursorX > 0 && cursorY > 0 {
		playerScreenX := screenW / 2
		playerScreenY := screenH / 2

		deltaX := cursorX - playerScreenX
		deltaY := cursorY - playerScreenY

		const moveThreshold = 50

		if deltaX > moveThreshold {
			dirBits |= DirRight
		}
		if deltaX < -moveThreshold {
			dirBits |= DirLeft
		}
		if deltaY > moveThreshold {
			dirBits |= DirDown
		}
		if deltaY < -moveThreshold {
			dirBits |= DirUp
		}
	}

	return dirBits
}

func (p *Player) Loc() image.Point {
	return image.Point{X: p.X, Y: p.Y}
}
