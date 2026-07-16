package domain

import (
	"fmt"
	"image"
	"maps"
	"math"
	"math/rand"
	"slices"
	"time"

	"github.com/benprew/s30/game/ui"
	"github.com/benprew/s30/game/ui/imageutil"
	"github.com/hajimehoshi/ebiten/v2"
)

type Player struct {
	Character
	CharacterInstance
	Name            string
	Gold            int
	Food            int
	MinDeckSize     int
	Amulets         map[ColorMask]int
	WorldMagics     []*WorldMagic
	ActiveDeck      int
	ActiveQuests    []*Quest // active quests (legacy delivery/defeat + deck-changing), up to MaxActiveQuests
	Days            int
	TimeAccumulator float64
	IsMale          bool
	BonusDuelLife   int
	BonusDuelCards  []*Card // One-time bonus cards that start in play in the next duel
	DungeonState    *DungeonState
}

const TicksPerDay = 5000.0

func NewPlayer(name string, visage *ebiten.Image, isM bool, difficulty Difficulty, color ColorMask) (*Player, error) {
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

	if color == ColorColorless {
		colors := []ColorMask{ColorWhite, ColorBlue, ColorBlack, ColorRed, ColorGreen}
		color = colors[rand.Intn(len(colors))]
	}
	deckGen := DeckBuilder(difficulty, color, time.Now().UnixNano())
	deck := deckGen.CreateStartingDeck()

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
		MinDeckSize: minDeckSize,
		IsMale:      isM,
		Amulets:     make(map[ColorMask]int),
		WorldMagics: make([]*WorldMagic, 0),
		ActiveDeck:  0,
	}, nil
}

// LoadImages reloads the player's sprite images. Required after deserializing
// from a save file since ebiten.Image doesn't survive JSON round-trips.
func (p *Player) LoadImages() error {
	spriteFn := "Ego_F.spr.png"
	shadowFn := "Sego_F.spr.png"
	if p.IsMale {
		spriteFn = "Ego_M.spr.png"
		shadowFn = "Sego_M.spr.png"
	}
	sprite, err := imageutil.LoadSpriteSheet(5, 8, getEmbeddedFile(spriteFn))
	if err != nil {
		return fmt.Errorf("failed to load player sprite: %w", err)
	}
	shadow, err := imageutil.LoadSpriteSheet(5, 8, getEmbeddedFile(shadowFn))
	if err != nil {
		return fmt.Errorf("failed to load player shadow: %w", err)
	}
	p.WalkingSprite = sprite
	p.ShadowSprite = shadow
	return nil
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
			for _, q := range p.ActiveQuests {
				q.DaysRemaining--
			}
			p.ExpireQuests()
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

	if ui.Pressed() {
		dirBits |= pointerMoveDirection(ui.Position(), screenW, screenH)
	}

	return dirBits
}

func pointerMoveDirection(position image.Point, screenW, screenH int) int {
	delta := position.Sub(image.Pt(screenW/2, screenH/2))
	const moveThreshold = 50
	direction := 0
	if delta.X > moveThreshold {
		direction |= DirRight
	}
	if delta.X < -moveThreshold {
		direction |= DirLeft
	}
	if delta.Y > moveThreshold {
		direction |= DirDown
	}
	if delta.Y < -moveThreshold {
		direction |= DirUp
	}
	return direction
}

// Pixel X,Y location of player (not tile)
func (p *Player) Loc() image.Point {
	return image.Point{X: p.X, Y: p.Y}
}

func (p *Player) GetDuelDeck() Deck {
	deck := p.CardCollection.GetDeck(p.ActiveDeck)

	deckSize := 0
	for _, count := range deck {
		deckSize += count
	}

	if deckSize >= p.MinDeckSize {
		return deck
	}

	landsToAdd := p.MinDeckSize - deckSize
	lands := []*Card{}
	for l := range maps.Keys(basicLands) {
		lands = append(lands, FindCardByName(l))
	}
	for range landsToAdd {
		i := rand.Intn(len(lands))
		deck[lands[i]] += 1
	}

	return deck
}

func (p *Player) RemoveCard(c *Card) error {
	return p.CardCollection.DecrementCardCount(c)
}

// ExitDungeon leaves the current dungeon, restoring the player's overworld life
// from the dungeon life pool and clearing the dungeon state. No-op when the
// player is not in a dungeon.
func (p *Player) ExitDungeon() {
	if p.DungeonState == nil {
		return
	}
	p.Life = p.DungeonState.DungeonLife
	p.DungeonState = nil
}

func (p *Player) AddAmulet(amulet Amulet) {
	p.Amulets[amulet.Color]++
}

// CanAcceptQuest reports whether the player has a free quest slot.
func (p *Player) CanAcceptQuest() bool {
	return len(p.ActiveQuests) < MaxActiveQuests
}

// AddQuest accepts a quest if there is a free slot.
func (p *Player) AddQuest(q *Quest) bool {
	if !p.CanAcceptQuest() {
		return false
	}
	p.ActiveQuests = append(p.ActiveQuests, q)
	return true
}

// HasQuest reports whether the player already holds a quest with the given
// template id (so the Wiseman avoids offering duplicates).
func (p *Player) HasQuest(id string) bool {
	for _, q := range p.ActiveQuests {
		if q.ID == id {
			return true
		}
	}
	return false
}

// RemoveQuest removes the given quest from the active quests.
func (p *Player) RemoveQuest(quest *Quest) {
	kept := p.ActiveQuests[:0]
	for _, q := range p.ActiveQuests {
		if q != quest {
			kept = append(kept, q)
		}
	}
	p.ActiveQuests = kept
}

// RedeemFulfilledQuests grants the reward for every fulfilled quest, removes
// them, and returns what was paid out. Called on entering any town — walking in
// is enough to claim a completed quest. Arriving at a delivery quest's target
// city fulfills it here, so it pays out in the same visit.
func (p *Player) RedeemFulfilledQuests(city *City) []DeckQuestReward {
	var rewards []DeckQuestReward
	kept := p.ActiveQuests[:0]
	for _, q := range p.ActiveQuests {
		if q.Type == QuestTypeDelivery && city != nil &&
			q.TargetCity != nil && q.TargetCity.Name == city.Name {
			q.IsCompleted = true
		}
		if q.IsFulfilled() {
			cards := GrantQuestReward(p, q.Reward)
			rewards = append(rewards, DeckQuestReward{Quest: q, Reward: q.Reward, Cards: cards})
			continue
		}
		kept = append(kept, q)
	}
	p.ActiveQuests = kept
	return rewards
}

// ExpireQuests drops any quest whose deadline has passed without being
// fulfilled and returns the number removed. A fulfilled-but-unredeemed quest
// survives so it can still be claimed at the next town.
func (p *Player) ExpireQuests() int {
	kept := p.ActiveQuests[:0]
	removed := 0
	for _, q := range p.ActiveQuests {
		if q.DaysRemaining <= 0 && !q.IsFulfilled() {
			removed++
			continue
		}
		kept = append(kept, q)
	}
	p.ActiveQuests = kept
	return removed
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
		for range count {
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
	return slices.Contains(p.WorldMagics, magic)
}

func (p *Player) AddWorldMagic(magic *WorldMagic) {
	if !p.HasWorldMagic(magic) {
		p.WorldMagics = append(p.WorldMagics, magic)
	}
}

func (p *Player) GetWorldMagics() []*WorldMagic {
	return p.WorldMagics
}
