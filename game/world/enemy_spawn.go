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
	maxRandomEnemyLevel      = 10
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
		maxLevel: progressionEnemyMaxLevel(l.Player, l.CombatsCompleted, l.defeatedCastleCount()),
	}

	castle := l.closestActiveCastleWithin(tile, castleSpawnInfluence)
	if castle == nil {
		return profile
	}

	profile.maxLevel = min(maxRandomEnemyLevel, profile.maxLevel+castleSpawnLevelBonus)
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
	return min(maxRandomEnemyLevel, max(baseEnemySpawnLevel, level))
}

func isWithinEnemySpawnRadius(distance float64) bool {
	return distance <= enemySpawnRadius
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
	names := weightedEnemyNames(rogues, profile, false)
	if len(names) == 0 {
		names = weightedEnemyNames(rogues, profile, true)
	}
	if len(names) == 0 {
		return "", false
	}
	return names[rng.Intn(len(names))], true
}

func weightedEnemyNames(rogues map[string]*domain.Character, profile enemySpawnProfile, ignoreProgression bool) []string {
	keys := make([]string, 0, len(rogues))
	for name := range rogues {
		keys = append(keys, name)
	}
	sort.Strings(keys)

	names := []string{}
	for _, name := range keys {
		rogue := rogues[name]
		if rogue == nil || rogue.Level <= 0 || rogue.Level > maxRandomEnemyLevel {
			continue
		}
		if !ignoreProgression && rogue.Level > profile.maxLevel {
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
