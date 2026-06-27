package domain

import "testing"

func deckOf(cards ...*Card) Deck {
	d := make(Deck)
	for _, c := range cards {
		d[c]++
	}
	return d
}

var (
	cardBlackBear = &Card{CardName: "Bear", Colors: []string{"G"}, CardType: CardTypeCreature, ManaCost: "{1}{G}"}
	cardRedBolt   = &Card{CardName: "Bolt", Colors: []string{"R"}, CardType: CardTypeInstant, ManaCost: "{R}"}
	cardBlueDraw  = &Card{CardName: "Draw", Colors: []string{"U"}, CardType: CardTypeSorcery, ManaCost: "{2}{U}"}
	cardFatGreen  = &Card{CardName: "Wurm", Colors: []string{"G"}, CardType: CardTypeCreature, ManaCost: "{5}{G}{G}"}
	cardForest    = &Card{CardName: "Forest", CardType: CardTypeLand}
	cardGoldGolem = &Card{CardName: "Golem", CardType: CardTypeCreature, ManaCost: "{3}"}
)

func TestValidateMonoColor(t *testing.T) {
	q := &Quest{Constraint: ConstraintMonoColor}

	// All green non-lands (+ a land) → mono.
	if ok, reason := ValidateDeckConstraint(q, deckOf(cardBlackBear, cardFatGreen, cardForest)); !ok {
		t.Errorf("mono-green deck should pass, got %q", reason)
	}
	// Green + red → not mono.
	if ok, _ := ValidateDeckConstraint(q, deckOf(cardBlackBear, cardRedBolt)); ok {
		t.Error("two-color deck should fail mono-color")
	}
	// Targeted mono color: require red specifically.
	qr := &Quest{Constraint: ConstraintMonoColor, Color: ColorRed}
	if ok, _ := ValidateDeckConstraint(qr, deckOf(cardBlackBear)); ok {
		t.Error("mono-green deck should fail a mono-red requirement")
	}
	if ok, reason := ValidateDeckConstraint(qr, deckOf(cardRedBolt)); !ok {
		t.Errorf("mono-red deck should satisfy mono-red requirement, got %q", reason)
	}
}

func TestValidateFatDeck(t *testing.T) {
	q := &Quest{Constraint: ConstraintFatDeck, ConstraintN: 3}
	big := Deck{cardBlackBear: 2, cardForest: 2} // 4 cards
	if ok, reason := ValidateDeckConstraint(q, big); !ok {
		t.Errorf("4-card deck should pass N=3, got %q", reason)
	}
	small := Deck{cardBlackBear: 2, cardForest: 1} // 3 cards
	if ok, _ := ValidateDeckConstraint(q, small); ok {
		t.Error("3-card deck should fail N=3 (must exceed it)")
	}
}

func TestValidateLowCurve(t *testing.T) {
	q := &Quest{Constraint: ConstraintLowCurve, ConstraintN: 3}
	// Bear (MV2) + colorless golem (MV3) + a non-creature high-cost spell are fine.
	ok, reason := ValidateDeckConstraint(q, deckOf(cardBlackBear, cardGoldGolem, cardBlueDraw))
	if !ok {
		t.Errorf("creatures within curve should pass, got %q", reason)
	}
	// Wurm is MV7 → fails.
	if ok, _ := ValidateDeckConstraint(q, deckOf(cardBlackBear, cardFatGreen)); ok {
		t.Error("MV7 creature should fail low-curve N=3")
	}
}

func TestValidateColorLight(t *testing.T) {
	q := &Quest{Constraint: ConstraintColorLight, Color: ColorBlue}
	if ok, reason := ValidateDeckConstraint(q, deckOf(cardBlackBear, cardRedBolt, cardForest)); !ok {
		t.Errorf("no-blue deck should pass, got %q", reason)
	}
	if ok, _ := ValidateDeckConstraint(q, deckOf(cardBlackBear, cardBlueDraw)); ok {
		t.Error("deck containing a blue card should fail color-light blue")
	}
}

func TestValidateNoAttackingIsDeckAgnostic(t *testing.T) {
	q := &Quest{Constraint: ConstraintNoAttacking}
	if ok, _ := ValidateDeckConstraint(q, deckOf(cardRedBolt, cardFatGreen)); !ok {
		t.Error("no-attacking is not a deck check; should always pass deck validation")
	}
}
