package screens

import (
	"encoding/json"
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
	// A colorless artifact that taps for a color. The color filter is about what a
	// card can produce, not about the card being a land.
	talisman := &domain.Card{CardName: "Talisman of Unity", Colors: nil, CardType: domain.CardTypeArtifact, ManaProduction: []string{"G"}}

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
		{"green matches an artifact that produces green", func(f *collectionFilter) { f.toggleColor("G") }, talisman, true},
		{"blue rejects an artifact that produces green", func(f *collectionFilter) { f.toggleColor("U") }, talisman, false},
		{
			"green AND artifact matches an artifact that produces green",
			func(f *collectionFilter) { f.toggleColor("G"); f.toggleType(domain.CardTypeArtifact) },
			talisman, true,
		},
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

// The chain the filter depends on, end to end: the card data carries the mana a
// card can produce, and the filter reads it from the same place. A Mox Emerald
// has to reach the green filter through the loader, not through a field only
// these tests fill in.
func TestColorFilterSeesManaFromTheCardData(t *testing.T) {
	const moxEmerald = `{
		"CardName": "Mox Emerald",
		"TypeLine": "Artifact",
		"ManaProduction": ["G"],
		"PriceUSD": "1.00"
	}`

	var cj domain.CardJSON
	if err := json.Unmarshal([]byte(moxEmerald), &cj); err != nil {
		t.Fatalf("unmarshal card data: %v", err)
	}
	mox := cj.ToCard()

	f := newCollectionFilter()
	f.toggleColor("G")
	if !f.matches(mox) {
		t.Error("the green filter does not see a Mox Emerald, whose card data says it produces green")
	}

	f = newCollectionFilter()
	f.toggleColor("U")
	if f.matches(mox) {
		t.Error("the blue filter matched a Mox Emerald, which produces green")
	}
}
