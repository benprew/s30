package domain

import (
	"slices"
	"testing"
)

// The bulk-data pipeline emits no Subtypes field, so subtypes have to come out
// of the type line. Without them every dual land looks like a land that gained
// a basic land type, which makes the duel screen draw basic land art for it.
func TestDualLandsCarryTheirBasicLandSubtypes(t *testing.T) {
	duals := map[string][]string{
		"Badlands":        {"Swamp", "Mountain"},
		"Bayou":           {"Swamp", "Forest"},
		"Plateau":         {"Mountain", "Plains"},
		"Savannah":        {"Forest", "Plains"},
		"Scrubland":       {"Plains", "Swamp"},
		"Taiga":           {"Mountain", "Forest"},
		"Tropical Island": {"Forest", "Island"},
		"Tundra":          {"Plains", "Island"},
		"Underground Sea": {"Island", "Swamp"},
		"Volcanic Island": {"Island", "Mountain"},
	}

	for name, want := range duals {
		card := FindCardByName(name)
		if card == nil {
			t.Errorf("%s missing from the card database", name)
			continue
		}
		if !slices.Equal(card.Subtypes, want) {
			t.Errorf("%s subtypes = %v, want %v", name, card.Subtypes, want)
		}
	}
}

func TestParseSubtypes(t *testing.T) {
	tests := []struct {
		typeLine string
		want     []string
	}{
		{"Land — Island Swamp", []string{"Island", "Swamp"}},
		{"Basic Land — Island", []string{"Island"}},
		{"Land — Urza's Mine", []string{"Urza's", "Mine"}},
		{"Legendary Creature — Elder Dragon", []string{"Elder", "Dragon"}},
		{"Enchantment — Aura", []string{"Aura"}},
		{"Land", nil},
		{"Artifact", nil},
		{"", nil},
	}

	for _, tc := range tests {
		if got := parseSubtypes(tc.typeLine); !slices.Equal(got, tc.want) {
			t.Errorf("parseSubtypes(%q) = %v, want %v", tc.typeLine, got, tc.want)
		}
	}
}
