package domain

import (
	"fmt"
	_ "image/png"

	"github.com/benprew/s30/assets"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	SpriteRows   = 8
	SpriteCols   = 5
	SpriteWidth  = 206
	SpriteHeight = 102

	// Direction bit flags
	DirUp    = 0x8 // 1000
	DirDown  = 0x4 // 0100
	DirLeft  = 0x2 // 0010
	DirRight = 0x1 // 0001

	// Animation directions
	DirectionDown      = 0
	DirectionDownLeft  = 1
	DirectionLeft      = 2
	DirectionUpLeft    = 3
	DirectionUp        = 4
	DirectionUpRight   = 5
	DirectionRight     = 6
	DirectionDownRight = 7
)

type Character struct {
	Name                  string            `toml:"name"`
	Tier                  int               `tome:"tier"`
	Visage                *ebiten.Image     // rogues headshot, seen at start of duel
	VisageFn              string            `toml:"face"` // filename only, lazy-loaded later
	WalkingSprite         [][]*ebiten.Image // sprites for walking animation
	ShadowSprite          [][]*ebiten.Image // sprites for shadow animation
	WalkingSpriteFn       string            `toml:"walking_sprite"`        // filename only, lazy-loaded later
	WalkingShadowSpriteFn string            `toml:"walking_shadow_sprite"` // filename only, lazy-loaded later
	Life                  int               `toml:"life"`
	Catchphrases          []string          `toml:"catchphrases"` // rogues only
	DeckRaw               [][]string        `toml:"main_cards"`
	Deck                  Deck              // TODO: make this card indexes or similar
	SideboardRaw          [][]string        `toml:"sideboard_cards"`
	Sideboard             Deck
}

// contains the common character traits between players and enemies
type CharacterInstance struct {
	Direction int
	Frame     int
	IsMoving  bool
	X         int
	Y         int
	MoveSpeed int
	Width     int
	Height    int
}

func (c *CharacterInstance) Update(dirBits int) {
	if dirBits == 0 {
		c.IsMoving = false
		return
	}
	c.IsMoving = true
	c.Direction = directionToSpriteIndex(dirBits)
	if dirBits&DirLeft != 0 {
		c.X -= c.MoveSpeed
	}
	if dirBits&DirRight != 0 {
		c.X += c.MoveSpeed
	}
	if dirBits&DirDown != 0 {
		c.Y += c.MoveSpeed
	}
	if dirBits&DirUp != 0 {
		c.Y -= c.MoveSpeed
	}
	if c.IsMoving {
		c.Frame = (c.Frame + 1) % SpriteCols
	} else {
		c.Frame = 0
	}
}

func directionToSpriteIndex(dirBits int) int {
	switch dirBits {
	case DirUp:
		return DirectionUp
	case DirDown:
		return DirectionDown
	case DirLeft:
		return DirectionLeft
	case DirRight:
		return DirectionRight
	case DirUp | DirLeft:
		return DirectionUpLeft
	case DirUp | DirRight:
		return DirectionUpRight
	case DirDown | DirLeft:
		return DirectionDownLeft
	case DirDown | DirRight:
		return DirectionDownRight
	default:
		return DirectionDown
	}
}

func getEmbeddedFile(filename string) []byte {
	data, err := assets.CharacterFS.ReadFile("art/sprites/world/characters/" + filename)
	if err != nil {
		fmt.Printf("Error loading sprite file %s: %v\n", filename, err)
		return nil
	}
	return data
}
