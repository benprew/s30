package domain

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
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
	CollectorNo    string
	Rarity         string
	Frame          string
	FlavorText     string
	FrameEffects   []string
	Watermark      string
	Artist         string
	PriceUSD       string
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
	price := normalizePrice(toFloat(cj.PriceUSD))

	return &Card{
		CardSet: CardSet{
			ID:          cj.SetID,
			Name:        cj.SetName,
			CollectorNo: cj.CollectorNo,
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
		Price:          price,
	}
}

func toInt(str string) int {
	if intValue, err := strconv.Atoi(str); err == nil {
		return intValue
	}
	return -1
}

func toFloat(str string) float64 {
	f, _ := strconv.ParseFloat(str, 64)
	return f
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

// normalizePrice converts a price from input range to output range using logarithmic scaling
func normalizePrice(price float64) int {
	const (
		minInput  = 0.10
		maxInput  = 20000.0
		minOutput = 10.0
		maxOutput = 1000.0
	)

	// Clamp input to avoid log(0)
	if price < minInput {
		price = minInput
	}

	// Log-scale
	logMin := math.Log(minInput)
	logMax := math.Log(maxInput)
	logPrice := math.Log(price)

	// Normalize to [0, 1]
	normalized := (logPrice - logMin) / (logMax - logMin)

	// Scale to output range
	scaled := minOutput + normalized*(maxOutput-minOutput)

	return int(scaled)
}
