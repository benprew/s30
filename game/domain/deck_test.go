package domain

import "testing"

func TestValidAnteCardsExcludesBasicLands(t *testing.T) {
	land := &Card{CardType: CardTypeLand}
	creature := &Card{CardType: CardTypeCreature}
	deck := Deck{land: 1, creature: 1}

	cards := deck.ValidAnteCards(ExcludeBasicLand)
	if len(cards) != 1 {
		t.Errorf("Expected 1 card, got %d", len(cards))
	}
	if cards[0] != creature {
		t.Error("Expected creature card")
	}
}

func TestValidAnteCardsNoExclusions(t *testing.T) {
	land := &Card{CardType: CardTypeLand}
	creature := &Card{CardType: CardTypeCreature}
	deck := Deck{land: 1, creature: 1}

	cards := deck.ValidAnteCards()
	if len(cards) != 2 {
		t.Errorf("Expected 2 cards, got %d", len(cards))
	}
}

func TestValidAnteCardsExcludesVintageRestricted(t *testing.T) {
	restricted := &Card{CardType: CardTypeCreature, VintageRestricted: true}
	normal := &Card{CardType: CardTypeCreature}
	deck := Deck{restricted: 1, normal: 1}

	cards := deck.ValidAnteCards(ExcludeVintageRestricted)
	if len(cards) != 1 {
		t.Errorf("Expected 1 card, got %d", len(cards))
	}
	if cards[0] != normal {
		t.Error("Expected normal card, not vintage restricted")
	}
}

func TestValidAnteCardsMultipleExclusions(t *testing.T) {
	land := &Card{CardType: CardTypeLand}
	restricted := &Card{CardType: CardTypeCreature, VintageRestricted: true}
	normal := &Card{CardType: CardTypeCreature}
	deck := Deck{land: 1, restricted: 1, normal: 1}

	cards := deck.ValidAnteCards(ExcludeBasicLand, ExcludeVintageRestricted)
	if len(cards) != 1 {
		t.Errorf("Expected 1 card, got %d", len(cards))
	}
	if cards[0] != normal {
		t.Error("Expected normal creature card")
	}
}
