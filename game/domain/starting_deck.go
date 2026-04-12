package domain

import (
	"math/rand"
)

type ColorMask int

const (
	ColorColorless ColorMask = 0
	ColorWhite     ColorMask = 1 << 0
	ColorBlue      ColorMask = 1 << 1
	ColorBlack     ColorMask = 1 << 2
	ColorRed       ColorMask = 1 << 3
	ColorGreen     ColorMask = 1 << 4
)

var basicLands = map[string]ColorMask{
	"Plains":   ColorWhite,
	"Island":   ColorBlue,
	"Swamp":    ColorBlack,
	"Mountain": ColorRed,
	"Forest":   ColorGreen,
}

var colorStringToMask = map[string]ColorMask{
	"W": ColorWhite,
	"U": ColorBlue,
	"B": ColorBlack,
	"R": ColorRed,
	"G": ColorGreen,
}

type Difficulty int

const (
	DifficultyEasy   Difficulty = 0
	DifficultyMedium Difficulty = 1
	DifficultyHard   Difficulty = 2
	DifficultyExpert Difficulty = 3
)

type DeckGenerator struct {
	deck        Deck
	difficulty  Difficulty
	playerColor ColorMask
	rng         *rand.Rand
	// weakPool is the set of non-land cards the starting deck may draw from:
	// the three lowest tiers from card_tiers.toml. Scoped to the generator so
	// tests can substitute a custom pool.
	weakPool []*Card
}

func DeckBuilder(difficulty Difficulty, playerColor ColorMask, seed int64) *DeckGenerator {
	return &DeckGenerator{
		deck:        Deck{},
		difficulty:  difficulty,
		playerColor: playerColor,
		rng:         rand.New(rand.NewSource(seed)),
		weakPool: CardsInTiers(
			TierRarelyPlayed,
			TierAlmostNeverPlayed,
			TierMeme,
		),
	}
}

func (dg *DeckGenerator) CreateStartingDeck() Deck {
	dg.deck = Deck{}

	primaryColor := dg.playerColor

	switch dg.difficulty {
	case DifficultyEasy:
		if primaryColor < ColorWhite {
			dg.generateRandomDeck(primaryColor, 13, 7, 15, true)
		} else {
			dg.generateRandomDeck(primaryColor, 13, 12, 10, true)
		}

	case DifficultyMedium:
		dg.generateRandomDeck(primaryColor, 11, 4, 12, true)
		secondaryColor := dg.pickRandomColorOtherThan(primaryColor)
		dg.generateRandomDeck(secondaryColor, 4, 3, 4, true)

	case DifficultyHard:
		dg.generateRandomDeck(primaryColor, 9, 3, 9, true)
		secondaryColor := dg.pickRandomColorOtherThan(primaryColor)
		dg.generateRandomDeck(secondaryColor, 5, 3, 4, true)
		tertiaryColor := dg.pickRandomColorOtherThan(primaryColor | secondaryColor)
		dg.generateRandomDeck(tertiaryColor, 4, 3, 3, true)

	case DifficultyExpert:
		dg.generateRandomDeck(primaryColor, 6, 3, 5, true)
		dg.generateRandomDeck(ColorColorless, 11, 5, 14, true)
	}

	return dg.deck
}

// maxPickAttempts bounds how many times we'll retry picking a card for a slot
// before giving up and leaving the slot empty. The weak-tier pool is small
// (~70 cards) and some color/type combinations may have zero candidates, so a
// bound is required to avoid an infinite loop.
const maxPickAttempts = 500

func (dg *DeckGenerator) generateRandomDeck(
	color ColorMask,
	numBasicLands int,
	numEnchantmentsAndArtifacts int,
	numCreatures int,
	allowArtifacts bool,
) {
	for i := 0; i < numBasicLands; i++ {
		card := dg.pickBasicLand(color)
		if card != nil {
			dg.addCardToDeck(card)
		} else {
			break
		}
	}

	for i := 0; i < numEnchantmentsAndArtifacts; i++ {
		var cardColor ColorMask
		var cardType CardType

		if allowArtifacts && dg.rng.Intn(2) == 1 {
			cardColor = ColorColorless
			cardType = CardTypeArtifact
		} else {
			cardColor = color
			cardType = CardTypeEnchantment
		}

		card := dg.pickWeakCard([]CardType{cardType}, cardColor)
		if card == nil {
			continue
		}
		dg.addCardToDeck(card)
	}

	for i := 0; i < numCreatures; i++ {
		card := dg.pickWeakCard([]CardType{CardTypeCreature}, color)
		if card == nil {
			continue
		}
		dg.addCardToDeck(card)
	}
}

func (dg *DeckGenerator) pickRandomColorOtherThan(excludeColors ColorMask) ColorMask {
	availableColors := []ColorMask{
		ColorWhite,
		ColorBlue,
		ColorBlack,
		ColorRed,
		ColorGreen,
	}

	var validColors []ColorMask
	for _, color := range availableColors {
		if excludeColors&color == 0 {
			validColors = append(validColors, color)
		}
	}

	if len(validColors) == 0 {
		return ColorColorless
	}

	return validColors[dg.rng.Intn(len(validColors))]
}

// pickBasicLand picks a random basic land matching the requested color.
// Basic lands are not in the tier pool, so we scan the full card database.
func (dg *DeckGenerator) pickBasicLand(color ColorMask) *Card {
	var candidates []*Card
	for _, card := range CARDS {
		if dg.isBasicLandOfColor(card, color) {
			candidates = append(candidates, card)
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	return candidates[dg.rng.Intn(len(candidates))]
}

// pickWeakCard draws a card from the bottom-tier pool matching the given
// types and color, filtering out restricted and non-viable cards. Returns nil
// if no suitable card is found within maxPickAttempts tries.
func (dg *DeckGenerator) pickWeakCard(cardTypes []CardType, color ColorMask) *Card {
	candidates := dg.filterPool(cardTypes, color)
	if len(candidates) == 0 {
		return nil
	}
	for attempt := 0; attempt < maxPickAttempts; attempt++ {
		card := candidates[dg.rng.Intn(len(candidates))]
		if dg.shouldSkipCard(card) {
			continue
		}
		if !dg.isViableCreature(card) {
			continue
		}
		checkColor := color
		if card.CardType == CardTypeArtifact {
			checkColor = ColorColorless
		}
		if !dg.colorsFriendlyEnough(card, checkColor, false) {
			continue
		}
		return card
	}
	return nil
}

func (dg *DeckGenerator) filterPool(cardTypes []CardType, color ColorMask) []*Card {
	var matching []*Card
	for _, card := range dg.weakPool {
		for _, ct := range cardTypes {
			if card.CardType == ct && dg.matchesColor(card, color) {
				matching = append(matching, card)
				break
			}
		}
	}
	return matching
}

func (dg *DeckGenerator) matchesColor(card *Card, color ColorMask) bool {
	if color == ColorColorless {
		return len(card.ColorIdentity) == 0
	}

	if len(card.ColorIdentity) == 0 {
		return false
	}

	for _, colorStr := range card.ColorIdentity {
		if colorMask, ok := colorStringToMask[colorStr]; ok {
			if color&colorMask != 0 {
				return true
			}
		}
	}

	return false
}

func (dg *DeckGenerator) isBasicLandOfColor(card *Card, color ColorMask) bool {
	if card.CardType != CardTypeLand {
		return false
	}

	if landColor, ok := basicLands[card.CardName]; ok {
		return color&landColor != 0
	}

	return false
}

func (dg *DeckGenerator) colorsFriendlyEnough(card *Card, color ColorMask, lenient bool) bool {
	if len(card.Colors) == 0 {
		return true
	}

	matchCount := 0
	for _, colorStr := range card.Colors {
		if colorMask, ok := colorStringToMask[colorStr]; ok {
			if color&colorMask != 0 {
				matchCount++
			}
		}
	}

	if lenient {
		return matchCount > 0
	}

	return matchCount == len(card.Colors)
}

func (dg *DeckGenerator) shouldSkipCard(card *Card) bool {
	if card.VintageRestricted {
		return true
	}

	if dg.difficulty == DifficultyEasy {
		for _, keyword := range card.Keywords {
			if keyword == "Islandwalk" || keyword == "Swampwalk" {
				return true
			}
		}
	}

	return false
}

func (dg *DeckGenerator) isViableCreature(card *Card) bool {
	if card.CardType != CardTypeCreature {
		return true
	}
	return card.Power >= 0 && card.Toughness >= 0
}

func (dg *DeckGenerator) addCardToDeck(card *Card) {
	dg.deck[card]++
}
