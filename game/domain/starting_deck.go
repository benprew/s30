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
	ColorAny       ColorMask = ColorWhite | ColorBlue | ColorBlack | ColorRed | ColorGreen
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

func DifficultyToString(d Difficulty) string {
	switch d {
	case DifficultyEasy:
		return "Apprentice"
	case DifficultyMedium:
		return "Magician"
	case DifficultyHard:
		return "Sorcerer"
	case DifficultyExpert:
		return "Wizard"
	default:
		return "Unknown"
	}
}

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
			TierPlayedQuiteOften,
			TierPlayedFromTimeToTime,
			TierPlayedInSpecificArchetypes,
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
		dg.generateRandomDeck(primaryColor, 13, 12, 10, true)
		dg.addGuaranteedRare(primaryColor)

	case DifficultyMedium:
		dg.generateRandomDeck(primaryColor, 11, 4, 12, true)
		dg.addGuaranteedRare(primaryColor)
		secondaryColor := dg.pickRandomColorOtherThan(primaryColor)
		dg.generateRandomDeck(secondaryColor, 4, 3, 4, true)

	case DifficultyHard:
		dg.generateRandomDeck(primaryColor, 9, 3, 9, true)
		dg.addGuaranteedRare(primaryColor)
		secondaryColor := dg.pickRandomColorOtherThan(primaryColor)
		dg.generateRandomDeck(secondaryColor, 5, 3, 4, true)
		tertiaryColor := dg.pickRandomColorOtherThan(primaryColor | secondaryColor)
		dg.generateRandomDeck(tertiaryColor, 4, 3, 3, true)

	case DifficultyExpert:
		dg.generateRandomDeck(primaryColor, 6, 3, 5, true)
		dg.addGuaranteedRare(primaryColor)
		dg.generateRandomDeck(ColorAny, 11, 5, 14, true)
	}

	return dg.deck
}

// CreateStartingResources generates the active starting deck (sized for the difficulty)
// and returns extra cards (bringing the total starting card pool to 50 cards)
// intended for the player's collection.
func (dg *DeckGenerator) CreateStartingResources() (Deck, []*Card) {
	deck := dg.CreateStartingDeck()

	totalDeckCards := 0
	for _, count := range deck {
		totalDeckCards += count
	}

	neededExtra := 50 - totalDeckCards
	if neededExtra <= 0 {
		return deck, nil
	}

	var extraCards []*Card
	seenInExtra := make(map[*Card]int)

	// Add 2-3 extra basic lands of player's primary color
	numExtraLands := 2
	if neededExtra > 10 {
		numExtraLands = 3
	}
	for range numExtraLands {
		if land := dg.pickBasicLand(dg.playerColor); land != nil {
			extraCards = append(extraCards, land)
			neededExtra--
		}
	}

	// Generate remaining extra non-land cards (creatures and spells)
	spellTypes := []CardType{CardTypeInstant, CardTypeSorcery, CardTypeEnchantment, CardTypeArtifact}
	creatureTypes := []CardType{CardTypeCreature}

	for i := 0; i < neededExtra; i++ {
		var candidateTypes []CardType
		var targetColor ColorMask

		if i%2 == 0 {
			candidateTypes = creatureTypes
			targetColor = dg.playerColor
		} else {
			if dg.rng.Intn(4) == 0 {
				candidateTypes = []CardType{CardTypeArtifact}
				targetColor = ColorColorless
			} else {
				candidateTypes = spellTypes
				targetColor = dg.playerColor
			}
		}

		card := dg.pickCardForCollection(candidateTypes, targetColor, seenInExtra)
		if card != nil {
			extraCards = append(extraCards, card)
			seenInExtra[card]++
		} else {
			if land := dg.pickBasicLand(dg.playerColor); land != nil {
				extraCards = append(extraCards, land)
			}
		}
	}

	return deck, extraCards
}

func (dg *DeckGenerator) pickCardForCollection(cardTypes []CardType, color ColorMask, seenInExtra map[*Card]int) *Card {
	candidates := dg.filterPool(cardTypes, color)
	if len(candidates) == 0 {
		return nil
	}
	perm := dg.rng.Perm(len(candidates))
	for _, idx := range perm {
		card := candidates[idx]
		if dg.deck[card] > 0 || seenInExtra[card] > 0 {
			continue
		}
		if dg.shouldSkipCard(card) || !dg.isViableCreature(card) {
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

func (dg *DeckGenerator) addGuaranteedRare(color ColorMask) {
	rare := dg.pickGuaranteedRare(color)
	if rare != nil {
		dg.addCardToDeck(rare)
	}
}

func (dg *DeckGenerator) pickGuaranteedRare(color ColorMask) *Card {
	var candidates []*Card
	seen := make(map[*Card]bool)

	// Sample from upper-mid tiers (TierStaple, TierPlayedInMostDecks, TierPlayedQuiteOften)
	highPool := CardsInTiers(TierStaple, TierPlayedInMostDecks, TierPlayedQuiteOften)
	for _, card := range highPool {
		if card.CardType == CardTypeLand || seen[card] {
			continue
		}
		if dg.shouldSkipCard(card) {
			continue
		}
		if dg.deck[card] > 0 {
			continue
		}
		if dg.matchesColor(card, color) && dg.colorsFriendlyEnough(card, color, true) {
			candidates = append(candidates, card)
			seen[card] = true
		}
	}

	// Also check CARDS for any rare matching the color (excluding restricted/ante/mandatory)
	for _, card := range CARDS {
		if card.CardType == CardTypeLand || seen[card] {
			continue
		}
		if dg.shouldSkipCard(card) {
			continue
		}
		if dg.deck[card] > 0 {
			continue
		}
		if (card.Rarity == "rare" || card.Rarity == "uncommon") && dg.matchesColor(card, color) && dg.colorsFriendlyEnough(card, color, false) {
			candidates = append(candidates, card)
			seen[card] = true
		}
	}

	if len(candidates) == 0 {
		return nil
	}

	return candidates[dg.rng.Intn(len(candidates))]
}

// Starting deck shape per difficulty matches the Shandalar specification:
//   Apprentice (Easy):   1 color,  36 cards (13 lands, 12 spells/artifacts, 10 creatures, 1 rare)
//   Magician   (Medium): 2 colors, 39 cards (15 lands,  7 spells/artifacts, 16 creatures, 1 rare)
//   Sorcerer   (Hard):   3 colors, 44 cards (18 lands,  9 spells/artifacts, 16 creatures, 1 rare)
//   Wizard     (Expert): 5 colors, 45 cards (17 lands,  8 spells/artifacts, 19 creatures, 1 rare)

func (dg *DeckGenerator) generateRandomDeck(
	color ColorMask,
	numBasicLands int,
	numSpellsAndArtifacts int,
	numCreatures int,
	allowArtifacts bool,
) {
	for range numBasicLands {
		card := dg.pickBasicLand(color)
		if card != nil {
			dg.addCardToDeck(card)
		} else {
			break
		}
	}

	spellTypes := []CardType{CardTypeInstant, CardTypeSorcery, CardTypeEnchantment}

	for range numSpellsAndArtifacts {
		var card *Card
		if allowArtifacts && dg.rng.Intn(3) == 0 {
			card = dg.pickWeakCard([]CardType{CardTypeArtifact}, ColorColorless)
		} else {
			card = dg.pickWeakCard(spellTypes, color)
		}

		if card == nil {
			card = dg.pickWeakCard(append(spellTypes, CardTypeArtifact), color)
		}

		if card != nil {
			dg.addCardToDeck(card)
		}
	}

	for range numCreatures {
		card := dg.pickWeakCard([]CardType{CardTypeCreature}, color)
		if card != nil {
			dg.addCardToDeck(card)
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

// pickWeakCard draws a card from the starting pool matching the given
// types and color, filtering out restricted, non-viable, and duplicate cards.
func (dg *DeckGenerator) pickWeakCard(cardTypes []CardType, color ColorMask) *Card {
	candidates := dg.filterPool(cardTypes, color)
	if len(candidates) == 0 {
		return nil
	}
	perm := dg.rng.Perm(len(candidates))
	for _, idx := range perm {
		card := candidates[idx]
		if dg.deck[card] >= 1 {
			continue
		}
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

	// Never allow top-tier mandatory or almost mandatory cards in starting decks
	if tier, ok := CardTierForName(card.CardName); ok {
		if tier == TierMandatory || tier == TierAlmostMandatory {
			return true
		}
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
