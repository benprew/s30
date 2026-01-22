package core_engine

import (
	"fmt"

	"github.com/benprew/s30/game/domain"
)

type EntityID int

type Card struct {
	domain.Card
	ID          EntityID // In a game, each card will have an entitiyID
	Owner       *Player
	CurrentZone Zone
	Tapped      bool
	Active      bool
	DamageTaken int
	Targetable  string
	// target         EntityID // target when casting this card. can be an instance
	// of something: another card, or a player. can also be a zone: library or hand
	// but it has to be something in the game
	Events  [][]string // Events that happen when a card is cast, format: [["EventType", param1, param2], ...]
	Actions []Event
}

func (c *Card) Name() string {
	return c.CardName
}

func (c *Card) CardActions() []Event {
	if c.Name() == "Lightning Bolt" {
		return []Event{&DirectDamage{Amount: 3}}
	}
	return nil
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

// NewCardFromDomain creates a new core_engine Card from a domain Card
func NewCardFromDomain(domainCard *domain.Card, id EntityID, owner *Player) *Card {
	return &Card{
		Card:        *domainCard,
		ID:          id,
		Owner:       owner,
		CurrentZone: ZoneLibrary,
		Tapped:      false,
		Active:      true,
		DamageTaken: 0,
		Targetable:  "",
		Events:      [][]string{},
		Actions:     []Event{},
	}
}

// DeepCopy creates a deep copy of the Card struct.
func (c *Card) DeepCopy() *Card {
	newCard := &Card{
		ID:          c.ID,
		Card:        c.Card,
		Owner:       c.Owner,
		CurrentZone: c.CurrentZone,
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
