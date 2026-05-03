package domain

import "image"

type DungeonTileType int

const (
	DungeonTileWall DungeonTileType = iota
	DungeonTileEmpty
	DungeonTileEnemy
	DungeonTileTreasure
	DungeonTileDice
	DungeonTileScroll
	DungeonTileEntrance
)

type CreatureSize int

const (
	CreatureSizeSmall CreatureSize = iota
	CreatureSizeLarge
)

type DungeonRewardType int

const (
	DungeonRewardRestrictedCard DungeonRewardType = iota
	DungeonRewardGoldAmulets
)

type DiceType int

const (
	DiceAdvantage DiceType = iota
	DiceDisadvantage
)

type ClueType int

const (
	ClueLocation ClueType = iota
	CluePopulation
	ClueEffect
)

type DungeonReward struct {
	Type    DungeonRewardType
	Card    *Card
	Gold    int
	Amulets []Amulet
}

type DiceEffect struct {
	Type    DiceType
	LifeMod int
	Card    *Card
}

type ScrollQuestion struct {
	Question string
	Choices  []string
	Answer   int
}

type CardRestriction struct {
	ForbiddenColor *ColorMask
	ForbiddenType  *CardType
}

type DungeonClue struct {
	Type     ClueType
	Text     string
	Revealed bool
}

type DungeonTile struct {
	Type    DungeonTileType
	Enemy   *Character
	Reward  *DungeonReward
	Scroll  *ScrollQuestion
	Dice    *DiceEffect
	Visited bool
	Seen    bool
}

func (t *DungeonTile) IsWalkable() bool {
	if t == nil {
		return false
	}
	return t.Type != DungeonTileWall
}

func (t *DungeonTile) BlocksSight() bool {
	if t == nil {
		return true
	}
	return t.Type == DungeonTileWall || t.Type == DungeonTileEnemy
}

type Dungeon struct {
	Name            string
	Level           int
	Color           ColorMask
	Grid            [][]DungeonTile
	Entrance        image.Point
	Enchantment     *Card
	CardRestriction *CardRestriction
	CreatureSize    CreatureSize
	RestrictedCards []*Card
	MapTile         image.Point
	Cleared         bool
	Clues           [3]DungeonClue
}

func (d *Dungeon) Width() int {
	if len(d.Grid) == 0 {
		return 0
	}
	return len(d.Grid[0])
}

func (d *Dungeon) Height() int {
	return len(d.Grid)
}

func (d *Dungeon) Tile(p image.Point) *DungeonTile {
	if p.Y < 0 || p.Y >= d.Height() || p.X < 0 || p.X >= d.Width() {
		return nil
	}
	return &d.Grid[p.Y][p.X]
}

// RevealFrom marks the tile at p and every tile visible from p in the four
// cardinal directions as Seen. Rays stop after hitting a wall, an enemy, or
// the grid boundary. The blocking tile itself (wall, enemy) is also revealed
// so the player can see what is blocking their view.
func (d *Dungeon) RevealFrom(p image.Point) {
	origin := d.Tile(p)
	if origin == nil {
		return
	}
	origin.Seen = true

	dirs := [4]image.Point{{0, 1}, {0, -1}, {1, 0}, {-1, 0}}
	for _, dir := range dirs {
		cur := p
		for {
			cur = image.Point{X: cur.X + dir.X, Y: cur.Y + dir.Y}
			t := d.Tile(cur)
			if t == nil {
				break
			}
			t.Seen = true
			if t.BlocksSight() {
				break
			}
		}
	}
}

type DungeonState struct {
	CurrentDungeon *Dungeon
	Position       image.Point
	DungeonLife    int
	DiceAdvantages []DiceEffect
	CollectedCards []*Card
}

// CollectReward applies the reward at `tile` to the player and the dungeon
// state, then clears the tile back to an empty corridor. No-op if the tile is
// not a treasure or has no reward.
func (st *DungeonState) CollectReward(tile *DungeonTile, player *Player) {
	if tile == nil || tile.Type != DungeonTileTreasure || tile.Reward == nil {
		return
	}
	r := tile.Reward
	switch r.Type {
	case DungeonRewardRestrictedCard:
		if r.Card != nil {
			player.CardCollection.AddCardToDeck(r.Card, 0, 1)
			st.CollectedCards = append(st.CollectedCards, r.Card)
		}
	case DungeonRewardGoldAmulets:
		player.Gold += r.Gold
		for _, a := range r.Amulets {
			player.AddAmulet(a)
		}
	}
	tile.Reward = nil
	tile.Type = DungeonTileEmpty
}

type PlayerClues struct {
	RevealedClues map[string][]DungeonClue
}

func NewPlayerClues() *PlayerClues {
	return &PlayerClues{RevealedClues: make(map[string][]DungeonClue)}
}
