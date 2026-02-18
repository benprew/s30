package domain

import (
	"bytes"
	"fmt"
	"image"
	"regexp"
	"sort"
	"strings"

	"github.com/benprew/s30/assets"
	"github.com/hajimehoshi/ebiten/v2"
)

type CardType string

const (
	CardTypeLand        CardType = "Land"
	CardTypeCreature    CardType = "Creature"
	CardTypeArtifact    CardType = "Artifact"
	CardTypeEnchantment CardType = "Enchantment"
	CardTypeInstant     CardType = "Instant"
	CardTypeSorcery     CardType = "Sorcery"
)

type CardView int

const (
	CardViewFull CardView = iota
	CardViewArtOnly
)
const CardArtHeight = 250

type EntityID int

// A struct representing the sets of magic
// ex. Arabian Nights, Zendikar, etc
type CardSet struct {
	SetID       string
	SetName     string
	SetType     string
	CollectorNo string
}

type ParsedCost struct {
	Mana        string `json:"Mana"`
	Tap         bool   `json:"Tap"`
	Sacrifice   bool   `json:"Sacrifice"`
	LifePayment int    `json:"LifePayment"`
}

type ParsedEffect struct {
	Keywords       []string `json:"Keywords"`
	Modifier       string   `json:"Modifier"`
	PowerBoost     int      `json:"PowerBoost"`
	ToughnessBoost int      `json:"ToughnessBoost"`
	Amount         int      `json:"Amount"`
	ManaTypes      []string `json:"ManaTypes"`
	AnyColor       bool     `json:"AnyColor"`
}

type ParsedTrigger struct {
	Type      string `json:"Type"`
	Condition string `json:"Condition"`
}

type ParsedTargetSpec struct {
	Type       string `json:"Type"`
	Controller string `json:"Controller"`
	Subtype    string `json:"Subtype"`
	Count      int    `json:"Count"`
	Condition  string `json:"Condition"`
}

type ParsedAbility struct {
	Type       string            `json:"Type"`
	Cost       *ParsedCost       `json:"Cost"`
	Effect     *ParsedEffect     `json:"Effect"`
	Keywords   []string          `json:"Keywords"`
	Trigger    *ParsedTrigger    `json:"Trigger"`
	TargetSpec *ParsedTargetSpec `json:"TargetSpec"`
	RawText    string            `json:"RawText"`
}

type Card struct {
	CardName string
	CardSet
	ScryfallID        string // Used to get images
	PngURL            string
	OracleID          string
	ManaCost          string   // ex. {3}{G}{R}
	ManaProduction    []string // This only has the possible colors of production
	Colors            []string
	ColorIdentity     []string
	Keywords          []string
	CardType          CardType
	TypeLine          string
	Subtypes          []string
	Abilities         []string
	Text              string
	Power             int // -1 means variable
	Toughness         int // -1 means variable
	Rarity            string
	Frame             string
	FlavorText        string
	FrameEffects      []string
	Watermark         string
	Artist            string
	Price             int
	VintageRestricted bool
	ParsedAbilities   []ParsedAbility
}

// Cards sorted by name
var CARDS = loadCardsWithAbilities()

func loadCardsWithAbilities() []*Card {
	cards := LoadCardDatabase(bytes.NewReader(assets.Cards_json))
	parsedAbilities := LoadParsedAbilities(assets.ParsedCards_json)
	if parsedAbilities != nil {
		ApplyParsedAbilities(cards, parsedAbilities)
	}
	ApplyLandManaAbilities(cards)
	return cards
}

func (c *Card) Name() string {
	return c.CardName
}

// Returns a unique string for each card.
// Used to identify cards when names are the same
func (c *Card) CardID() string {
	id := fmt.Sprintf(
		"%s-%s-%s",
		c.SetID,
		c.CollectorNo,
		sanitizeFilename(c.CardName))
	return id
}

// FindCardByName searches for a card by name using binary search
// Returns the first card found with the given name, or nil if not found
func FindCardByName(name string) *Card {
	index := sort.Search(len(CARDS), func(i int) bool {
		return CARDS[i].CardName >= name
	})

	if index < len(CARDS) && CARDS[index].CardName == name {
		return CARDS[index]
	}

	return nil
}

// FindAllCardsByName searches for all cards with the given name
// Returns a slice of all cards with the matching name (different sets)
func FindAllCardsByName(name string) []*Card {
	var result []*Card

	// Find the first occurrence
	index := sort.Search(len(CARDS), func(i int) bool {
		return CARDS[i].CardName >= name
	})

	// Collect all cards with the same name
	for index < len(CARDS) && CARDS[index].CardName == name {
		result = append(result, CARDS[index])
		index++
	}

	return result
}

func (card *Card) CardImage(view CardView) (*ebiten.Image, error) {
	var fullImg *ebiten.Image

	if cached, ok := cardImages.Load(card.OracleID); ok {
		fullImg = cached.(*ebiten.Image)
	} else {
		if _, alreadyFetching := fetchingSet.LoadOrStore(card.OracleID, true); !alreadyFetching {
			go fetchAndCacheCardImage(card)
		}
		fullImg = blankCard()
	}

	if view == CardViewArtOnly {
		bounds := fullImg.Bounds()
		width := bounds.Dx()
		artRect := image.Rect(0, 0, width, CardArtHeight)
		return fullImg.SubImage(artRect).(*ebiten.Image), nil
	}

	return fullImg, nil
}

func (c *Card) SalePrice(city *City) int {
	basePercentage := 0.5
	switch city.Tier {
	case TierTown:
		basePercentage = 0.6
	case TierCapital:
		basePercentage = 0.75
	}

	hasColorMatch := false
	for _, colorStr := range c.ColorIdentity {
		if colorMask, ok := colorStringToMask[colorStr]; ok {
			if city.AmuletColor&colorMask != 0 {
				hasColorMatch = true
				break
			}
		}
	}

	if hasColorMatch {
		basePercentage += 0.1
	}

	return int(float64(c.Price) * basePercentage)
}

func sanitizeFilename(name string) string {
	name = strings.ToLower(name)

	re1 := regexp.MustCompile(`[^\p{L}\p{N}\s-]`)
	name = re1.ReplaceAllString(name, "")

	re2 := regexp.MustCompile(`[-\s]+`)
	name = re2.ReplaceAllString(name, "-")

	return strings.Trim(name, "-")
}
