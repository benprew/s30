package domain

import (
	"fmt"
	"image"
	"time"

	"github.com/benprew/s30/game/ui"
	"github.com/benprew/s30/game/ui/imageutil"
	"github.com/hajimehoshi/ebiten/v2"
)

type Player struct {
	Character
	CharacterInstance
	Name           string
	Gold           int
	Food           int
	CardCollection Deck
	Amulets        map[ColorMask]int
	WorldMagics    []*WorldMagic
	// ActiveDeck     int
	// Decks          []Deck
}

func NewPlayer(name string, visage *ebiten.Image, isM bool) (*Player, error) {
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

	deckGen := NewDeckGenerator(DifficultyEasy, ColorRed, time.Now().UnixNano())
	deck := deckGen.GenerateStartingDeck()

	c := Character{
		Visage:        visage,
		WalkingSprite: sprite,
		ShadowSprite:  shadow,
		Life:          8,
		Deck:          deck,
	}

	cardCollection := make(map[*Card]int)
	for card := range deck {
		cardCollection[card]++
	}

	return &Player{
		Character: c,
		CharacterInstance: CharacterInstance{
			MoveSpeed: 10,
		},
		Name:           string(name),
		Gold:           1200,
		Food:           30,
		CardCollection: cardCollection,
		Amulets:        make(map[ColorMask]int),
		WorldMagics:    make([]*WorldMagic, 0),
	}, nil
}

func (p *Player) Draw(screen *ebiten.Image, options *ebiten.DrawImageOptions) {
	screen.DrawImage(p.ShadowSprite[p.Direction][p.Frame], options)
	screen.DrawImage(p.WalkingSprite[p.Direction][p.Frame], options)
}

func (p *Player) NumCards() (cnt int) {
	for _, v := range p.CardCollection {
		cnt += v
	}
	return cnt
}

func (p *Player) Update(screenW, screenH, levelW, levelH int) error {
	dirBits := p.Move(screenW, screenH)
	p.CharacterInstance.Update(dirBits)

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

func (p *Player) RemoveCard(c *Card) error {
	cnt := p.CardCollection[c]
	if cnt < 1 {
		return fmt.Errorf("Card not in collection: %s", c.Name())
	}
	p.CardCollection[c]--
	p.Deck[c]--
	return nil
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
