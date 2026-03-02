package domain

import (
	"fmt"
	"image"
	"math"
	"math/rand"
	"time"

	"github.com/benprew/s30/game/ui"
	"github.com/benprew/s30/game/ui/imageutil"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type Player struct {
	Character
	CharacterInstance
	Name            string
	Gold            int
	Food            int
	Difficulty      Difficulty
	MinDeckSize     int
	Amulets         map[ColorMask]int
	WorldMagics     []*WorldMagic
	ActiveDeck      int
	ActiveQuest     *Quest
	Days            int
	TimeAccumulator float64
	mouseMoving     bool
}

const TicksPerDay = 5000.0

func NewPlayer(name string, visage *ebiten.Image, isM bool, difficulty Difficulty) (*Player, error) {
	sprite, err := imageutil.LoadSpriteSheet(5, 8, getEmbeddedFile("Ego_F.spr.png"))
	if err != nil {
		return nil, err
	}
	shadow, err := imageutil.LoadSpriteSheet(5, 8, getEmbeddedFile("Sego_F.spr.png"))
	if err != nil {
		return nil, err
	}

	if isM {
		sprite, err = imageutil.LoadSpriteSheet(5, 8, getEmbeddedFile("Ego_M.spr.png"))
		if err != nil {
			return nil, err
		}
		shadow, err = imageutil.LoadSpriteSheet(5, 8, getEmbeddedFile("Sego_M.spr.png"))
		if err != nil {
			return nil, err
		}
	}

	colors := []ColorMask{ColorWhite, ColorBlue, ColorBlack, ColorRed, ColorGreen}
	color := colors[rand.Intn(len(colors))]
	deckGen := NewDeckGenerator(difficulty, color, time.Now().UnixNano())
	deck := deckGen.GenerateStartingDeck()

	cardCollection := NewCardCollection()
	for card, count := range deck {
		cardCollection.AddCardToDeck(card, 0, count)
	}

	var gold, food, minDeckSize int
	switch difficulty {
	case DifficultyEasy:
		gold = 250
		food = 50
		minDeckSize = 30
	case DifficultyMedium:
		gold = 200
		food = 50
		minDeckSize = 35
	case DifficultyHard:
		gold = 150
		food = 50
		minDeckSize = 40
	case DifficultyExpert:
		gold = 100
		food = 50
		minDeckSize = 40
	}

	c := Character{
		Visage:         visage,
		WalkingSprite:  sprite,
		ShadowSprite:   shadow,
		Life:           10,
		CardCollection: cardCollection,
	}

	return &Player{
		Character: c,
		CharacterInstance: CharacterInstance{
			MoveSpeed: 10,
		},
		Name:        string(name),
		Gold:        gold,
		Food:        food,
		Difficulty:  difficulty,
		MinDeckSize: minDeckSize,
		Amulets:     make(map[ColorMask]int),
		WorldMagics: make([]*WorldMagic, 0),
		ActiveDeck:  0,
	}, nil
}

func (p *Player) Draw(screen *ebiten.Image, options *ebiten.DrawImageOptions) {
	screen.DrawImage(p.ShadowSprite[p.Direction][p.Frame], options)
	screen.DrawImage(p.WalkingSprite[p.Direction][p.Frame], options)
}

func (p *Player) NumCards() int {
	return p.CardCollection.NumCards()
}

func (p *Player) Update(screenW, screenH, levelW, levelH int) error {
	oldX, oldY := p.X, p.Y
	dirBits := p.Move(screenW, screenH)
	p.CharacterInstance.Update(dirBits)

	if p.X != oldX || p.Y != oldY {
		// Player moved
		dist := math.Sqrt(math.Pow(float64(p.X-oldX), 2) + math.Pow(float64(p.Y-oldY), 2))
		p.TimeAccumulator += dist
		if p.TimeAccumulator >= TicksPerDay {
			p.TimeAccumulator -= TicksPerDay
			p.Days++
			if p.ActiveQuest != nil {
				p.ActiveQuest.DaysRemaining--
			}
		}
	}

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

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		p.mouseMoving = true
	}
	if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		p.mouseMoving = false
	}

	var cursorX, cursorY = ui.TouchPosition()
	if p.mouseMoving {
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

// Pixel X,Y location of player (not tile)
func (p *Player) Loc() image.Point {
	return image.Point{X: p.X, Y: p.Y}
}

func (p *Player) RemoveCard(c *Card) error {
	return p.CardCollection.DecrementCardCount(c)
}

func (p *Player) AddAmulet(amulet Amulet) {
	p.Amulets[amulet.Color]++
}

func (p *Player) HasAmulet(color ColorMask) bool {
	return p.Amulets[color] > 0
}

func (p *Player) GetAmuletCount() map[ColorMask]int {
	return p.Amulets
}

func (p *Player) GetAmulets() []Amulet {
	amulets := make([]Amulet, 0)
	for color, count := range p.Amulets {
		for i := 0; i < count; i++ {
			amulets = append(amulets, NewAmulet(color))
		}
	}
	return amulets
}

func (p *Player) RemoveAmulet(color ColorMask) error {
	if p.Amulets[color] < 1 {
		return fmt.Errorf("no amulet of color %s to remove", ColorMaskToString(color))
	}
	p.Amulets[color]--
	if p.Amulets[color] == 0 {
		delete(p.Amulets, color)
	}
	return nil
}

func (p *Player) HasWorldMagic(magic *WorldMagic) bool {
	for _, m := range p.WorldMagics {
		if m == magic {
			return true
		}
	}
	return false
}

func (p *Player) AddWorldMagic(magic *WorldMagic) {
	if !p.HasWorldMagic(magic) {
		p.WorldMagics = append(p.WorldMagics, magic)
	}
}

func (p *Player) GetWorldMagics() []*WorldMagic {
	return p.WorldMagics
}
