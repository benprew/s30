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
	PngURL            string
	BorderCropURL     string
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
	price := CalculateCardPrice(cj.CardName, cj.TypeLine, toFloat(cj.PriceUSD))

	return &Card{
		CardSet: CardSet{
			SetID:       cj.SetID,
			SetName:     cj.SetName,
			CollectorNo: cj.CollectorNo,
		},
		PngURL:        cj.PngURL,
		BorderCropURL: cj.BorderCropURL,
		cardID: fmt.Sprintf("%s-%s-%s",
			cj.SetID, cj.CollectorNo, sanitizeFilename(cj.CardName)),
		CardName:          cj.CardName,
		ManaCost:          cj.ManaCost,
		ManaProduction:    cj.ManaProduction,
		Colors:            cj.Colors,
		ColorIdentity:     cj.ColorIdentity,
		Keywords:          cj.Keywords,
		CardType:          parseCardType(cj.TypeLine),
		TypeLine:          cj.TypeLine,
		Subtypes:          cj.Subtypes,
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

// CalculateCardPrice computes the gold purchase price of a card based on its
// power tier, card type, and real-world market value (priceUSD).
func CalculateCardPrice(cardName string, typeLine string, priceUSD float64) int {
	if isBasicLandNameOrType(cardName, typeLine) {
		return 30
	}

	tier, ok := CardTierForName(cardName)
	if !ok {
		tier = TierRarelyPlayed
	}

	basePrice := BasePriceForTier(tier)

	const (
		minUSD = 0.25
		maxUSD = 10000.0
	)
	clampedUSD := priceUSD
	if clampedUSD < minUSD {
		clampedUSD = minUSD
	} else if clampedUSD > maxUSD {
		clampedUSD = maxUSD
	}

	logMin := math.Log(minUSD)
	logMax := math.Log(maxUSD)
	logVal := math.Log(clampedUSD)
	norm := (logVal - logMin) / (logMax - logMin)

	multiplier := 1.0 + 0.5*(norm-0.5)
	price := int(math.Round(float64(basePrice) * multiplier))

	switch tier {
	case TierMandatory:
		if price > 7000 {
			price = 7000
		}
	case TierRarelyPlayed:
		if price > 30 {
			price = 30
		}
	}

	if price < 7 {
		price = 7
	}

	return price
}

func isBasicLandNameOrType(name string, typeLine string) bool {
	if _, ok := basicLands[name]; ok {
		return true
	}
	lower := strings.ToLower(typeLine)
	return strings.Contains(lower, "basic") && strings.Contains(lower, "land")
}
