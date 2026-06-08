package domain

import (
	"fmt"
	"image"
	"math/rand"
	"strings"
)

type DungeonGenOptions struct {
	Name            string
	Level           int
	Color           ColorMask
	CreatureSize    CreatureSize
	Enchantment     *Card
	CardRestriction *CardRestriction
	RestrictedCards []*Card
	EnemyPool       []*Character
	DiceCardPool    []*Card
	GridSize        int
	NumEnemies      int
	NumDice         int
	NumScrolls      int
	NumGoldChests   int
	Seed            int64
}

// GenerateDungeon builds a Dungeon with a carved hallway grid and event tiles
// placed across enemies, treasures, dice, and scrolls. The MapTile field is
// left at the zero value; callers are expected to assign it during world
// placement.
func GenerateDungeon(opts DungeonGenOptions) *Dungeon {
	size := opts.GridSize
	if size <= 0 {
		size = 11
	}
	if size%2 == 0 {
		size++
	}

	rng := rand.New(rand.NewSource(opts.Seed))

	d := &Dungeon{
		Name:            opts.Name,
		Level:           opts.Level,
		Color:           opts.Color,
		CreatureSize:    opts.CreatureSize,
		Enchantment:     opts.Enchantment,
		CardRestriction: opts.CardRestriction,
		RestrictedCards: opts.RestrictedCards,
		Grid:            makeWalls(size, size),
	}

	entrance := image.Point{X: 1, Y: size - 2}
	d.Entrance = entrance
	carveMaze(d.Grid, entrance, rng)
	d.Tile(entrance).Type = DungeonTileEntrance

	emptyTiles := collectByType(d.Grid, DungeonTileEmpty)
	deadEnds := findDeadEnds(d.Grid)

	rng.Shuffle(len(emptyTiles), func(i, j int) {
		emptyTiles[i], emptyTiles[j] = emptyTiles[j], emptyTiles[i]
	})
	rng.Shuffle(len(deadEnds), func(i, j int) {
		deadEnds[i], deadEnds[j] = deadEnds[j], deadEnds[i]
	})

	used := map[image.Point]bool{entrance: true}
	available := func(p image.Point) bool {
		return !used[p] && p != entrance
	}

	// Restricted-card chests get the dead-ends farthest from the entrance
	// so the player has to traverse the dungeon to claim them.
	sortDeadEndsByDistance(d.Grid, entrance, deadEnds)
	for _, card := range opts.RestrictedCards {
		p, ok := takeFirst(deadEnds, available)
		if !ok {
			break
		}
		t := d.Tile(p)
		t.Type = DungeonTileTreasure
		t.Reward = &DungeonReward{Type: DungeonRewardRestrictedCard, Card: card}
		used[p] = true
	}

	for range opts.NumGoldChests {
		p, ok := takeFirst(deadEnds, available)
		if !ok {
			break
		}
		t := d.Tile(p)
		t.Type = DungeonTileTreasure
		t.Reward = &DungeonReward{
			Type: DungeonRewardGoldAmulets,
			Gold: 50 + rng.Intn(150),
		}
		used[p] = true
	}

	placeOnEmpty := func(count int, set func(*DungeonTile, *rand.Rand)) {
		for range count {
			p, ok := takeFirst(emptyTiles, available)
			if !ok {
				return
			}
			set(d.Tile(p), rng)
			used[p] = true
		}
	}

	placeOnEmpty(opts.NumEnemies, func(t *DungeonTile, r *rand.Rand) {
		t.Type = DungeonTileEnemy
		if len(opts.EnemyPool) > 0 {
			t.Enemy = opts.EnemyPool[r.Intn(len(opts.EnemyPool))]
		}
	})

	placeOnEmpty(opts.NumDice, func(t *DungeonTile, r *rand.Rand) {
		t.Type = DungeonTileDice
		t.Dice = rollDiceEffect(opts.Level, r, opts.DiceCardPool)
	})

	placeOnEmpty(opts.NumScrolls, func(t *DungeonTile, r *rand.Rand) {
		t.Type = DungeonTileScroll
		t.Scroll = &ScrollQuestion{}
	})

	return d
}

func makeWalls(w, h int) [][]DungeonTile {
	g := make([][]DungeonTile, h)
	for y := range h {
		g[y] = make([]DungeonTile, w)
		for x := range w {
			g[y][x] = DungeonTile{Type: DungeonTileWall}
		}
	}
	return g
}

// carveMaze runs a recursive backtracker from `start`, stepping two cells at a
// time and knocking down the wall in between. Cells at odd (x, y) become the
// hallway grid; even-indexed cells stay as walls or get carved as connectors.
func carveMaze(g [][]DungeonTile, start image.Point, rng *rand.Rand) {
	h := len(g)
	w := len(g[0])
	g[start.Y][start.X].Type = DungeonTileEmpty

	type frame struct{ pos image.Point }
	stack := []frame{{start}}

	dirs := []image.Point{{0, -2}, {2, 0}, {0, 2}, {-2, 0}}

	for len(stack) > 0 {
		top := stack[len(stack)-1]
		shuffled := append([]image.Point(nil), dirs...)
		rng.Shuffle(len(shuffled), func(i, j int) {
			shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
		})

		carved := false
		for _, d := range shuffled {
			next := image.Point{X: top.pos.X + d.X, Y: top.pos.Y + d.Y}
			if next.X <= 0 || next.X >= w-1 || next.Y <= 0 || next.Y >= h-1 {
				continue
			}
			if g[next.Y][next.X].Type != DungeonTileWall {
				continue
			}
			between := image.Point{X: top.pos.X + d.X/2, Y: top.pos.Y + d.Y/2}
			g[between.Y][between.X].Type = DungeonTileEmpty
			g[next.Y][next.X].Type = DungeonTileEmpty
			stack = append(stack, frame{next})
			carved = true
			break
		}
		if !carved {
			stack = stack[:len(stack)-1]
		}
	}
}

func collectByType(g [][]DungeonTile, t DungeonTileType) []image.Point {
	var out []image.Point
	for y := range g {
		for x := range g[y] {
			if g[y][x].Type == t {
				out = append(out, image.Point{X: x, Y: y})
			}
		}
	}
	return out
}

func findDeadEnds(g [][]DungeonTile) []image.Point {
	var out []image.Point
	for y := range g {
		for x := range g[y] {
			if g[y][x].Type != DungeonTileEmpty {
				continue
			}
			open := 0
			for _, d := range [4]image.Point{{0, 1}, {0, -1}, {1, 0}, {-1, 0}} {
				ny, nx := y+d.Y, x+d.X
				if ny < 0 || ny >= len(g) || nx < 0 || nx >= len(g[0]) {
					continue
				}
				if g[ny][nx].Type != DungeonTileWall {
					open++
				}
			}
			if open == 1 {
				out = append(out, image.Point{X: x, Y: y})
			}
		}
	}
	return out
}

// distancesFrom runs a BFS from `start` over all non-wall tiles. Unreachable
// tiles are -1.
func distancesFrom(g [][]DungeonTile, start image.Point) [][]int {
	h := len(g)
	w := len(g[0])
	dist := make([][]int, h)
	for y := range dist {
		dist[y] = make([]int, w)
		for x := range dist[y] {
			dist[y][x] = -1
		}
	}
	dist[start.Y][start.X] = 0
	queue := []image.Point{start}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, d := range [4]image.Point{{0, 1}, {0, -1}, {1, 0}, {-1, 0}} {
			n := image.Point{X: cur.X + d.X, Y: cur.Y + d.Y}
			if n.X < 0 || n.X >= w || n.Y < 0 || n.Y >= h {
				continue
			}
			if g[n.Y][n.X].Type == DungeonTileWall {
				continue
			}
			if dist[n.Y][n.X] != -1 {
				continue
			}
			dist[n.Y][n.X] = dist[cur.Y][cur.X] + 1
			queue = append(queue, n)
		}
	}
	return dist
}

func sortDeadEndsByDistance(g [][]DungeonTile, start image.Point, points []image.Point) {
	dist := distancesFrom(g, start)
	sortByDescending(points, func(p image.Point) int {
		return dist[p.Y][p.X]
	})
}

func sortByDescending(points []image.Point, key func(image.Point) int) {
	for i := 1; i < len(points); i++ {
		for j := i; j > 0 && key(points[j]) > key(points[j-1]); j-- {
			points[j], points[j-1] = points[j-1], points[j]
		}
	}
}

func takeFirst(slice []image.Point, ok func(image.Point) bool) (image.Point, bool) {
	for _, p := range slice {
		if ok(p) {
			return p, true
		}
	}
	return image.Point{}, false
}

// rollDiceEffect rolls a single dice tile's effect for a dungeon of the given
// level and color. Apprentice (level 1) dungeons never roll a disadvantage;
// higher levels occasionally do. Advantages are either bonus life (more on
// easier levels) or a free card that starts the next duel in play (powerful
// creatures on easier levels, weaker creatures or mana on harder ones).
func rollDiceEffect(level int, rng *rand.Rand, cardPool []*Card) *DiceEffect {
	allowDisadvantage := level >= 2
	if allowDisadvantage && rng.Intn(4) == 0 {
		return &DiceEffect{Type: DiceDisadvantage, LifeMod: -(1 + rng.Intn(3))}
	}

	// Half the advantages are a free card, half are bonus life.
	if rng.Intn(2) == 0 {
		if card := randomDiceCard(rng, cardPool); card != nil {
			return &DiceEffect{Type: DiceAdvantage, Card: card}
		}
	}

	// Easier dungeons give more life (~3-4), harder ones less (~1-2).
	bonus := max(4-level, 1)
	return &DiceEffect{Type: DiceAdvantage, LifeMod: bonus + rng.Intn(2)}
}

// randomDiceCard picks the free card granted by a dice advantage from the
// player's own deck (cardPool, which the caller supplies with lands already
// excluded). Drawing from the deck guarantees the grant is a real, playable
// card the duel can put into play. Returns nil when the pool is empty, in which
// case the caller falls back to a life bonus.
func randomDiceCard(rng *rand.Rand, cardPool []*Card) *Card {
	eligible := make([]*Card, 0, len(cardPool))
	for _, c := range cardPool {
		if !canStartOnEmptyBattlefieldFromDice(c) {
			continue
		}
		eligible = append(eligible, c)
	}
	if len(eligible) == 0 {
		return nil
	}
	return eligible[rng.Intn(len(eligible))]
}

func canStartOnEmptyBattlefieldFromDice(card *Card) bool {
	if card == nil || isAuraLike(card) {
		return false
	}

	switch card.CardType {
	case CardTypeCreature, CardTypeArtifact, CardTypeEnchantment:
		return true
	default:
		return false
	}
}

func isAuraLike(card *Card) bool {
	if containsCaseInsensitive(card.Subtypes, "Aura") || containsCaseInsensitive(card.Keywords, "Enchant") {
		return true
	}
	if strings.Contains(strings.ToLower(card.TypeLine), "aura") {
		return true
	}
	for _, ability := range card.ParsedAbilities {
		if ability.TargetSpec != nil && strings.EqualFold(ability.TargetSpec.Condition, "enchant") {
			return true
		}
		if strings.HasPrefix(strings.ToLower(ability.RawText), "enchant ") {
			return true
		}
	}
	return false
}

func containsCaseInsensitive(values []string, want string) bool {
	for _, value := range values {
		if strings.EqualFold(value, want) {
			return true
		}
	}
	return false
}

// AllRestrictedCardsReachable verifies that every restricted-card treasure
// tile can be reached from the entrance by walking through empty/event tiles
// (enemies count as walkable since the player can defeat them). This is a
// post-condition the generator should satisfy.
func (d *Dungeon) AllRestrictedCardsReachable() error {
	dist := distancesFrom(d.Grid, d.Entrance)
	for y := range d.Grid {
		for x := range d.Grid[y] {
			t := &d.Grid[y][x]
			if t.Type != DungeonTileTreasure || t.Reward == nil {
				continue
			}
			if t.Reward.Type != DungeonRewardRestrictedCard {
				continue
			}
			if dist[y][x] < 0 {
				return fmt.Errorf("restricted card at (%d,%d) is unreachable from entrance", x, y)
			}
		}
	}
	return nil
}
