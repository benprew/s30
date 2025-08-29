package domain

import (
	"fmt"
	_ "image/png"

	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/sprites"
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

// contains the common character traits between players and enemies
type Character struct {
	Animations [][]*ebiten.Image
	Shadows    [][]*ebiten.Image
	Direction  int
	Frame      int
	IsMoving   bool
	X          int
	Y          int
	MoveSpeed  int
	Width      int
	Height     int
}

type Sprites struct {
	Animations [][]*ebiten.Image
	Shadows    [][]*ebiten.Image
}

var Characters map[CharacterName]Sprites = make(map[CharacterName]Sprites, 0)

func NewCharacter(name CharacterName) (*Character, error) {
	charSprite, ok := Characters[name]

	if !ok {
		charSprite = LoadCharacterSprite(name)
		Characters[name] = charSprite
	}

	return &Character{
		Animations: charSprite.Animations,
		Shadows:    charSprite.Shadows,
		Direction:  DirectionDown,
		Frame:      0,
		IsMoving:   false,
		MoveSpeed:  10,
		Width:      charSprite.Animations[0][0].Bounds().Dx(),
		Height:     charSprite.Animations[0][0].Bounds().Dy(),
	}, nil
}

func (c *Character) Draw(screen *ebiten.Image, options *ebiten.DrawImageOptions) {
	screen.DrawImage(c.Shadows[c.Direction][c.Frame], options)
	screen.DrawImage(c.Animations[c.Direction][c.Frame], options)
}

func (c *Character) Update(dirBits int) {
	if dirBits == 0 {
		c.IsMoving = false
		return
	}
	c.IsMoving = true
	c.Direction = DirectionToSpriteIndex(dirBits)
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

func LoadCharacterSprite(name CharacterName) Sprites {
	charFileName := string(name) + ".spr.png"
	shadowFileName := ShadowName(name) + ".spr.png"

	charFile := getEmbeddedFile(charFileName)
	charSheet, err := sprites.LoadSpriteSheet(5, 8, charFile)
	if err != nil {
		panic(fmt.Sprintf("failed to load character sprite: %s file: %s", err, charFileName))
	}

	shadowFile := getEmbeddedFile(shadowFileName)
	shadowSheet, err := sprites.LoadSpriteSheet(5, 8, shadowFile)
	if err != nil {
		panic(fmt.Sprintf("failed to load shadow sprite: %s file: %s", err, shadowFileName))
	}

	return Sprites{Animations: charSheet, Shadows: shadowSheet}
}

func DirectionToSpriteIndex(dirBits int) int {
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

func (c *Character) SetPosition(x, y int) { c.X = x; c.Y = y }
func (c *Character) SetDirection(direction int) {
	if direction >= 0 && direction < SpriteRows {
		c.Direction = direction
	}
}
func (c *Character) SetMoving(moving bool) {
	c.IsMoving = moving
	if !moving {
		c.Frame = 0
	}
}
