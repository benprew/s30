package entities

import (
	"fmt"
	_ "image/png"

	"github.com/benprew/s30/art"
	"github.com/benprew/s30/game/sprites"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	// Sprite sheet dimensions
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
	Animations [][]*ebiten.Image // [direction][frame]
	Shadows    [][]*ebiten.Image // [direction][frame]
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
	Animations [][]*ebiten.Image // [direction][frame]
	Shadows    [][]*ebiten.Image // [direction][frame]
}

var Characters map[CharacterName]Sprites = make(map[CharacterName]Sprites, 0)

// NewCharacter creates a new character sprite with animations and shadows
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

// Draw renders the character and its shadow at the center of the screen
func (c *Character) Draw(screen *ebiten.Image, options *ebiten.DrawImageOptions) {
	// Draw shadow first
	screen.DrawImage(c.Shadows[c.Direction][c.Frame], options)
	// Draw character
	screen.DrawImage(c.Animations[c.Direction][c.Frame], options)
}

// Update characters location
func (c *Character) Update(dirBits int) {
	if dirBits == 0 {
		c.IsMoving = false
		return
	} else {
		c.IsMoving = true
	}
	// Update sprite direction based on movement direction
	c.Direction = DirectionToSpriteIndex(dirBits)

	// Apply movement based on direction bits
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

	// Update animation frame if moving
	if c.IsMoving {
		c.Frame = (c.Frame + 1) % SpriteCols
	} else if !c.IsMoving {
		c.Frame = 0
	}
}

func LoadCharacterSprite(name CharacterName) Sprites {
	// Get the shadow name for this character
	charFileName := string(name) + ".spr.png"
	shadowFileName := shadowName(name) + ".spr.png"
	// fmt.Printf("char %s shad %s\n", charFileName, shadowFileName)

	charFile := getEmbeddedFile(charFileName)
	charSheet, err := sprites.LoadSpriteSheet(5, 8, charFile)
	if err != nil {
		panic(fmt.Sprintf("failed to load character sprite: %s file: %s", err, charFile))
	}

	shadowFile := getEmbeddedFile(shadowFileName)
	shadowSheet, err := sprites.LoadSpriteSheet(5, 8, shadowFile)
	if err != nil {
		panic(fmt.Sprintf("failed to load shadow sprite: %s file: %s", err, shadowFile))
	}

	return Sprites{
		Animations: charSheet, Shadows: shadowSheet,
	}
}

// DirectionToSpriteIndex converts bit-based direction to sprite index
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
		return DirectionDown // Default direction
	}
}

// Helper function to get embedded file bytes
func getEmbeddedFile(filename string) []byte {
	// Access character sprites from the embedded filesystem
	data, err := art.CharacterFS.ReadFile("sprites/world/characters/" + filename)
	if err != nil {
		// Log the error but don't crash
		fmt.Printf("Error loading sprite file %s: %v\n", filename, err)
		return nil
	}
	return data
}

// SetPosition sets the character's position on the map
func (c *Character) SetPosition(x, y int) {
	c.X = x
	c.Y = y
}

// SetDirection changes the character's facing direction
func (c *Character) SetDirection(direction int) {
	if direction >= 0 && direction < SpriteRows {
		c.Direction = direction
	}
}

// SetMoving updates the character's movement state
func (c *Character) SetMoving(moving bool) {
	c.IsMoving = moving
	if !moving {
		c.Frame = 0
	}
}
