package domain

import (
	"encoding/json"
	"testing"
)

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Juzám Djinn",
			input:    "Juzám Djinn",
			expected: "juzám-djinn",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// Cards in saved games (CardsForSale, BonusDuelCards, dungeon rewards, etc.)
// must serialize as just their CardID and reload from the embedded CARDS
// database. Otherwise distinct cards collapse to a single image cache key and
// the buy-cards screen shows one image for every card.
func TestCardMarshalJSONOnlyEmitsCardID(t *testing.T) {
	if len(CARDS) == 0 {
		t.Fatal("no cards loaded")
	}
	card := CARDS[0]

	data, err := json.Marshal(card)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("could not parse marshaled card: %v", err)
	}

	if got, want := len(raw), 1; got != want {
		t.Errorf("Card JSON should have exactly 1 field (CardID), got %d: %s", got, data)
	}
	if id, _ := raw["CardID"].(string); id != card.CardID() {
		t.Errorf("CardID field = %q, want %q", id, card.CardID())
	}
}

func TestCardJSONRoundTripRestoresFullCard(t *testing.T) {
	if len(CARDS) == 0 {
		t.Fatal("no cards loaded")
	}
	original := CARDS[0]

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var restored Card
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if restored.CardID() != original.CardID() {
		t.Errorf("cardID = %q, want %q", restored.CardID(), original.CardID())
	}
	if restored.CardName != original.CardName {
		t.Errorf("CardName = %q, want %q", restored.CardName, original.CardName)
	}
	if restored.PngURL != original.PngURL {
		t.Errorf("PngURL = %q, want %q", restored.PngURL, original.PngURL)
	}
	if restored.BorderCropURL != original.BorderCropURL {
		t.Errorf("BorderCropURL = %q, want %q", restored.BorderCropURL, original.BorderCropURL)
	}
	if restored.ManaCost != original.ManaCost {
		t.Errorf("ManaCost = %q, want %q", restored.ManaCost, original.ManaCost)
	}
}

func TestCardSliceJSONRoundTripPreservesDistinctCardIDs(t *testing.T) {
	if len(CARDS) < 3 {
		t.Skip("need at least 3 cards for this test")
	}
	originals := []*Card{CARDS[0], CARDS[1], CARDS[2]}

	data, err := json.Marshal(originals)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var restored []*Card
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if len(restored) != len(originals) {
		t.Fatalf("expected %d cards after round trip, got %d", len(originals), len(restored))
	}
	for i, want := range originals {
		if restored[i].CardID() != want.CardID() {
			t.Errorf("card %d: cardID = %q, want %q", i, restored[i].CardID(), want.CardID())
		}
		if restored[i].CardName != want.CardName {
			t.Errorf("card %d: CardName = %q, want %q", i, restored[i].CardName, want.CardName)
		}
	}
}

// Saves written before Card switched to CardID-only serialization include the
// full card struct. Make sure they still load.
func TestCardUnmarshalJSONLegacySaveFormat(t *testing.T) {
	if len(CARDS) == 0 {
		t.Fatal("no cards loaded")
	}
	original := CARDS[0]

	legacy, err := json.Marshal(struct {
		CardName    string
		SetID       string
		SetName     string
		CollectorNo string
		PngURL      string
		ManaCost    string
	}{
		CardName:    original.CardName,
		SetID:       original.SetID,
		SetName:     original.SetName,
		CollectorNo: original.CollectorNo,
		PngURL:      "stale-url-from-old-save",
		ManaCost:    "stale",
	})
	if err != nil {
		t.Fatalf("marshal legacy failed: %v", err)
	}

	var restored Card
	if err := json.Unmarshal(legacy, &restored); err != nil {
		t.Fatalf("unmarshal legacy failed: %v", err)
	}
	if restored.CardID() != original.CardID() {
		t.Errorf("legacy cardID = %q, want %q", restored.CardID(), original.CardID())
	}
	if restored.PngURL != original.PngURL {
		t.Errorf("legacy load should overwrite stale fields with canonical data; PngURL = %q, want %q", restored.PngURL, original.PngURL)
	}
}

func TestCardUnmarshalJSONOldVersionMissingInCurrentDatabase(t *testing.T) {
	card := FindCardByName("Lightning Bolt")
	if card == nil {
		t.Fatal("Lightning Bolt not found")
	}

	data := []byte(`{"CardID": "4ed-999-lightning-bolt"}`)
	var restored Card
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("unmarshal old card version failed: %v", err)
	}
	if restored.CardName != "Lightning Bolt" {
		t.Errorf("CardName = %q, want %q", restored.CardName, "Lightning Bolt")
	}
	if restored.CardID() != card.CardID() {
		t.Errorf("CardID = %q, want %q", restored.CardID(), card.CardID())
	}
}

func TestCardUnmarshalJSONLegacySaveWithMissingSetID(t *testing.T) {
	card := FindCardByName("Lightning Bolt")
	if card == nil {
		t.Fatal("Lightning Bolt not found")
	}

	data := []byte(`{"CardName": "Lightning Bolt", "SetID": "4ed", "CollectorNo": "999"}`)
	var restored Card
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("unmarshal legacy card with missing set_id failed: %v", err)
	}
	if restored.CardName != "Lightning Bolt" {
		t.Errorf("CardName = %q, want %q", restored.CardName, "Lightning Bolt")
	}
	if restored.CardID() != card.CardID() {
		t.Errorf("CardID = %q, want %q", restored.CardID(), card.CardID())
	}
}

func TestFindCardBySanitizedName(t *testing.T) {
	card := FindCardBySanitizedName("lightning-bolt")
	if card == nil || card.CardName != "Lightning Bolt" {
		t.Errorf("expected Lightning Bolt, got %v", card)
	}
}

func TestFindCard(t *testing.T) {
	bolt := FindCardByName("Lightning Bolt")
	if bolt == nil {
		t.Fatal("Lightning Bolt not found")
	}

	if got := FindCard(bolt.CardID(), ""); got != bolt {
		t.Errorf("FindCard with canonical ID = %v, want %v", got, bolt)
	}
	if got := FindCard("4ed-999-lightning-bolt", ""); got != bolt {
		t.Errorf("FindCard with old ID = %v, want %v", got, bolt)
	}
	if got := FindCard("4ed-999-lightning-bolt", "Lightning Bolt"); got != bolt {
		t.Errorf("FindCard with old ID and name = %v, want %v", got, bolt)
	}
	if got := FindCard("", "Lightning Bolt"); got != bolt {
		t.Errorf("FindCard with name alone = %v, want %v", got, bolt)
	}
	if got := FindCard("unknown-id-123", "nonexistent-card-name"); got != nil {
		t.Errorf("FindCard with nonexistent card = %v, want nil", got)
	}
}

func TestCardUnmarshalJSONRemovedCardLeavesReceiverEmpty(t *testing.T) {
	data := []byte(`{"CardID": "2ed-136-word-of-command", "CardName": "Word of Command"}`)
	var restored Card
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("unmarshal removed card failed: %v", err)
	}
	if restored.CardID() != "" {
		t.Errorf("expected empty CardID for removed card, got %q", restored.CardID())
	}
	if restored.CardName != "" {
		t.Errorf("expected empty CardName for removed card, got %q", restored.CardName)
	}
}

