package domain

import (
	"fmt"
	"image"
	_ "image/png"
	"math"

	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/timing"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	SpriteRows   = 8
	SpriteCols   = 5
	SpriteWidth  = 206
	SpriteHeight = 102

	updatesPerWalkingFrame = timing.UpdatesPerSecond / 10
	diagonalMovementScale  = 1 / math.Sqrt2

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

// MovementSpeed converts pixels per second to distance per update.
func MovementSpeed(pixelsPerSecond float64) float64 {
	return pixelsPerSecond / timing.UpdatesPerSecond
}

type Character struct {
	Name                  string `toml:"name"`
	Tier                  int    `toml:"tier"`
	Level                 int    `toml:"level"`
	PrimaryColor          string
	ColorIdentity         []string
	Visage                *ebiten.Image     // rogues headshot, seen at start of duel
	VisageFn              string            `toml:"face"` // filename only, lazy-loaded later
	WalkingSprite         [][]*ebiten.Image // sprites for walking animation
	ShadowSprite          [][]*ebiten.Image // sprites for shadow animation
	WalkingSpriteFn       string            `toml:"walking_sprite"`        // filename only, lazy-loaded later
	WalkingShadowSpriteFn string            `toml:"walking_shadow_sprite"` // filename only, lazy-loaded later
	Life                  int               `toml:"life"`
	Catchphrases          []string          `toml:"catchphrases"` // rogues only
	DeckRaw               [][]string        `toml:"main_cards"`
	SideboardRaw          [][]string        `toml:"sideboard_cards"`
	CardCollection        CardCollection    // replaces Deck and Sideboard
}

// contains the common character traits between players and enemies
type CharacterInstance struct {
	Direction        int
	Frame            int
	IsMoving         bool
	X                int
	Y                int
	MoveSpeed        float64
	MoveSpeedPenalty float64 // 0.0 is full speed. Single multiplier for now, can make more robust with an array later.
	Width            int
	Height           int

	moveRemainderX float64
	moveRemainderY float64
	animationTicks int
}

func (c *CharacterInstance) Update(dirBits int) {
	c.UpdateWithCollision(dirBits, func(image.Point) bool {
		return false
	})
}

func (c *CharacterInstance) UpdateWithCollision(dirBits int, isBlocked func(image.Point) bool) {
	if dirBits == 0 || !c.canMove(dirBits, isBlocked) {
		c.IsMoving = false
		c.animationTicks = 0
		return
	}
	c.IsMoving = true
	c.Direction = directionToSpriteIndex(dirBits)
	dx, dy := c.movementDelta(dirBits)
	if dx != 0 {
		moveCoordinate(&c.X, &c.moveRemainderX, dx)
	}
	if dy != 0 {
		moveCoordinate(&c.Y, &c.moveRemainderY, dy)
	}
	c.animationTicks++
	if c.animationTicks >= updatesPerWalkingFrame {
		c.Frame = (c.Frame + 1) % SpriteCols
		c.animationTicks = 0
	}
}

func (c *CharacterInstance) movementDelta(dirBits int) (float64, float64) {
	dx, dy := 0, 0

	if dirBits&DirLeft != 0 {
		dx--
	}
	if dirBits&DirRight != 0 {
		dx++
	}
	if dirBits&DirDown != 0 {
		dy++
	}
	if dirBits&DirUp != 0 {
		dy--
	}

	distance := c.activeMoveSpeed()
	if dx != 0 && dy != 0 {
		distance *= diagonalMovementScale
	}

	return float64(dx) * distance, float64(dy) * distance
}

func (c *CharacterInstance) canMove(dirBits int, isBlocked func(image.Point) bool) bool {
	dx, dy := c.movementDelta(dirBits)

	newX := c.X + int(math.Trunc(c.moveRemainderX+dx))
	newY := c.Y + int(math.Trunc(c.moveRemainderY+dy))

	return !isBlocked(image.Point{X: newX, Y: newY})
}

func moveCoordinate(position *int, remainder *float64, distance float64) {
	total := *remainder + distance
	nearestInteger := math.Round(total)
	if math.Abs(total-nearestInteger) < 1e-9 {
		total = nearestInteger
	}
	whole := math.Trunc(total)
	*position += int(whole)
	*remainder = total - whole
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

func (c *Character) calculateLifeFromLevel() int {
	if c.Level > 0 {
		switch c.Level {
		case 1:
			return 10
		case 2:
			return 12
		case 3:
			return 14
		case 4:
			return 16
		case 5:
			return 18
		case 6:
			return 19
		case 7:
			return 20
		case 8:
			return 22
		case 9:
			return 24
		case 10:
			return 27
		case 11:
			return 30
		case 12:
			return 300
		default:
			return 10 + (c.Level * 2) // Fallback for any missing cases
		}
	}
	return c.Life // fallback to TOML-defined life if no level set
}

func (c *Character) GetActiveDeck() Deck {
	return c.CardCollection.GetDeck(0)
}

func (c *Character) GetDeck(index int) Deck {
	return c.CardCollection.GetDeck(index)
}

func (c *Character) GetNumDecks() int {
	maxDeck := 0
	for _, item := range c.CardCollection {
		if len(item.DeckCounts) > maxDeck {
			maxDeck = len(item.DeckCounts)
		}
	}
	return maxDeck
}

func getEmbeddedFile(filename string) []byte {
	data, err := assets.CharacterFS.ReadFile("art/screens/world/characters/" + filename)
	if err != nil {
		fmt.Printf("Error loading sprite file %s: %v\n", filename, err)
		return nil
	}
	return data
}

func (c *CharacterInstance) activeMoveSpeed() float64 {
	penalty := max(0.0, min(1.0, c.MoveSpeedPenalty))
	return c.MoveSpeed * (1.0 - penalty)
}
