package domain

import (
	"bytes"
	"fmt"
	"image"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/ui/imageutil"
	"github.com/benprew/s30/game/utils"
	"github.com/benprew/s30/game/utils/httpzip"
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

type Card struct {
	CardName string
	CardSet
	ScryfallID        string // Used to get images
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
}

// Cards sorted by name
var CARDS = LoadCardDatabase(bytes.NewReader(assets.Cards_json))
var CARD_IMAGES map[*Card]*ebiten.Image
var remoteCardArchive *httpzip.Reader
var initRemoteCardArchiveOnce sync.Once
var remoteCardArchiveErr error

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
	if CARD_IMAGES == nil {
		CARD_IMAGES = make(map[*Card]*ebiten.Image, 0)
	}

	fullImg := CARD_IMAGES[card]
	if fullImg == nil {
		fmt.Println("Loading uncached card")
		filename := card.Filename()
		var data []byte
		var err error

		if assets.CardImages_zip == nil {
			initRemoteCardArchiveOnce.Do(func() {
				remoteCardArchive, remoteCardArchiveErr = httpzip.NewReader("https://throwingbones.com/ben/s30/cardimages.zip", http.DefaultClient)
			})
			if remoteCardArchiveErr != nil {
				return nil, remoteCardArchiveErr
			}
			data, err = remoteCardArchive.ReadFile(filename)
		} else {
			data, err = utils.ReadFromEmbeddedZip(assets.CardImages_zip, filename)
		}

		if err != nil {
			data = assets.CardBlank_png
			fmt.Printf("WARN: Unable to load card image for: %s, using blank\n", filename)
		}
		fullImg, err = imageutil.LoadImage(data)
		if err != nil {
			return nil, err
		}
		CARD_IMAGES[card] = fullImg
	}

	if view == CardViewArtOnly {
		bounds := fullImg.Bounds()
		width := bounds.Dx()
		artRect := image.Rect(0, 0, width, CardArtHeight)
		return fullImg.SubImage(artRect).(*ebiten.Image), nil
	}

	return fullImg, nil
}

func (c *Card) Filename() string {
	res := "200"
	// for art only images
	// res := "236x"
	fn := fmt.Sprintf("%s-%s-%s-%s.png", c.SetID, c.CollectorNo, res, sanitizeFilename(c.CardName))
	return fn
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
