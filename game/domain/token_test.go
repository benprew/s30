package domain

import "testing"

func TestFindTokenByNameWasp(t *testing.T) {
	wasp := FindTokenByName("Wasp")
	if wasp == nil {
		t.Fatal("FindTokenByName(\"Wasp\") = nil, want the token created by The Hive")
	}
	if wasp.Power != 1 || wasp.Toughness != 1 {
		t.Errorf("Wasp = %d/%d, want 1/1", wasp.Power, wasp.Toughness)
	}
	if wasp.BorderCropURL == "" {
		t.Error("Wasp has no BorderCropURL, so no art can be fetched for it")
	}
}

func TestFindTokenByNameUnknown(t *testing.T) {
	if got := FindTokenByName("Not A Token"); got != nil {
		t.Errorf("FindTokenByName(\"Not A Token\") = %v, want nil", got)
	}
}

// Tokens are never owned, traded or sold, so they must stay out of CARDS, or
// they would show up in shops, the collection and deck lists.
func TestTokensAreNotInCardDatabase(t *testing.T) {
	for _, token := range TOKENS {
		if card := FindCardByName(token.CardName); card != nil {
			t.Errorf("token %q is also in CARDS", token.CardName)
		}
	}
}

// The image cache is keyed by cardID, so a shared id would draw one card's art
// for the other.
func TestTokensHaveUniqueCardIDs(t *testing.T) {
	seen := make(map[string]string, len(TOKENS))
	for _, token := range TOKENS {
		if other, ok := seen[token.cardID]; ok {
			t.Errorf("tokens %q and %q share cardID %q", other, token.CardName, token.cardID)
		}
		seen[token.cardID] = token.CardName
	}
	for _, card := range CARDS {
		if name, ok := seen[card.cardID]; ok {
			t.Errorf("token %q shares cardID %q with card %q", name, card.cardID, card.CardName)
		}
	}
}
