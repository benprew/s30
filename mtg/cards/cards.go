package mtg

var CardDatabase = map[string]*Card{
	"Lightning Bolt": {
		Name:      "Lightning Bolt",
		ManaCost:  "R",
		Colors:    []string{"Red"},
		CardType:  "Instant",
		Abilities: []string{"Deal 3 damage to any target."},
		Text:      "Lightning Bolt deals 3 damage to any target.",
	},
	"Giant Growth": {
		Name:      "Giant Growth",
		ManaCost:  "G",
		Colors:    []string{"Green"},
		CardType:  "Instant",
		Abilities: []string{"Target creature gets +3/+3 until end of turn."},
		Text:      "Target creature gets +3/+3 until end of turn.",
	},
	"Island": {
		Name:     "Island",
		CardType: "Land",
		Subtypes: []string{"Island"},
		Text:     "({T}: Add {U} to your mana pool.)",
	},
}
