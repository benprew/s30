package core_engine

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

type Card struct {
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
}

func (c *Card) IsActive() bool {
	return !c.Tapped && c.Active
}
