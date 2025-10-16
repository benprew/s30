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
}

func NewDeckGenerator(difficulty Difficulty, playerColor ColorMask, seed int64) *DeckGenerator {
	return &DeckGenerator{
		deck:        Deck{},
		difficulty:  difficulty,
		playerColor: playerColor,
		rng:         rand.New(rand.NewSource(seed)),
	}
}

func (dg *DeckGenerator) GenerateStartingDeck() Deck {
	dg.deck = Deck{}

	primaryColor := dg.playerColor

	switch dg.difficulty {
	case DifficultyEasy:
		if primaryColor < ColorWhite {
			dg.generateRandomDeck(primaryColor, 13, 7, 15, true, true)
		} else {
			dg.generateRandomDeck(primaryColor, 13, 12, 10, true, true)
		}

	case DifficultyMedium:
		dg.generateRandomDeck(primaryColor, 11, 4, 12, true, true)
		secondaryColor := dg.pickRandomColorOtherThan(primaryColor)
		dg.generateRandomDeck(secondaryColor, 4, 3, 4, false, true)

	case DifficultyHard:
		dg.generateRandomDeck(primaryColor, 9, 3, 9, true, true)
		secondaryColor := dg.pickRandomColorOtherThan(primaryColor)
		dg.generateRandomDeck(secondaryColor, 5, 3, 4, false, true)
		tertiaryColor := dg.pickRandomColorOtherThan(primaryColor | secondaryColor)
		dg.generateRandomDeck(tertiaryColor, 4, 3, 3, false, true)

	case DifficultyExpert:
		dg.generateRandomDeck(primaryColor, 6, 3, 5, true, true)
		dg.generateRandomDeck(ColorColorless, 11, 5, 14, false, true)
	}

	return dg.deck
}

func (dg *DeckGenerator) generateRandomDeck(
	color ColorMask,
	numBasicLands int,
	numEnchantmentsAndArtifacts int,
	numCreatures int,
	addRareCard bool,
	allowArtifacts bool,
) {
	for i := 0; i < numBasicLands; i++ {
		card := dg.pickRandomCard([]CardType{CardTypeLand}, color)
		if card != nil && dg.isBasicLandOfColor(card, color) {
			dg.addCardToDeck(card)
		} else {
			i--
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

		card := dg.pickRandomCard([]CardType{cardType}, cardColor)
		if card == nil {
			i--
			continue
		}

		if dg.shouldSkipCard(card) {
			i--
			continue
		}

		checkColor := cardColor
		if cardType == CardTypeArtifact {
			checkColor = ColorColorless
		}

		rarity := dg.getRarityThreshold(i)
		if dg.colorsFriendlyEnough(card, checkColor, false) &&
			dg.cardRarity(card) <= rarity {
			dg.addCardToDeck(card)
		} else {
			i--
		}
	}

	attempts := 0
	for added := 0; added < numCreatures; added++ {
		card := dg.pickRandomCard([]CardType{CardTypeCreature}, color)
		if card == nil {
			added--
			attempts++
			continue
		}

		if dg.shouldSkipCard(card) {
			added--
			attempts++
			continue
		}

		viable := attempts >= 1000 || dg.isViableCreature(card)
		rarity := dg.getRarityThreshold(added)

		if dg.colorsFriendlyEnough(card, color, false) &&
			dg.cardRarity(card) <= rarity &&
			viable {
			dg.addCardToDeck(card)
		} else {
			added--
		}
		attempts++
	}

	if addRareCard {
		for {
			card := dg.pickRandomCard(
				[]CardType{CardTypeSorcery, CardTypeEnchantment, CardTypeCreature},
				ColorColorless,
			)
			if card == nil {
				continue
			}

			if !dg.colorsFriendlyEnough(card, color, true) {
				continue
			}

			if dg.cardRarity(card) < 3 {
				continue
			}

			if !dg.isViableCreature(card) {
				continue
			}

			if dg.shouldSkipCard(card) {
				continue
			}

			dg.addCardToDeck(card)
			break
		}
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

func (dg *DeckGenerator) pickRandomCard(cardTypes []CardType, color ColorMask) *Card {
	var matchingCards []*Card

	for _, card := range CARDS {
		for _, cardType := range cardTypes {
			if card.CardType == cardType {
				if dg.matchesColor(card, color) {
					matchingCards = append(matchingCards, card)
					break
				}
			}
		}
	}

	if len(matchingCards) == 0 {
		return nil
	}

	return matchingCards[dg.rng.Intn(len(matchingCards))]
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
	if dg.difficulty == DifficultyEasy {
		for _, keyword := range card.Keywords {
			if keyword == "Islandwalk" || keyword == "Swampwalk" {
				return true
			}
		}
	}

	return false
}

func (dg *DeckGenerator) cardRarity(card *Card) int {
	switch card.Rarity {
	case "common":
		return 1
	case "uncommon":
		return 2
	case "rare":
		return 3
	case "mythic":
		return 4
	default:
		return 1
	}
}

func (dg *DeckGenerator) getRarityThreshold(index int) int {
	if index%2 == 0 {
		return 2
	}
	return 1
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
