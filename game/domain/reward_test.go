package domain

import "testing"

func TestDeckPrimaryColor(t *testing.T) {
	mountain := FindCardByName("Mountain")
	lightningBolt := FindCardByName("Lightning Bolt")
	forest := FindCardByName("Forest")

	deck := Deck{
		mountain:      4,
		lightningBolt: 4,
		forest:        1,
	}

	if got := DeckPrimaryColor(deck); got != ColorRed {
		t.Errorf("DeckPrimaryColor = %v, want ColorRed", got)
	}
}

func TestDeckPrimaryColor_Colorless(t *testing.T) {
	if got := DeckPrimaryColor(Deck{}); got != ColorColorless {
		t.Errorf("DeckPrimaryColor of empty deck = %v, want ColorColorless", got)
	}
}

func TestRandomBasicLand(t *testing.T) {
	land := RandomBasicLand()
	if land == nil {
		t.Fatal("RandomBasicLand returned nil")
	}
	if _, ok := basicLands[land.CardName]; !ok {
		t.Errorf("RandomBasicLand returned %q, which is not a basic land", land.CardName)
	}
}

func TestRewardChoices(t *testing.T) {
	mountain := FindCardByName("Mountain")
	lightningBolt := FindCardByName("Lightning Bolt")
	giantGrowth := FindCardByName("Giant Growth")

	deck := Deck{mountain: 4, lightningBolt: 4}

	choices := RewardChoices(deck, giantGrowth)

	if len(choices) != 3 {
		t.Fatalf("expected 3 choices, got %d", len(choices))
	}

	if choices[0] != giantGrowth {
		t.Errorf("first choice = %q, want the opponent ante card %q", choices[0].Name(), giantGrowth.Name())
	}

	// Second choice should be a card in the player's primary color (red).
	if !cardMatchesColorOrColorless(choices[1], ColorRed) {
		t.Errorf("second choice %q does not match player's red color", choices[1].Name())
	}

	// Third choice should be a basic land.
	if _, ok := basicLands[choices[2].CardName]; !ok {
		t.Errorf("third choice %q is not a basic land", choices[2].Name())
	}
}

func TestRewardChoices_NoOpponentCard(t *testing.T) {
	mountain := FindCardByName("Mountain")
	deck := Deck{mountain: 4}

	choices := RewardChoices(deck, nil)
	for _, c := range choices {
		if c == nil {
			t.Fatal("RewardChoices returned a nil card")
		}
	}
}
