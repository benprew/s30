package rules

type Card struct {
	Name           string
	ManaCost       string
	ManaProduction []string
	Colors         []string
	CardType       string
	Subtypes       []string
	Abilities      []string
	Text           string
	Power          int
	Toughness      int
	IsTapped       bool
	Active         bool
}

func (c *Card) IsActive() bool {
	return !c.IsTapped && c.Active
}
