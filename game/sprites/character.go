package sprites

import (
	"fmt"
	_ "image/png"
	"time"

	"github.com/benprew/s30/art"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	// Sprite sheet dimensions
	CharacterRows    = 8
	CharacterColumns = 5

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

// Character represents an animated character sprite with its shadow
type Character struct {
	Animations [][]*ebiten.Image // [direction][frame]
	Shadows    [][]*ebiten.Image // [direction][frame]
	Direction  int
	Frame      int
	LastUpdate time.Time
	IsMoving   bool
	X          float64
	Y          float64
	MoveSpeed  float64
}

// NewCharacter creates a new character sprite with animations and shadows
func NewCharacter(animations, shadows [][]*ebiten.Image) *Character {
	return &Character{
		Animations: animations,
		Shadows:    shadows,
		Direction:  DirectionDown,
		Frame:      0,
		LastUpdate: time.Now(),
		IsMoving:   false,
		MoveSpeed:  3.75,
	}
}

// LoadCharacter loads a character and its shadow sprites by character name
func LoadCharacter(name CharacterName) (*Character, error) {
	// Get the shadow name for this character
	shadowName := getShadowName(name)

	// Get the embedded sprite files
	charFile := getEmbeddedFile(string(name) + ".spr.png")
	shadowFile := getEmbeddedFile(shadowName + ".spr.png")

	if charFile == nil || shadowFile == nil {
		return nil, fmt.Errorf("failed to find sprite files for character %s", name)
	}

	charSheet, err := LoadSpriteSheet(5, 8, charFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load character sprite: %w", err)
	}

	shadowSheet, err := LoadSpriteSheet(5, 8, shadowFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load shadow sprite: %w", err)
	}

	return NewCharacter(charSheet, shadowSheet), nil
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

// Update characters location
func (c *Character) Update(up, down, left, right bool) {
	// Update player position based on input
	if left {
		c.X -= c.MoveSpeed
	}
	if right {
		c.X += c.MoveSpeed
	}
	if down {
		c.Y -= c.MoveSpeed
	}
	if up {
		c.Y += c.MoveSpeed
	}

	// Update player movement state based on input
	c.IsMoving = up || down || left || right

	if c.IsMoving {
		if up && right {
			c.Direction = 5 // upRight
		} else if up && left {
			c.Direction = 3 // upLeft
		} else if down && right {
			c.Direction = 7 // downRight
		} else if down && left {
			c.Direction = 1 // downLeft
		} else if up {
			c.Direction = 4 // up
		} else if down {
			c.Direction = 0 // down
		} else if left {
			c.Direction = 2 // left
		} else if right {
			c.Direction = 6 // right
		}
	}

	// This method can be expanded later for AI movement
}

// SetPosition sets the character's position on the map
func (c *Character) SetPosition(x, y float64) {
	c.X = x
	c.Y = y
}

// Draw renders the character and its shadow at the center of the screen
func (c *Character) Draw(screen *ebiten.Image, screenWidth, screenHeight int, scale float64) {
	// Update animation frame if moving
	if c.IsMoving && time.Since(c.LastUpdate) > time.Millisecond*100 {
		c.Frame = (c.Frame + 1) % CharacterColumns
		c.LastUpdate = time.Now()
	} else if !c.IsMoving {
		c.Frame = 0
	}

	options := &ebiten.DrawImageOptions{}
	options.GeoM.Scale(scale, scale)
	options.GeoM.Translate(-float64(124)*scale, -float64(87)*scale) // Center the sprite
	options.GeoM.Translate(float64(screenWidth)/2, float64(screenHeight)/2)

	// Draw shadow first
	screen.DrawImage(c.Shadows[c.Direction][c.Frame], options)
	// Draw character
	screen.DrawImage(c.Animations[c.Direction][c.Frame], options)
}

// SetDirection changes the character's facing direction
func (c *Character) SetDirection(direction int) {
	if direction >= 0 && direction < CharacterRows {
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
