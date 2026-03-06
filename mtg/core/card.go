package core

import (
	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/mtg/effects"
)

type EntityID int

type Event = effects.Event

type CounterType struct {
	Power     int
	Toughness int
}

var CounterPlusOnePlusOne = CounterType{1, 1}
var CounterPlusOneZero = CounterType{1, 0}
var CounterMinusOneMinusOne = CounterType{-1, -1}

const cardNameKirdApe = "Kird Ape"

const (
	conditionEnchanted = "enchanted"
	conditionEnchant   = "enchant"
	abilityTypeStatic  = "Static"
)

type Card struct {
	domain.Card
	ID                EntityID
	Owner             *Player
	CurrentZone       Zone
	Tapped            bool
	Active            bool
	DamageTaken       int
	DeathtouchDamaged bool
	Destroyed         bool
	PowerBoost        int
	ToughnessBoost    int
	Attachments       []*Card
	AttachedTo        *Card
	Counters          map[CounterType]int
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

	// Static abilities are continuous effects, not one-shot stack events
	if ability.Type == abilityTypeStatic {
		return nil
	}

	// Aura continuous effects are applied through attachments, not one-shot events
	if ability.TargetSpec != nil && ability.TargetSpec.Condition == conditionEnchanted {
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

	if ability.Effect.Destroy {
		return &effects.DestroyPermanent{Destroy: true}
	}

	return nil
}

func (c *Card) IsAura() bool {
	if c.CardType != domain.CardTypeEnchantment {
		return false
	}
	for _, ability := range c.ParsedAbilities {
		if ability.TargetSpec != nil && ability.TargetSpec.Condition == conditionEnchanted {
			return true
		}
		if ability.TargetSpec != nil && ability.TargetSpec.Condition == conditionEnchant {
			return true
		}
	}
	return false
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
	if c.Destroyed {
		return true
	}
	if c.DeathtouchDamaged && c.DamageTaken > 0 {
		return true
	}
	return c.DamageTaken >= c.EffectiveToughness()
}

func (c *Card) MarkDestroyed() {
	c.Destroyed = true
}

func (c *Card) AddPowerBoost(amount int) {
	c.PowerBoost += amount
}

func (c *Card) AddToughnessBoost(amount int) {
	c.ToughnessBoost += amount
}

func (c *Card) EffectivePower() int {
	return c.Power + c.PowerBoost + c.staticPowerBonus() + c.auraPowerBonus() + c.counterPowerBonus()
}

func (c *Card) EffectiveToughness() int {
	return c.Toughness + c.ToughnessBoost + c.staticToughnessBonus() + c.auraToughnessBonus() + c.counterToughnessBonus()
}

func (c *Card) staticPowerBonus() int {
	bonus := 0
	if c.Name() == cardNameKirdApe && c.Owner != nil && c.Owner.ControlsLandType("Forest") {
		bonus += 1
	}
	bonus += c.lordPowerBonus()
	return bonus
}

func (c *Card) staticToughnessBonus() int {
	bonus := 0
	if c.Name() == cardNameKirdApe && c.Owner != nil && c.Owner.ControlsLandType("Forest") {
		bonus += 2
	}
	bonus += c.lordToughnessBonus()
	return bonus
}

func (c *Card) lordPowerBonus() int {
	if c.Owner == nil || c.CardType != domain.CardTypeCreature {
		return 0
	}
	bonus := 0
	for _, perm := range c.Owner.Battlefield {
		if perm == c {
			continue
		}
		for _, ability := range perm.ParsedAbilities {
			if ability.Type != "Static" || ability.Effect == nil {
				continue
			}
			if !c.matchesLordEffect(perm, &ability) {
				continue
			}
			bonus += ability.Effect.PowerBoost
		}
	}
	return bonus
}

func (c *Card) lordToughnessBonus() int {
	if c.Owner == nil || c.CardType != domain.CardTypeCreature {
		return 0
	}
	bonus := 0
	for _, perm := range c.Owner.Battlefield {
		if perm == c {
			continue
		}
		for _, ability := range perm.ParsedAbilities {
			if ability.Type != "Static" || ability.Effect == nil {
				continue
			}
			if !c.matchesLordEffect(perm, &ability) {
				continue
			}
			bonus += ability.Effect.ToughnessBoost
		}
	}
	return bonus
}

func (c *Card) matchesLordEffect(source *Card, ability *domain.ParsedAbility) bool {
	if ability.TargetSpec != nil && (ability.TargetSpec.Condition == conditionEnchanted || ability.TargetSpec.Condition == conditionEnchant) {
		return false
	}
	if source.AttachedTo != nil {
		return false
	}
	eff := ability.Effect
	if eff.Subtype == "" && eff.GrantedKeyword == "" && eff.PowerBoost == 0 && eff.ToughnessBoost == 0 {
		return false
	}
	if eff.ExcludeSelf && source == c {
		return false
	}
	if eff.Subtype != "" {
		found := false
		for _, st := range c.Subtypes {
			if st == eff.Subtype {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func (c *Card) auraPowerBonus() int {
	bonus := 0
	for _, aura := range c.Attachments {
		for _, ability := range aura.ParsedAbilities {
			if ability.TargetSpec != nil && ability.TargetSpec.Condition == conditionEnchanted && ability.Effect != nil {
				bonus += ability.Effect.PowerBoost
			}
		}
	}
	return bonus
}

func (c *Card) auraToughnessBonus() int {
	bonus := 0
	for _, aura := range c.Attachments {
		for _, ability := range aura.ParsedAbilities {
			if ability.TargetSpec != nil && ability.TargetSpec.Condition == conditionEnchanted && ability.Effect != nil {
				bonus += ability.Effect.ToughnessBoost
			}
		}
	}
	return bonus
}

func (c *Card) ClearEndOfTurnEffects() {
	c.PowerBoost = 0
	c.ToughnessBoost = 0
	c.DamageTaken = 0
	c.DeathtouchDamaged = false
}

func (c *Card) AddCounters(ct CounterType, n int) {
	if c.Counters == nil {
		c.Counters = make(map[CounterType]int)
	}
	c.Counters[ct] += n
}

func (c *Card) RemoveCounters(ct CounterType, n int) {
	if c.Counters == nil {
		return
	}
	c.Counters[ct] -= n
	if c.Counters[ct] <= 0 {
		delete(c.Counters, ct)
	}
}

func (c *Card) CounterCount(ct CounterType) int {
	if c.Counters == nil {
		return 0
	}
	return c.Counters[ct]
}

func (c *Card) counterPowerBonus() int {
	bonus := 0
	for ct, count := range c.Counters {
		bonus += ct.Power * count
	}
	return bonus
}

func (c *Card) counterToughnessBonus() int {
	bonus := 0
	for ct, count := range c.Counters {
		bonus += ct.Toughness * count
	}
	return bonus
}

// remove counters that cancel each other out (ex. +1/+1 and -1/-1)
func (c *Card) CancelCounters() {
	if c.Counters == nil {
		return
	}
	plus := c.Counters[CounterPlusOnePlusOne]
	minus := c.Counters[CounterMinusOneMinusOne]
	if plus > 0 && minus > 0 {
		remove := min(plus, minus)
		c.Counters[CounterPlusOnePlusOne] -= remove
		c.Counters[CounterMinusOneMinusOne] -= remove
		if c.Counters[CounterPlusOnePlusOne] <= 0 {
			delete(c.Counters, CounterPlusOnePlusOne)
		}
		if c.Counters[CounterMinusOneMinusOne] <= 0 {
			delete(c.Counters, CounterMinusOneMinusOne)
		}
	}
}

func (c *Card) GetTargetSpec() *domain.ParsedTargetSpec {
	for _, ability := range c.ParsedAbilities {
		if ability.TargetSpec != nil && ability.TargetSpec.Condition != conditionEnchanted && ability.TargetSpec.Condition != conditionEnchant {
			return ability.TargetSpec
		}
	}
	if c.IsAura() {
		for _, ability := range c.ParsedAbilities {
			if ability.TargetSpec != nil && (ability.TargetSpec.Condition == conditionEnchanted || ability.TargetSpec.Condition == conditionEnchant) {
				return &domain.ParsedTargetSpec{Type: ability.TargetSpec.Type, Count: 1, Condition: ability.TargetSpec.Condition}
			}
		}
		return &domain.ParsedTargetSpec{Type: "creature", Count: 1}
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

func (c *Card) LandwalkType() string {
	lwMap := map[string]string{
		"swampwalk":    "Swamp",
		"forestwalk":   "Forest",
		"mountainwalk": "Mountain",
		"islandwalk":   "Island",
		"plainswalk":   "Plains",
	}
	for _, ability := range c.ParsedAbilities {
		for _, k := range ability.Keywords {
			if k == string(effects.KeywordLandwalk) && ability.Effect != nil {
				if landType, ok := lwMap[ability.Effect.Modifier]; ok {
					return landType
				}
			}
		}
	}
	for _, aura := range c.Attachments {
		for _, ability := range aura.ParsedAbilities {
			if ability.TargetSpec != nil && ability.TargetSpec.Condition == conditionEnchanted {
				for _, k := range ability.Keywords {
					if k == string(effects.KeywordLandwalk) && ability.Effect != nil {
						if landType, ok := lwMap[ability.Effect.Modifier]; ok {
							return landType
						}
					}
				}
			}
		}
	}
	return ""
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
	for _, aura := range c.Attachments {
		for _, ability := range aura.ParsedAbilities {
			if ability.TargetSpec != nil && ability.TargetSpec.Condition == conditionEnchanted {
				for _, k := range ability.Keywords {
					if k == kw {
						return true
					}
				}
			}
		}
	}
	return c.hasLordKeyword(keyword)
}

func (c *Card) hasLordKeyword(keyword effects.Keyword) bool {
	if c.Owner == nil || c.CardType != domain.CardTypeCreature {
		return false
	}
	kw := string(keyword)
	for _, perm := range c.Owner.Battlefield {
		if perm == c {
			continue
		}
		for _, ability := range perm.ParsedAbilities {
			if ability.Type != "Static" || ability.Effect == nil {
				continue
			}
			if !c.matchesLordEffect(perm, &ability) {
				continue
			}
			if ability.Effect.GrantedKeyword == kw {
				return true
			}
		}
	}
	return false
}

func (c *Card) GetActivatedAbilities() []int {
	var indices []int
	for i, ability := range c.ParsedAbilities {
		if ability.Type == "Activated" && ability.Effect != nil && len(ability.Effect.ManaTypes) == 0 {
			indices = append(indices, i)
		}
	}
	return indices
}

func (c *Card) GetManaProduction() []string {
	var result []string
	for _, ability := range c.ParsedAbilities {
		if ability.Type == "Mana" && ability.Cost != nil && ability.Cost.Tap {
			if ability.Effect != nil && len(ability.Effect.ManaTypes) > 0 {
				result = append(result, ability.Effect.ManaTypes...)
			}
		}
	}
	if len(result) == 0 {
		result = c.ManaProduction
	}
	for _, aura := range c.Attachments {
		for _, ability := range aura.ParsedAbilities {
			if ability.Type == "Mana" && ability.TargetSpec != nil && ability.TargetSpec.Condition == conditionEnchanted {
				if ability.Effect != nil && len(ability.Effect.ManaTypes) > 0 {
					result = append(result, ability.Effect.ManaTypes...)
				}
			}
		}
	}
	return result
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
		ID:                c.ID,
		Card:              c.Card,
		Owner:             c.Owner,
		CurrentZone:       c.CurrentZone,
		Tapped:            c.Tapped,
		Active:            c.Active,
		DamageTaken:       c.DamageTaken,
		DeathtouchDamaged: c.DeathtouchDamaged,
		Destroyed:         c.Destroyed,
		PowerBoost:        c.PowerBoost,
		ToughnessBoost:    c.ToughnessBoost,
		Attachments:       nil,
		AttachedTo:        nil,
	}

	if c.Counters != nil {
		newCard.Counters = make(map[CounterType]int, len(c.Counters))
		for k, v := range c.Counters {
			newCard.Counters[k] = v
		}
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
