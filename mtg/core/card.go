package core

import (
	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/mtg/effects"
)

type EntityID int

type Event = effects.Event

const cardNameKirdApe = "Kird Ape"

type Card struct {
	domain.Card
	ID             EntityID
	Owner          *Player
	CurrentZone    Zone
	Tapped         bool
	Active         bool
	DamageTaken    int
	PowerBoost     int
	ToughnessBoost int
}

func (c *Card) Name() string {
	return c.CardName
}

func (c *Card) EntityID() int {
	return int(c.ID)
}

func (c *Card) CardActions() []effects.Event {
	if action := c.specialCaseAction(); action != nil {
		return action
	}

	var events []effects.Event
	for _, ability := range c.ParsedAbilities {
		if event := abilityToEvent(&ability); event != nil {
			events = append(events, event)
		}
	}
	return events
}

func abilityToEvent(ability *domain.ParsedAbility) effects.Event {
	if ability.Effect == nil {
		return nil
	}

	// Activated abilities require a cost and should not resolve when casting
	if ability.Type == "Activated" {
		return nil
	}

	if ability.Effect.Amount > 0 {
		return &effects.DirectDamage{Amount: ability.Effect.Amount}
	}

	if ability.Effect.PowerBoost != 0 || ability.Effect.ToughnessBoost != 0 {
		return &effects.StatBoost{
			PowerBoost:     ability.Effect.PowerBoost,
			ToughnessBoost: ability.Effect.ToughnessBoost,
		}
	}

	return nil
}

func (c *Card) specialCaseAction() []effects.Event {
	switch c.Name() {
	case cardNameKirdApe:
		return []effects.Event{}
	}
	return nil
}

func (c *Card) ReceiveDamage(amount int) {
	c.DamageTaken += amount
}

func (c *Card) TargetType() effects.TargetType {
	return effects.TargetTypeCard
}

func (c *Card) IsDead() bool {
	return c.DamageTaken >= c.EffectiveToughness()
}

func (c *Card) AddPowerBoost(amount int) {
	c.PowerBoost += amount
}

func (c *Card) AddToughnessBoost(amount int) {
	c.ToughnessBoost += amount
}

func (c *Card) EffectivePower() int {
	return c.Power + c.PowerBoost + c.staticPowerBonus()
}

func (c *Card) EffectiveToughness() int {
	return c.Toughness + c.ToughnessBoost + c.staticToughnessBonus()
}

func (c *Card) staticPowerBonus() int {
	if c.Name() == cardNameKirdApe && c.Owner != nil && c.Owner.ControlsLandType("Forest") {
		return 1
	}
	return 0
}

func (c *Card) staticToughnessBonus() int {
	if c.Name() == cardNameKirdApe && c.Owner != nil && c.Owner.ControlsLandType("Forest") {
		return 2
	}
	return 0
}

func (c *Card) ClearEndOfTurnEffects() {
	c.PowerBoost = 0
	c.ToughnessBoost = 0
	c.DamageTaken = 0
}

func (c *Card) GetTargetSpec() *domain.ParsedTargetSpec {
	for _, ability := range c.ParsedAbilities {
		if ability.TargetSpec != nil {
			return ability.TargetSpec
		}
	}
	return nil
}

func (c *Card) TargetsCreaturesOnly() bool {
	spec := c.GetTargetSpec()
	return spec != nil && spec.Type == "creature"
}

func (c *Card) IsActive() bool {
	if c.Tapped {
		return false
	}
	return c.Active || c.HasKeyword(effects.KeywordHaste)
}

func (c *Card) HasKeyword(keyword effects.Keyword) bool {
	kw := string(keyword)
	for _, k := range c.Keywords {
		if k == kw {
			return true
		}
	}
	for _, ability := range c.ParsedAbilities {
		for _, k := range ability.Keywords {
			if k == kw {
				return true
			}
		}
	}
	return false
}

func (c *Card) GetManaProduction() []string {
	for _, ability := range c.ParsedAbilities {
		if ability.Type == "Mana" && ability.Cost != nil && ability.Cost.Tap {
			if ability.Effect != nil && len(ability.Effect.ManaTypes) > 0 {
				return ability.Effect.ManaTypes
			}
		}
	}
	return c.ManaProduction
}

func (c *Card) GetManaAbility() *effects.ManaAbility {
	for _, ability := range c.ParsedAbilities {
		if ability.Type == "Mana" && ability.Cost != nil && ability.Cost.Tap {
			if ability.Effect != nil && len(ability.Effect.ManaTypes) > 0 {
				return &effects.ManaAbility{
					ManaTypes: ability.Effect.ManaTypes,
					AnyColor:  ability.Effect.AnyColor,
				}
			}
		}
	}
	if len(c.ManaProduction) > 0 {
		return &effects.ManaAbility{
			ManaTypes: c.ManaProduction,
			AnyColor:  false,
		}
	}
	return nil
}

func NewCardFromDomain(domainCard *domain.Card, id EntityID, owner *Player) *Card {
	return &Card{
		Card:        *domainCard,
		ID:          id,
		Owner:       owner,
		CurrentZone: ZoneLibrary,
		Tapped:      false,
		Active:      true,
		DamageTaken: 0,
	}
}

func (c *Card) DeepCopy() *Card {
	newCard := &Card{
		ID:             c.ID,
		Card:           c.Card,
		Owner:          c.Owner,
		CurrentZone:    c.CurrentZone,
		Tapped:         c.Tapped,
		Active:         c.Active,
		DamageTaken:    c.DamageTaken,
		PowerBoost:     c.PowerBoost,
		ToughnessBoost: c.ToughnessBoost,
	}

	newCard.ManaProduction = make([]string, len(c.ManaProduction))
	copy(newCard.ManaProduction, c.ManaProduction)

	newCard.Colors = make([]string, len(c.Colors))
	copy(newCard.Colors, c.Colors)

	newCard.Subtypes = make([]string, len(c.Subtypes))
	copy(newCard.Subtypes, c.Subtypes)

	newCard.Abilities = make([]string, len(c.Abilities))
	copy(newCard.Abilities, c.Abilities)

	newCard.ParsedAbilities = make([]domain.ParsedAbility, len(c.ParsedAbilities))
	copy(newCard.ParsedAbilities, c.ParsedAbilities)

	return newCard
}
