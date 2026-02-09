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
	ScryfallID        string // Used to get images
	OracleID          string
	CardName          string
	ManaCost          string
	ManaProduction    []string
	Colors            []string
	ColorIdentity     []string
	Keywords          []string
	TypeLine          string
	Subtypes          []string
	Abilities         []string
	Text              string
	Power             string // can be "*"
	Toughness         string // can be "*"
	SetName           string
	SetID             string
	CollectorNo       string
	Rarity            string
	Frame             string
	FlavorText        string
	FrameEffects      []string
	Watermark         string
	Artist            string
	PriceUSD          string
	VintageRestricted bool
}

type ParsedCardJSON struct {
	CardName  string          `json:"card_name"`
	Text      string          `json:"text"`
	Abilities []ParsedAbility `json:"abilities"`
	Unparsed  []string        `json:"unparsed"`
}

type ParsedCardsFile struct {
	Parsed []ParsedCardJSON `json:"parsed"`
}

func LoadCardDatabase(reader io.Reader) []*Card {
	decompressedReader, err := decompress(reader)
	if err != nil {
		log.Fatalf("Error decompressing card data: %v", err)
		return nil
	}

	var cardJSONArray []*CardJSON
	decoder := json.NewDecoder(decompressedReader)
	err = decoder.Decode(&cardJSONArray)
	if err != nil {
		log.Fatalf("Error unmarshalling card data: %v", err)
		return nil
	}

	cards := make([]*Card, 0, len(cardJSONArray))
	for _, cardJSON := range cardJSONArray {
		cards = append(cards, cardJSON.ToCard())
	}

	sort.Slice(cards, func(i, j int) bool {
		return cards[i].CardName < cards[j].CardName
	})

	fmt.Printf("Loaded %d cards\n", len(cards))

	return cards
}

func LoadParsedAbilities(data []byte) map[string][]ParsedAbility {
	var parsedCards []ParsedCardJSON
	if err := json.Unmarshal(data, &parsedCards); err != nil {
		log.Printf("Error unmarshalling parsed cards: %v", err)
		return nil
	}

	result := make(map[string][]ParsedAbility, len(parsedCards))
	for _, pc := range parsedCards {
		if len(pc.Abilities) > 0 {
			result[pc.CardName] = pc.Abilities
		}
	}

	fmt.Printf("Loaded %d parsed card abilities\n", len(result))
	return result
}

func ApplyParsedAbilities(cards []*Card, parsedAbilities map[string][]ParsedAbility) {
	matched := 0
	for _, card := range cards {
		if abilities, ok := parsedAbilities[card.CardName]; ok {
			card.ParsedAbilities = abilities
			matched++
		}
	}
	fmt.Printf("Applied parsed abilities to %d cards\n", matched)
}

var subtypeToMana = map[string]string{
	"Plains":   "W",
	"Island":   "U",
	"Swamp":    "B",
	"Mountain": "R",
	"Forest":   "G",
}

func ApplyLandManaAbilities(cards []*Card) {
	added := 0
	for _, card := range cards {
		if card.CardType != CardTypeLand {
			continue
		}

		hasManaAbility := false
		for _, a := range card.ParsedAbilities {
			if a.Type == "Mana" {
				hasManaAbility = true
				break
			}
		}
		if hasManaAbility {
			continue
		}

		var manaTypes []string
		for _, subtype := range card.Subtypes {
			if mana, ok := subtypeToMana[subtype]; ok {
				manaTypes = append(manaTypes, mana)
			}
		}

		if len(manaTypes) > 0 {
			card.ParsedAbilities = append(card.ParsedAbilities, ParsedAbility{
				Type: "Mana",
				Cost: &ParsedCost{Tap: true},
				Effect: &ParsedEffect{
					ManaTypes: manaTypes,
				},
			})
			added++
		}
	}
	fmt.Printf("Added land mana abilities to %d cards\n", added)
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
			SetID:       cj.SetID,
			SetName:     cj.SetName,
			CollectorNo: cj.CollectorNo,
		},
		ScryfallID:        cj.ScryfallID,
		OracleID:          cj.OracleID,
		CardName:          cj.CardName,
		ManaCost:          cj.ManaCost,
		ManaProduction:    cj.ManaProduction,
		Colors:            cj.Colors,
		ColorIdentity:     cj.ColorIdentity,
		Keywords:          cj.Keywords,
		CardType:          parseCardType(cj.TypeLine),
		TypeLine:          cj.TypeLine,
		Subtypes:          cj.Subtypes,
		Abilities:         cj.Abilities,
		Text:              cj.Text,
		Power:             power,
		Toughness:         toughness,
		Rarity:            cj.Rarity,
		Frame:             cj.Frame,
		FlavorText:        cj.FlavorText,
		FrameEffects:      cj.FrameEffects,
		Watermark:         cj.Watermark,
		Artist:            cj.Artist,
		Price:             price,
		VintageRestricted: cj.VintageRestricted,
	}
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
		minInput  = 0.25
		maxInput  = 20000.0
		minOutput = 10.0
		maxOutput = 1500.0
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
