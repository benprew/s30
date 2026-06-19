package world

import (
	"image"
	"math"
	"math/rand"
	"sort"

	"github.com/benprew/s30/game/domain"
)

const (
	baseEnemySpawnLevel      = 2
	castleSpawnInfluence     = 8
	castleSpawnLevelBonus    = 2
	castleColorWeight        = 4
	normalEnemyWeight        = 1
	enemySpawnRadius         = 500.0
	extraCardsPerEnemyLevel  = 10
	combatsPerEnemyLevel     = 3
	daysPerEnemyLevel        = 20
	castlesPerEnemyLevelBump = 1
	powerCardsPerEnemyLevel  = 2
)

type enemySpawnProfile struct {
	maxLevel       int
	preferredColor string
}

// EnemySpawnMaxLevelAt returns the highest enemy level appropriate to spawn
// near the given tile, factoring in player progression and nearby castles.
func (l *Level) EnemySpawnMaxLevelAt(tile image.Point) int {
	return l.enemySpawnProfileAt(tile).maxLevel
}

func (l *Level) enemySpawnProfileAt(tile image.Point) enemySpawnProfile {
	profile := enemySpawnProfile{
		maxLevel: progressionEnemyMaxLevel(l.Player, l.CombatsWon, l.defeatedCastleCount()),
	}

	castle := l.closestActiveCastleWithin(tile, castleSpawnInfluence)
	if castle == nil {
		return profile
	}

	profile.maxLevel = min(domain.MaxRandomEnemyLevel, profile.maxLevel+castleSpawnLevelBonus)
	profile.preferredColor = domain.ColorMaskToString(castle.Color)
	return profile
}

func progressionEnemyMaxLevel(player *domain.Player, combatsCompleted int, defeatedCastles int) int {
	level := baseEnemySpawnLevel
	if player != nil {
		level += max(player.NumCards()-player.MinDeckSize, 0) / extraCardsPerEnemyLevel
		level += powerfulCardProgressionLevels(player.CardCollection)
		level += player.Days / daysPerEnemyLevel
	}
	level += combatsCompleted / combatsPerEnemyLevel
	level += defeatedCastles / castlesPerEnemyLevelBump
	return min(domain.MaxRandomEnemyLevel, max(baseEnemySpawnLevel, level))
}

func isWithinEnemySpawnRadius(distance float64) bool {
	return distance <= enemySpawnRadius
}

func (l *Level) enemySpawnTiles(origin image.Point) []image.Point {
	tiles := []image.Point{}
	for y := 0; y < l.H; y++ {
		for x := 0; x < l.W; x++ {
			tileLocation := image.Point{X: x, Y: y}
			tile := l.Tile(tileLocation)
			if tile == nil || tile.TerrainType == TerrainWater {
				continue
			}

			position := l.clampLevelPixel(l.TileToPixel(tileLocation))
			dx := position.X - origin.X
			dy := position.Y - origin.Y
			distance := math.Sqrt(float64(dx*dx + dy*dy))
			if isWithinEnemySpawnRadius(distance) {
				tiles = append(tiles, tileLocation)
			}
		}
	}
	return tiles
}

func (l *Level) randomEnemySpawnPositionInTile(rng *rand.Rand, tile image.Point) image.Point {
	x := tile.X * l.TileWidth
	if tile.Y%2 != 0 {
		x += l.TileWidth / 2
	}
	y := tile.Y * l.TileHeight / 2

	width := max(l.TileWidth, 1)
	height := max(l.TileHeight/2, 1)
	return l.clampLevelPixel(image.Point{
		X: x + rng.Intn(width),
		Y: y + rng.Intn(height),
	})
}

func (l *Level) clampLevelPixel(position image.Point) image.Point {
	if levelW := l.LevelW(); levelW > 0 {
		position.X = min(max(position.X, 0), levelW-1)
	}
	if levelH := l.LevelH(); levelH > 0 {
		position.Y = min(max(position.Y, 0), levelH-1)
	}
	return position
}

func powerfulCardProgressionLevels(collection domain.CardCollection) int {
	if collection == nil {
		return 0
	}

	powerTiers := []domain.CardTier{
		domain.TierMandatory,
		domain.TierAlmostMandatory,
		domain.TierStaple,
	}
	cardPower := map[string]bool{}
	for _, tier := range powerTiers {
		for _, card := range domain.CardsByTier[tier] {
			if card != nil && card.CardType != domain.CardTypeLand {
				cardPower[card.CardName] = true
			}
		}
	}

	count := 0
	for card, item := range collection {
		if card == nil || item == nil || !cardPower[card.CardName] {
			continue
		}
		count += item.Count
	}
	return count / powerCardsPerEnemyLevel
}

func chooseEnemyName(rng *rand.Rand, rogues map[string]*domain.Character, profile enemySpawnProfile) (string, bool) {
	names := weightedEnemyNames(rogues, profile)
	return names[rng.Intn(len(names))], true
}

func weightedEnemyNames(rogues map[string]*domain.Character, profile enemySpawnProfile) []string {
	keys := make([]string, 0, len(rogues))
	for name := range rogues {
		keys = append(keys, name)
	}
	sort.Strings(keys)

	names := []string{}
	for _, name := range keys {
		rogue := rogues[name]
		if rogue == nil || rogue.Level > profile.maxLevel {
			continue
		}

		weight := normalEnemyWeight
		if profile.preferredColor != "" && rogue.PrimaryColor == profile.preferredColor {
			weight = castleColorWeight
		}
		for range weight {
			names = append(names, name)
		}
	}
	return names
}

func (l *Level) closestActiveCastleWithin(tile image.Point, maxDistance int) *domain.Castle {
	var closest *domain.Castle
	closestDistance := math.MaxInt
	for _, castle := range l.Castles {
		if castle == nil || castle.Defeated {
			continue
		}
		distance := absInt(castle.MapTile.X-tile.X) + absInt(castle.MapTile.Y-tile.Y)
		if distance <= maxDistance && distance < closestDistance {
			closest = castle
			closestDistance = distance
		}
	}
	return closest
}

func (l *Level) defeatedCastleCount() int {
	count := 0
	for _, castle := range l.Castles {
		if castle != nil && castle.Defeated {
			count++
		}
	}
	return count
}
