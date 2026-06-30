package screens

import (
	"testing"

	"github.com/benprew/s30/game/domain"
)

func TestCollectionFilterMatches(t *testing.T) {
	blackCreature := &domain.Card{CardName: "Black Knight", Colors: []string{"B"}, CardType: domain.CardTypeCreature}
	redInstant := &domain.Card{CardName: "Lightning Bolt", Colors: []string{"R"}, CardType: domain.CardTypeInstant}
	whiteCreature := &domain.Card{CardName: "Serra Angel", Colors: []string{"W"}, CardType: domain.CardTypeCreature}
	colorlessArtifact := &domain.Card{CardName: "Black Lotus", Colors: nil, CardType: domain.CardTypeArtifact}
	mountain := &domain.Card{CardName: "Mountain", Colors: nil, CardType: domain.CardTypeLand, ManaProduction: []string{"R"}}
	cityOfBrass := &domain.Card{CardName: "City of Brass", Colors: nil, CardType: domain.CardTypeLand, ManaProduction: []string{"W", "U", "B", "R", "G"}}

	tests := []struct {
		name  string
		setup func(f *collectionFilter)
		card  *domain.Card
		want  bool
	}{
		{"no filter matches all", func(f *collectionFilter) {}, redInstant, true},
		{"black matches black creature", func(f *collectionFilter) { f.toggleColor("B") }, blackCreature, true},
		{"black rejects red instant", func(f *collectionFilter) { f.toggleColor("B") }, redInstant, false},
		{"color filter rejects colorless", func(f *collectionFilter) { f.toggleColor("B") }, colorlessArtifact, false},
		{
			"black AND creature matches black creature",
			func(f *collectionFilter) { f.toggleColor("B"); f.toggleType(domain.CardTypeCreature) },
			blackCreature, true,
		},
		{
			"black AND creature rejects white creature",
			func(f *collectionFilter) { f.toggleColor("B"); f.toggleType(domain.CardTypeCreature) },
			whiteCreature, false,
		},
		{
			"creature OR instant matches instant",
			func(f *collectionFilter) { f.toggleType(domain.CardTypeCreature); f.toggleType(domain.CardTypeInstant) },
			redInstant, true,
		},
		{"red matches mountain by mana production", func(f *collectionFilter) { f.toggleColor("R") }, mountain, true},
		{"blue rejects mountain", func(f *collectionFilter) { f.toggleColor("U") }, mountain, false},
		{
			"red AND land matches mountain",
			func(f *collectionFilter) { f.toggleColor("R"); f.toggleType(domain.CardTypeLand) },
			mountain, true,
		},
		{"city of brass matches red", func(f *collectionFilter) { f.toggleColor("R") }, cityOfBrass, true},
		{"city of brass matches white", func(f *collectionFilter) { f.toggleColor("W") }, cityOfBrass, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := newCollectionFilter()
			tc.setup(&f)
			if got := f.matches(tc.card); got != tc.want {
				t.Errorf("matches(%s) = %v, want %v", tc.card.Name(), got, tc.want)
			}
		})
	}
}

func TestCollectionFilterToggleClears(t *testing.T) {
	f := newCollectionFilter()
	if f.active() {
		t.Fatal("new filter should be inactive")
	}
	f.toggleColor("R")
	if !f.active() {
		t.Fatal("filter should be active after toggling a color")
	}
	f.toggleColor("R")
	if f.active() {
		t.Fatal("toggling the same color twice should clear it")
	}
}
