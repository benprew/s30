package domain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
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
	CardViewFullMini
	CardViewArtMini
)
const cardSourceArtHeight = 215
const CardArtWidth = 147
const CardArtHeight = 129
const CardFullMiniWidth = 183
const CardFullMiniHeight = 256
const CardFullWidth = 245

const CardArtMiniWidth = 110
const CardArtMiniHeight = 96

type EntityID int

type ParsedCost struct {
	Mana        string `json:"Mana"`
	Tap         bool   `json:"Tap"`
	Sacrifice   bool   `json:"Sacrifice"`
	LifePayment int    `json:"LifePayment"`
}

type ParsedEffect struct {
	Keywords        []string `json:"Keywords"`
	Modifier        string   `json:"Modifier"`
	PowerBoost      int      `json:"PowerBoost"`
	ToughnessBoost  int      `json:"ToughnessBoost"`
	Amount          int      `json:"Amount"`
	ManaTypes       []string `json:"ManaTypes"`
	AnyColor        bool     `json:"AnyColor"`
	Destroy         bool     `json:"Destroy,omitempty"`
	Subtype         string   `json:"Subtype,omitempty"`
	GrantedKeyword  string   `json:"GrantedKeyword,omitempty"`
	GrantedModifier string   `json:"GrantedModifier,omitempty"`
	ExcludeSelf     bool     `json:"ExcludeSelf,omitempty"`
	ETBCounters     int      `json:"ETBCounters"`
	CounterPower    int      `json:"CounterPower"`
	CounterTough    int      `json:"CounterTough"`
	TapTarget       bool     `json:"TapTarget,omitempty"`
	DoesNotUntap    bool     `json:"DoesNotUntap,omitempty"`
	UntapCost       string   `json:"UntapCost,omitempty"`
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
	CardName          string
	ManaCost          string   // ex. {3}{G}{R}
	ManaProduction    []string // This only has the possible colors of production
	Colors            []string
	ColorIdentity     []string
	Keywords          []string
	CardType          CardType
	TypeLine          string
	Subtypes          []string
	Text              string
	Power             int // -1 means variable
	Toughness         int // -1 means variable
	Rarity            string
	Frame             string
	FlavorText        string
	FrameEffects      []string
	Watermark         string
	Artworks          []Artwork
	Price             int
	VintageRestricted bool
	ParsedAbilities   []ParsedAbility
}

// Cards sorted by name
var CARDS = loadCardsWithAbilities()

func loadCardsWithAbilities() []*Card {
	return LoadCardDatabase(bytes.NewReader(assets.Cards_json))
}

func (c *Card) Name() string {
	return c.CardName
}

// CardID returns the card name, independent of its artwork.
func (c *Card) CardID() string {
	return c.CardName
}

// ColorMask returns the card's colors as a ColorMask (colorless = 0).
func (c *Card) ColorMask() ColorMask {
	var m ColorMask
	for _, s := range c.Colors {
		m |= colorStringToMask[s]
	}
	return m
}

var manaCostToken = regexp.MustCompile(`\{([^}]+)\}`)

// ManaValue returns the card's converted mana cost parsed from ManaCost
// (e.g. "{3}{G}{R}" -> 5). Generic numbers contribute their value, every other
// pip (colored, hybrid, phyrexian) contributes 1, and {X} contributes 0.
func (c *Card) ManaValue() int {
	total := 0
	for _, m := range manaCostToken.FindAllStringSubmatch(c.ManaCost, -1) {
		tok := strings.ToUpper(m[1])
		if tok == "X" || tok == "Y" || tok == "Z" {
			continue
		}
		if n, err := strconv.Atoi(tok); err == nil {
			total += n
			continue
		}
		total++
	}
	return total
}

// IsLand reports whether the card is a land.
func (c *Card) IsLand() bool {
	return c.CardType == CardTypeLand
}

// MarshalJSON serializes a Card as just its CardID. The full card data lives
// in the embedded CARDS database; saves only need the identifier.
func (c *Card) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		CardID string `json:"CardID"`
	}{CardID: c.CardID()})
}

// UnmarshalJSON resolves name references and printing IDs from older saves.
func (c *Card) UnmarshalJSON(data []byte) error {
	var aux struct {
		CardID      string `json:"CardID"`
		CardName    string `json:"CardName,omitempty"`
		SetID       string `json:"SetID,omitempty"`
		CollectorNo string `json:"CollectorNo,omitempty"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	id := aux.CardID
	if id == "" && aux.CardName != "" {
		id = aux.CardName
	}

	found := FindCard(id, aux.CardName)
	if found == nil {
		return nil
	}
	*c = *found
	return nil
}

func (c *Card) ImageLoaded() bool {
	return c.DefaultArtwork().ImageLoaded()
}

// FindCard attempts to find a card by its ID, falling back to looking up by name
// or extracting the sanitized card name from the ID if the specific printing/set
// is not found in CARDS.
func FindCard(id string, name string) *Card {
	if card := FindCardByID(id); card != nil {
		return card
	}
	if card := FindCardByName(name); card != nil {
		return card
	}
	if parts := strings.SplitN(id, "-", 3); len(parts) == 3 {
		return FindCardBySanitizedName(parts[2])
	}
	fmt.Println("couldn't find card from id:", id, "name:", name)
	return nil
}

// FindCardByName searches for a card by name using binary search.
// Returns the first card found with the given name, or nil if not found.
func FindCardByName(name string) *Card {
	index := sort.Search(len(CARDS), func(i int) bool {
		return CARDS[i].CardName >= name
	})

	if index < len(CARDS) && CARDS[index].CardName == name {
		return CARDS[index]
	}

	return nil
}

// FindCardBySanitizedName searches for a card matching the sanitized name slug.
func FindCardBySanitizedName(nameish string) *Card {
	target := sanitizeFilename(nameish)
	if target == "" {
		return nil
	}
	for _, c := range CARDS {
		if sanitizeFilename(c.CardName) == target {
			return c
		}
	}
	return nil
}

// FindCardByID finds a definition by its card name.
func FindCardByID(id string) *Card {
	return FindCardByName(id)
}

func (card *Card) fullCardImage() *ebiten.Image {
	return card.DefaultArtwork().fullCardImage()
}

func (art Artwork) fullCardImage() *ebiten.Image {
	if cached, ok := cardImages.Load(art.ID); ok {
		return cached.(*ebiten.Image)
	}
	if _, alreadyFetching := fetchingSet.LoadOrStore(art.ID, true); !alreadyFetching {
		go fetchAndCacheArtwork(art)
	}
	return labeledBlankCard(art.CardName)
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
