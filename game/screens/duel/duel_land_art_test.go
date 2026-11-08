package duel

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/benprew/s30/game/domain"
)

func TestPermanentArtName_ChangedBasicLandSubtype(t *testing.T) {
	cityOfBrass := &domain.Card{
		CardName: "City of Brass",
		CardType: domain.CardTypeLand,
	}
	perm := interactive.PermanentState{
		Name:     "City of Brass",
		IsLand:   true,
		SubTypes: "Forest",
	}

	if got := permanentArtName(perm, cityOfBrass); got != "Forest" {
		t.Fatalf("permanentArtName() = %q, want %q", got, "Forest")
	}
}

func TestPermanentArtName_UnchangedBasicLandSubtype(t *testing.T) {
	forest := &domain.Card{
		CardName: "Forest",
		CardType: domain.CardTypeLand,
		Subtypes: []string{"Forest"},
	}
	perm := interactive.PermanentState{
		Name:     "Forest",
		IsLand:   true,
		SubTypes: "Forest",
	}

	if got := permanentArtName(perm, forest); got != "Forest" {
		t.Fatalf("permanentArtName() = %q, want %q", got, "Forest")
	}
}

func TestPermanentArtName_AddedBasicLandSubtype(t *testing.T) {
	forest := &domain.Card{
		CardName: "Forest",
		CardType: domain.CardTypeLand,
		Subtypes: []string{"Forest"},
	}
	perm := interactive.PermanentState{
		Name:     "Forest",
		IsLand:   true,
		SubTypes: "Forest Mountain",
	}

	if got := permanentArtName(perm, forest); got != "Mountain" {
		t.Fatalf("permanentArtName() = %q, want %q", got, "Mountain")
	}
}

func TestPermanentArtName_RevertsWhenSubtypeChangeEnds(t *testing.T) {
	cityOfBrass := &domain.Card{
		CardName: "City of Brass",
		CardType: domain.CardTypeLand,
	}
	perm := interactive.PermanentState{
		Name:   "City of Brass",
		IsLand: true,
	}

	if got := permanentArtName(perm, cityOfBrass); got != "City of Brass" {
		t.Fatalf("permanentArtName() = %q, want %q", got, "City of Brass")
	}
}

func TestPermanentArtName_NonlandWithSubtypeWord(t *testing.T) {
	card := &domain.Card{CardName: "Forest Bear", CardType: domain.CardTypeCreature}
	perm := interactive.PermanentState{
		Name:     "Forest Bear",
		SubTypes: "Bear Forest",
	}

	if got := permanentArtName(perm, card); got != "Forest Bear" {
		t.Fatalf("permanentArtName() = %q, want %q", got, "Forest Bear")
	}
}

func TestBuildCardImageMap_IncludesBasicLands(t *testing.T) {
	cardImages := buildCardImageMap(nil)
	for _, name := range basicLandNames {
		if cardImages[name] == nil {
			t.Errorf("buildCardImageMap() missing %q", name)
		}
	}
}

// A dual land has printed basic land types, so nothing about it has changed and
// it must keep its own art instead of degrading to one of the basics.
func TestPermanentArtName_DualLandKeepsItsOwnArt(t *testing.T) {
	duals := map[string]string{
		"Badlands":        "Swamp Mountain",
		"Bayou":           "Swamp Forest",
		"Plateau":         "Mountain Plains",
		"Savannah":        "Forest Plains",
		"Scrubland":       "Plains Swamp",
		"Taiga":           "Mountain Forest",
		"Tropical Island": "Forest Island",
		"Tundra":          "Plains Island",
		"Underground Sea": "Island Swamp",
		"Volcanic Island": "Island Mountain",
	}

	for name, subtypes := range duals {
		printed := domain.FindCardByName(name)
		if printed == nil {
			t.Errorf("%s missing from the card database", name)
			continue
		}
		perm := interactive.PermanentState{Name: name, IsLand: true, SubTypes: subtypes}
		if got := permanentArtName(perm, printed); got != name {
			t.Errorf("permanentArtName(%s) = %q, want %q", name, got, name)
		}
	}
}

// Blood Moon-style effects still override a dual land's art.
func TestPermanentArtName_DualLandTurnedIntoMountain(t *testing.T) {
	printed := domain.FindCardByName("Underground Sea")
	if printed == nil {
		t.Fatal("Underground Sea missing from the card database")
	}
	perm := interactive.PermanentState{Name: "Underground Sea", IsLand: true, SubTypes: "Mountain"}

	if got := permanentArtName(perm, printed); got != "Mountain" {
		t.Fatalf("permanentArtName() = %q, want %q", got, "Mountain")
	}
}
