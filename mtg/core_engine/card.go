package core_engine

import "github.com/benprew/s30/mtg/core_engine/events"

// CardType represents the type of a Magic: The Gathering card.
type CardType string

const (
	CardTypeLand        CardType = "Land"        // Produces mana.
	CardTypeCreature    CardType = "Creature"    // Attacks and blocks.
	CardTypeArtifact    CardType = "Artifact"    // Represents magical items or constructs.
	CardTypeEnchantment CardType = "Enchantment" // Ongoing magical effects.
	CardTypeInstant     CardType = "Instant"     // Cast at almost any time.
	CardTypeSorcery     CardType = "Sorcery"     // Cast on your turn, main phase, stack empty.
)

type EntityID int

type Card struct {
	ID             EntityID // In a game, each card will have an entitiyID
	Name           string
	ManaCost       string
	ManaProduction []string
	Colors         []string
	CardType       CardType // Use the new CardType enum
	Subtypes       []string
	Abilities      []string
	Text           string
	Power          int
	Toughness      int
	Tapped         bool
	Active         bool
	DamageTaken    int
	Targetable     string
	Target         EntityID // target when casting this card. can be an instance
	// of something: another card, or a player. can also be a zone: library or hand
	// but it has to be something in the game
	Events  [][]string // Events that happen when a card is cast, format: [["EventType", param1, param2], ...]
	Actions []Event
}

func (c *Card) IsActive() bool {
	return !c.Tapped && c.Active
}

func (c *Card) ReceiveDamage(amount int) {
	c.DamageTaken += amount
}

func (c *Card) CardName() string                   { return c.Name }
func (c *Card) AddTarget(target events.Targetable) {}
