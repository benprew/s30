package rules

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
		Name:           "Island",
		CardType:       "Land",
		Subtypes:       []string{"Island"},
		Text:           "({T}: Add {U} to your mana pool.)",
		ManaProduction: []string{"U"},
	},
	"Forest": {
		Name:           "Forest",
		CardType:       "Land",
		Subtypes:       []string{"Forest"},
		Text:           "({T}: Add {G} to your mana pool.)",
		ManaProduction: []string{"G"},
	},
	"Tropical Island": {
		Name:           "Tundra",
		CardType:       "Land",
		Subtypes:       []string{"Forest", "Island"},
		Text:           "({T}: Add {G} or {U} to your mana pool.)",
		ManaProduction: []string{"G", "U"},
	},
	"Llanowar Elves": {
		Name:           "Llanowar Elves",
		ManaCost:       "G",
		Colors:         []string{"Green"},
		CardType:       "Creature",
		Subtypes:       []string{"Elf"},
		Power:          1,
		Toughness:      1,
		Abilities:      []string{"{T}: Add {G} to your mana pool."},
		Text:           "{T}: Add {G} to your mana pool.",
		ManaProduction: []string{"G"},
	},
}
