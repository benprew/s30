package mtg

type Card struct {
	Name      string
	ManaCost  string
	Colors    []string
	CardType  string
	Subtypes  []string
	Abilities []string
	Power     int
	Toughness int
	Text      string
}
