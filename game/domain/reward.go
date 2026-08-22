package domain

import (
	"math/rand"
)

// DuelReward represents the complete bundle of rewards granted upon winning a duel.
type DuelReward struct {
	Cards   []*Card
	Gold    int
	Amulets []Amulet
}

var dualLands = map[string]ColorMask{
	"Badlands":        ColorBlack | ColorRed,
	"Bayou":           ColorBlack | ColorGreen,
	"Plateau":         ColorRed | ColorWhite,
	"Savannah":        ColorGreen | ColorWhite,
	"Scrubland":       ColorBlack | ColorWhite,
	"Taiga":           ColorGreen | ColorRed,
	"Tropical Island": ColorGreen | ColorBlue,
	"Tundra":          ColorBlue | ColorWhite,
	"Underground Sea": ColorBlack | ColorBlue,
	"Volcanic Island": ColorRed | ColorBlue,
}

// IsBasicLand reports whether the card is one of the 5 basic lands.
func IsBasicLand(c *Card) bool {
	if c == nil {
		return false
	}
	_, ok := basicLands[c.CardName]
	return ok
}

// IsDualLand reports whether the card is an original dual land.
func IsDualLand(c *Card) bool {
	if c == nil {
		return false
	}
	_, ok := dualLands[c.CardName]
	return ok
}

// IsNonBasicLand reports whether the card is a land that is not a basic land.
func IsNonBasicLand(c *Card) bool {
	if c == nil || c.CardType != CardTypeLand {
		return false
	}
	return !IsBasicLand(c)
}

// DeckPrimaryColor returns the color that appears most often across the deck's
// cards, weighted by how many copies of each card the deck runs. A deck with no
// colored cards returns ColorColorless.
func DeckPrimaryColor(deck Deck) ColorMask {
	counts := map[ColorMask]int{}
	for card, n := range deck {
		for _, s := range card.ColorIdentity {
			if m, ok := colorStringToMask[s]; ok {
				counts[m] += n
			}
		}
	}

	best := ColorColorless
	bestCount := 0
	for _, m := range []ColorMask{ColorWhite, ColorBlue, ColorBlack, ColorRed, ColorGreen} {
		if counts[m] > bestCount {
			bestCount = counts[m]
			best = m
		}
	}
	return best
}

// RandomBasicLand returns a basic land card of a random color, or nil if the
// card database contains none.
func RandomBasicLand() *Card {
	names := make([]string, 0, len(basicLands))
	for name := range basicLands {
		names = append(names, name)
	}
	rand.Shuffle(len(names), func(i, j int) { names[i], names[j] = names[j], names[i] })
	for _, name := range names {
		if card := FindCardByName(name); card != nil {
			return card
		}
	}
	return nil
}

// RandomDualLand returns a dual land card, prioritizing ones that match the given
// color identity if provided.
func RandomDualLand(color ColorMask) *Card {
	var preferred, all []string
	for name, mask := range dualLands {
		all = append(all, name)
		if color != ColorColorless && color&mask != 0 {
			preferred = append(preferred, name)
		}
	}
	pool := preferred
	if len(pool) == 0 {
		pool = all
	}
	rand.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	for _, name := range pool {
		if card := FindCardByName(name); card != nil {
			return card
		}
	}
	return nil
}

// RandomNonBasicLand returns a non-basic land card (utility land or dual land),
// prioritizing lands that match the given color or produce mana for it. Restricted
// lands (e.g. Library of Alexandria, Strip Mine) only appear very rarely.
func RandomNonBasicLand(color ColorMask) *Card {
	allowRestricted := rand.Float64() < RestrictedRewardChance
	var preferred, all []*Card
	var preferredRestricted, allRestricted []*Card
	seen := make(map[string]bool)
	for _, c := range CARDS {
		if !IsNonBasicLand(c) || seen[c.CardName] {
			continue
		}
		seen[c.CardName] = true
		if c.VintageRestricted {
			allRestricted = append(allRestricted, c)
			if color != ColorColorless && cardMatchesColorOrColorless(c, color) {
				preferredRestricted = append(preferredRestricted, c)
			}
		} else {
			all = append(all, c)
			if color != ColorColorless && cardMatchesColorOrColorless(c, color) {
				preferred = append(preferred, c)
			}
		}
	}

	if allowRestricted && len(allRestricted) > 0 {
		pool := preferredRestricted
		if len(pool) == 0 {
			pool = allRestricted
		}
		return pool[rand.Intn(len(pool))]
	}

	pool := preferred
	if len(pool) == 0 {
		pool = all
	}
	if len(pool) == 0 {
		return RandomBasicLand()
	}
	return pool[rand.Intn(len(pool))]
}

// RandomLandForTier picks a random land appropriate for the enemy tier. Higher
// tiers have higher chances of returning non-basic and dual lands.
func RandomLandForTier(tier int, color ColorMask) *Card {
	roll := rand.Float64()
	switch {
	case tier <= 2:
		return RandomBasicLand()
	case tier == 3:
		if roll < 0.20 {
			return RandomNonBasicLand(color)
		}
		return RandomBasicLand()
	case tier == 4:
		if roll < 0.10 {
			return RandomDualLand(color)
		} else if roll < 0.40 {
			return RandomNonBasicLand(color)
		}
		return RandomBasicLand()
	case tier <= 6:
		if roll < 0.25 {
			return RandomDualLand(color)
		} else if roll < 0.60 {
			return RandomNonBasicLand(color)
		}
		return RandomBasicLand()
	case tier <= 8:
		if roll < 0.40 {
			return RandomDualLand(color)
		} else if roll < 0.80 {
			return RandomNonBasicLand(color)
		}
		return RandomBasicLand()
	default: // Tier 9+ (including Tier 10 and 11)
		if roll < 0.45 {
			return RandomDualLand(color)
		} else if roll < 0.90 {
			return RandomNonBasicLand(color)
		}
		return RandomBasicLand()
	}
}

// GenerateDuelReward creates a complete reward package for winning a duel,
// scaling cards, lands, gold, and amulets based on the enemy's tier and color.
// Tier 11+ enemies cap at Tier 10 rewards because they grant separate castle/boss bonuses.
func GenerateDuelReward(playerDeck Deck, opponentAnte *Card, enemyTier int, enemyColor ColorMask) DuelReward {
	playerColor := DeckPrimaryColor(playerDeck)
	if playerColor == ColorColorless {
		playerColor = ColorAny
	}

	var cards []*Card
	if opponentAnte != nil {
		cards = append(cards, opponentAnte)
	}

	var randomCards []*Card
	var landCount int

	switch {
	case enemyTier <= 2:
		randomCards = RandomLowCardsForColor(playerColor, 1)
		landCount = 1
	case enemyTier == 3:
		randomCards = RandomLowCardsForColor(playerColor, 2)
		landCount = 1
	case enemyTier == 4:
		randomCards = RandomLowMidCardsForColor(playerColor, 2)
		landCount = 1
	case enemyTier <= 6:
		randomCards = RandomMidCardsForColor(playerColor, 2)
		landCount = 2
	case enemyTier == 7:
		randomCards = RandomMidCardsForColor(playerColor, 3)
		landCount = 2
	case enemyTier <= 9:
		randomCards = append(randomCards, RandomMidCardsForColor(playerColor, 2)...)
		randomCards = append(randomCards, RandomHighCardsForColor(playerColor, 1)...)
		landCount = 2
	default: // Tier 10+
		randomCards = append(randomCards, RandomMidCardsForColor(playerColor, 1)...)
		randomCards = append(randomCards, RandomHighCardsForColor(playerColor, 2)...)
		landCount = 2
	}

	cards = append(cards, randomCards...)

	for range landCount {
		if land := RandomLandForTier(enemyTier, playerColor); land != nil {
			cards = append(cards, land)
		}
	}

	gold := generateDuelGold(enemyTier)
	amulets := generateDuelAmulets(enemyTier, enemyColor)

	return DuelReward{
		Cards:   cards,
		Gold:    gold,
		Amulets: amulets,
	}
}

func generateDuelGold(tier int) int {
	roll := rand.Float64()
	switch {
	case tier <= 2:
		if roll < 0.50 {
			return 15 + rand.Intn(16)
		}
	case tier == 3:
		if roll < 0.60 {
			return 30 + rand.Intn(21)
		}
	case tier == 4:
		if roll < 0.65 {
			return 40 + rand.Intn(31)
		}
	case tier <= 6:
		if roll < 0.75 {
			return 50 + rand.Intn(51)
		}
	case tier == 7:
		if roll < 0.80 {
			return 70 + rand.Intn(61)
		}
	case tier <= 9:
		if roll < 0.90 {
			return 100 + rand.Intn(81)
		}
	default: // Tier 10+
		return 150 + rand.Intn(101)
	}
	return 0
}

func generateDuelAmulets(tier int, enemyColor ColorMask) []Amulet {
	amuletColor := pickAmuletColor(enemyColor)
	roll := rand.Float64()
	count := 0

	switch {
	case tier <= 2:
		if roll < 0.15 {
			count = 1
		}
	case tier == 3:
		if roll < 0.25 {
			count = 1
		}
	case tier == 4:
		if roll < 0.30 {
			count = 1
		}
	case tier <= 6:
		if roll < 0.45 {
			count = 1
		}
	case tier == 7:
		if roll < 0.60 {
			count = 1
		}
	case tier <= 9:
		if roll < 0.75 {
			count = 1
		}
	default: // Tier 10+
		if roll < 0.85 {
			count = 1
			if rand.Float64() < 0.20 {
				count = 2
			}
		}
	}

	if count == 0 {
		return nil
	}

	amulets := make([]Amulet, count)
	for i := range count {
		amulets[i] = NewAmulet(amuletColor)
	}
	return amulets
}

func pickAmuletColor(enemyColor ColorMask) ColorMask {
	var validColors []ColorMask
	for _, c := range GetAllAmuletColors() {
		if enemyColor&c != 0 {
			validColors = append(validColors, c)
		}
	}
	if len(validColors) > 0 {
		return validColors[rand.Intn(len(validColors))]
	}
	all := GetAllAmuletColors()
	return all[rand.Intn(len(all))]
}

// DuelRewards returns the list of reward cards awarded after winning a duel
// against an enemy of the specified tier.
func DuelRewards(playerDeck Deck, opponentAnte *Card, enemyTier int) []*Card {
	return GenerateDuelReward(playerDeck, opponentAnte, enemyTier, ColorColorless).Cards
}

// RewardChoices returns the cards awarded after winning a duel against a tier 1
// opponent (compatibility wrapper).
func RewardChoices(playerDeck Deck, opponentCard *Card) []*Card {
	return DuelRewards(playerDeck, opponentCard, 1)
}
