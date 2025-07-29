package domain

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sort"
	"strconv"
	"strings"

	"github.com/klauspost/compress/zstd"
)

type CardJSON struct {
	ScryfallID     string // Used to get images
	OracleID       string
	CardName       string
	ManaCost       string
	ManaProduction []string
	Colors         []string
	ColorIdentity  []string
	Keywords       []string
	TypeLine       string
	Subtypes       []string
	Abilities      []string
	Text           string
	Power          string // can be "*"
	Toughness      string // can be "*"
	SetName        string
	SetID          string
	Rarity         string
	Frame          string
	FlavorText     string
	FrameEffects   []string
	Watermark      string
	Artist         string
}

func LoadCardDatabase(reader io.Reader) []*Card {
	fmt.Println("Loading cards from compressed data")

	// Decompress the data
	decompressedReader, err := decompress(reader)
	if err != nil {
		log.Fatalf("Error decompressing card data: %v", err)
		return nil
	}

	// Decode JSON directly from the reader as an array
	var cardJSONArray []*CardJSON
	decoder := json.NewDecoder(decompressedReader)
	err = decoder.Decode(&cardJSONArray)
	if err != nil {
		log.Fatalf("Error unmarshalling card data: %v", err)
		return nil
	}

	// Convert CardJSON to Card structs
	cards := make([]*Card, 0, len(cardJSONArray))
	for _, cardJSON := range cardJSONArray {
		cards = append(cards, cardJSON.ToCard())
	}

	// Sort by card name
	sort.Slice(cards, func(i, j int) bool {
		return cards[i].CardName < cards[j].CardName
	})

	return cards
}

func decompress(input io.Reader) (io.Reader, error) {
	// Create a zstd decoder
	decoder, err := zstd.NewReader(input)
	if err != nil {
		return nil, fmt.Errorf("failed to create zstd decoder: %w", err)
	}

	return decoder, nil
}

// ToCard converts a CardJSON struct to a Card struct for use in the game
func (cj *CardJSON) ToCard() *Card {
	// Convert string power/toughness to int, handling special cases
	power := convertPowerToughness(cj.Power)
	toughness := convertPowerToughness(cj.Toughness)

	return &Card{
		CardSet: CardSet{
			ID:   cj.SetID,
			Name: cj.SetName,
		},
		ScryfallID:     cj.ScryfallID,
		OracleID:       cj.OracleID,
		CardName:       cj.CardName,
		ManaCost:       cj.ManaCost,
		ManaProduction: cj.ManaProduction,
		Colors:         cj.Colors,
		ColorIdentity:  cj.ColorIdentity,
		Keywords:       cj.Keywords,
		CardType:       parseCardType(cj.TypeLine),
		TypeLine:       cj.TypeLine,
		Subtypes:       cj.Subtypes,
		Abilities:      cj.Abilities,
		Text:           cj.Text,
		Power:          power,
		Toughness:      toughness,
		Rarity:         cj.Rarity,
		Frame:          cj.Frame,
		FlavorText:     cj.FlavorText,
		FrameEffects:   cj.FrameEffects,
		Watermark:      cj.Watermark,
		Artist:         cj.Artist,
	}
}

// parseCardType converts a TypeLine string to a CardType enum
// It looks for the primary card type in the type line
func parseCardType(typeLine string) CardType {
	// Convert to lowercase for case-insensitive matching
	lower := strings.ToLower(typeLine)

	// Check for each card type (order matters for multi-type cards)
	if strings.Contains(lower, "land") {
		return CardTypeLand
	}
	if strings.Contains(lower, "creature") {
		return CardTypeCreature
	}
	if strings.Contains(lower, "artifact") {
		return CardTypeArtifact
	}
	if strings.Contains(lower, "enchantment") {
		return CardTypeEnchantment
	}
	if strings.Contains(lower, "instant") {
		return CardTypeInstant
	}
	if strings.Contains(lower, "sorcery") {
		return CardTypeSorcery
	}

	// Default to empty string if no match found
	return CardType("")
}

// convertPowerToughness converts string power/toughness values to integers
// Returns -1 for variable values like "*" or "X"
func convertPowerToughness(value string) int {
	if intValue, err := strconv.Atoi(value); err == nil {
		return intValue
	}
	return -1
}
