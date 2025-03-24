package sprites

import (
	_ "image/png"
	"time"

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
	Shadows    [][]*ebiten.Image
	Direction  int
	Frame      int
	LastUpdate time.Time
	IsMoving   bool
}

func NewPlayerSprite(animations, shadows [][]*ebiten.Image) *PlayerSprite {
	return &PlayerSprite{
		Animations: animations,
		Shadows:    shadows,
		Direction:  0,
		Frame:      0,
		LastUpdate: time.Now(),
		IsMoving:   false,
	}
}

func LoadPlayer(sprWidth, sprHeight int, file []byte, Sfile []byte) (*PlayerSprite, error) {
	sheet, err := LoadSpriteSheet(sprWidth, sprHeight, file)
	if err != nil {
		return nil, err
	}
	shadow, err := LoadSpriteSheet(sprWidth, sprHeight, Sfile)
	if err != nil {
		return nil, err
	}
	return NewPlayerSprite(sheet, shadow), nil
}

func (p *PlayerSprite) Draw(screen *ebiten.Image, width, height int, camScale float64) {
	if p.IsMoving && time.Since(p.LastUpdate) > time.Millisecond*100 {
		p.Frame = (p.Frame + 1) % 5
		p.LastUpdate = time.Now()
	} else if !p.IsMoving {
		p.Frame = 0 // Reset to standing frame when not moving
	}

	playerOp := &ebiten.DrawImageOptions{}
	playerOp.GeoM.Translate(-float64(124), -float64(87)) // Center the sprite
	playerOp.GeoM.Scale(camScale, camScale)              // Apply camera zoom
	playerOp.GeoM.Translate(float64(width)/2, float64(height)/2)
	screen.DrawImage(p.Shadows[p.Direction][p.Frame], playerOp)
	screen.DrawImage(p.Animations[p.Direction][p.Frame], playerOp)
}
