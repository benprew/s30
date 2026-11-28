package domain

import "testing"

// DeckShortfall is what the deck editor warns about before closing: the player
// should see how far their active deck is from the minimum, not a silent pad in
// the duel. The minimum itself comes from the difficulty, so the test pins one.
func TestDeckShortfallCountsTheGapToTheMinimum(t *testing.T) {
	cases := []struct {
		name   string
		inDeck int
		want   int
	}{
		{"empty deck needs the whole minimum", 0, 36},
		{"one card short", 35, 1},
		{"exactly at the minimum", 36, 0},
		{"above the minimum", 40, 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plains := FindCardByName("Plains")
			collection := NewCardCollection()
			if tc.inDeck > 0 {
				collection.AddCard(plains, tc.inDeck, 0)
			}
			player := &Player{
				Character:   Character{CardCollection: collection},
				MinDeckSize: 36,
				ActiveDeck:  0,
			}

			if got := player.DeckShortfall(); got != tc.want {
				t.Errorf("DeckShortfall() = %d, want %d", got, tc.want)
			}
		})
	}
}

// The deck counted is the active one, which is the deck the editor edits and the
// duel plays. A shortfall read off another deck would warn about the wrong thing.
func TestDeckShortfallReadsTheActiveDeck(t *testing.T) {
	plains := FindCardByName("Plains")
	collection := NewCardCollection()
	collection.AddCard(plains, 40, 1)
	player := &Player{
		Character:   Character{CardCollection: collection},
		MinDeckSize: 36,
		ActiveDeck:  0,
	}

	if got := player.DeckShortfall(); got != 36 {
		t.Errorf("DeckShortfall() = %d, want 36 for an empty active deck", got)
	}
}

func TestPlayerGetActiveDeckRespectsActiveDeck(t *testing.T) {
	plains := FindCardByName("Plains")
	forest := FindCardByName("Forest")
	collection := NewCardCollection()
	collection.AddCard(plains, 40, 0)
	collection.AddCard(forest, 40, 1)

	player := &Player{
		Character:   Character{CardCollection: collection},
		MinDeckSize: 36,
		ActiveDeck:  1,
	}

	deck := player.GetActiveDeck()
	if deck[forest] != 40 {
		t.Errorf("expected 40 forests in active deck 1, got %d", deck[forest])
	}
	if deck[plains] != 0 {
		t.Errorf("expected 0 plains in active deck 1, got %d", deck[plains])
	}
}
