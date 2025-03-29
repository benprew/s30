package sprites

import (
	"fmt"
	_ "image/png"
	"math"
	"time"

	"github.com/benprew/s30/art"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	// Sprite sheet dimensions
	CharacterRows    = 8
	CharacterColumns = 5

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

// LoadCharacter loads a character and its shadow sprites by character name
func LoadCharacter(name CharacterName) (*Character, error) {
	// Get the shadow name for this character
	charFileName := string(name) + ".spr.png"
	shadowFileName := shadowName(name) + ".spr.png"
	fmt.Printf("char %s shad %s\n", charFileName, shadowFileName)

	// Get the embedded sprite files
	charFile := getEmbeddedFile(charFileName)
	shadowFile := getEmbeddedFile(shadowFileName)

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
		c.Y -= c.MoveSpeed
	}
	if dirBits&DirUp != 0 {
		c.Y += c.MoveSpeed
	}
}

// UpdateAI updates an enemy character's movement based on AI behavior
func (c *Character) UpdateAI(playerX, playerY float64) int {
	// Calculate distance to player
	dx := playerX - c.X
	dy := playerY - c.Y
	distanceToPlayer := math.Sqrt(dx*dx + dy*dy)

	dx /= distanceToPlayer
	dy /= distanceToPlayer

	// Convert to directional movement
	dirbits := 0
	if dx > 0.3 {
		dirbits |= DirRight
	}
	if dx < -0.3 {
		dirbits |= DirLeft
	}
	if dy < 0.3 {
		dirbits |= DirDown
	}
	if dy > -0.3 {
		dirbits |= DirUp
	}

	return dirbits
}

// SetPosition sets the character's position on the map
func (c *Character) SetPosition(x, y float64) {
	c.X = x
	c.Y = y
}

// Draw renders the character and its shadow at the center of the screen
func (c *Character) Draw(screen *ebiten.Image, screenWidth, screenHeight int, scale float64, options *ebiten.DrawImageOptions) {
	// Update animation frame if moving
	if c.IsMoving && time.Since(c.LastUpdate) > time.Millisecond*100 {
		c.Frame = (c.Frame + 1) % CharacterColumns
		c.LastUpdate = time.Now()
	} else if !c.IsMoving {
		c.Frame = 0
	}

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
