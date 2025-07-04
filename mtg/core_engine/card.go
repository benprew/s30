package core_engine

import (
	"fmt"
)

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
	CardName       string   `json:"name"`
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
	// target         EntityID // target when casting this card. can be an instance
	// of something: another card, or a player. can also be a zone: library or hand
	// but it has to be something in the game
	Events  [][]string // Events that happen when a card is cast, format: [["EventType", param1, param2], ...]
	Actions []Event
}

func (c *Card) Name() string {
	return c.CardName
}

func (c *Card) ReceiveDamage(amount int) {
	c.DamageTaken += amount
}

func (c *Card) TargetType() TargetType {
	return TargetTypeCard
}

func (c *Card) IsDead() bool {
	return c.DamageTaken >= c.Toughness
}

func (c *Card) IsActive() bool {
	return !c.Tapped && c.Active
}

func (c *Card) AddTarget(target Targetable) {
	for _, a := range c.Actions {
		a.AddTarget(target)
	}
}

func (c *Card) UnMarshalActions() {
	for _, e := range c.Events {
		switch e[0] {
		case "DirectDamage":
			// Convert string amount to int
			amount := 0
			fmt.Sscanf(e[1], "%d", &amount)

			// Create DirectDamage with proper amount and target
			// The target should be set based on the card's Target field
			damage := DirectDamage{Amount: amount}
			c.Actions = append(c.Actions, &damage)
		}
	}
}

// DeepCopy creates a deep copy of the Card struct.
func (c *Card) DeepCopy() *Card {
	// Create a new Card instance
	newCard := &Card{
		ID:          c.ID, // ID might need special handling depending on its use (e.g., unique per game instance)
		CardName:    c.CardName,
		ManaCost:    c.ManaCost,
		CardType:    c.CardType,
		Text:        c.Text,
		Power:       c.Power,
		Toughness:   c.Toughness,
		Tapped:      c.Tapped,
		Active:      c.Active,
		DamageTaken: c.DamageTaken,
		Targetable:  c.Targetable,
	}

	// Deep copy slices
	newCard.ManaProduction = make([]string, len(c.ManaProduction))
	copy(newCard.ManaProduction, c.ManaProduction)

	newCard.Colors = make([]string, len(c.Colors))
	copy(newCard.Colors, c.Colors)

	newCard.Subtypes = make([]string, len(c.Subtypes))
	copy(newCard.Subtypes, c.Subtypes)

	newCard.Abilities = make([]string, len(c.Abilities))
	copy(newCard.Abilities, c.Abilities)

	// Deep copy the slice of slices for Events
	newCard.Events = make([][]string, len(c.Events))
	for i, event := range c.Events {
		newCard.Events[i] = make([]string, len(event))
		copy(newCard.Events[i], event)
	}

	// Actions slice contains interfaces, which might need careful deep copying
	// depending on the underlying types. For now, we'll create a new slice
	// but the underlying Event implementations might still be shared references
	// if they are not value types or don't have their own DeepCopy methods.
	// Assuming the current Event implementations (like DirectDamage) are simple
	// value-like structs or don't hold mutable state that needs deep copying
	// for test purposes, a shallow copy of the slice is a starting point.
	// If Events become more complex, this will need refinement.
	newCard.Actions = make([]Event, len(c.Actions))
	copy(newCard.Actions, c.Actions) // This is a shallow copy of the interface values

	return newCard
}
