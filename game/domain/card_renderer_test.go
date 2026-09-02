package domain

import (
	"image"
	"image/color"
	"testing"
)

func TestGetFrameFilename(t *testing.T) {
	tests := []struct {
		name     string
		card     *Card
		expected string
	}{
		{
			name: "White Creature",
			card: &Card{
				Colors:   []string{"W"},
				TypeLine: "Creature — Angel",
			},
			expected: "Cardbk_White.pic.png",
		},
		{
			name: "Blue Instant",
			card: &Card{
				Colors:   []string{"U"},
				TypeLine: "Instant",
			},
			expected: "Cardbk_Blue.pic.png",
		},
		{
			name: "Black Sorcery",
			card: &Card{
				Colors:   []string{"B"},
				TypeLine: "Sorcery",
			},
			expected: "Cardbk_Black.pic.png",
		},
		{
			name: "Red Instant",
			card: &Card{
				Colors:   []string{"R"},
				TypeLine: "Instant",
			},
			expected: "Cardbk_Red.pic.png",
		},
		{
			name: "Green Creature",
			card: &Card{
				Colors:   []string{"G"},
				TypeLine: "Creature — Elf",
			},
			expected: "Cardbk_Green.pic.png",
		},
		{
			name: "Gold Multicolored",
			card: &Card{
				Colors:   []string{"U", "B"},
				TypeLine: "Creature",
			},
			expected: "Cardbk_Gold.pic.png",
		},
		{
			name: "Artifact Creature",
			card: &Card{
				Colors:   []string{},
				TypeLine: "Artifact Creature — Construct",
			},
			expected: "Cardbk_Artifact.pic.png",
		},
		{
			name: "Basic Land Mountain",
			card: &Card{
				Colors:   []string{},
				TypeLine: "Basic Land — Mountain",
			},
			expected: "Cardbk_Redland.pic.png",
		},
		{
			name: "Arabian Nights Land",
			card: &Card{
				CardSet:  CardSet{SetID: "arn"},
				Colors:   []string{},
				TypeLine: "Land",
			},
			expected: "Cardbk_Arabiannightsland.pic.png",
		},
		{
			name: "Antiquities Land",
			card: &Card{
				CardSet:  CardSet{SetID: "atq"},
				Colors:   []string{},
				TypeLine: "Land",
			},
			expected: "Cardbk_Antiquitiesland.pic.png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetFrameFilename(tt.card)
			if got != tt.expected {
				t.Errorf("GetFrameFilename() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestExtractArtSubImage(t *testing.T) {
	// Full image of 228x325
	img := image.NewRGBA(image.Rect(0, 0, 228, 325))
	for y := range 325 {
		for x := range 228 {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 100, A: 255})
		}
	}

	art := ExtractArtSubImage(img)
	if art == nil {
		t.Fatalf("ExtractArtSubImage() returned nil")
	}

	bounds := art.Bounds()
	if bounds.Dx() <= 0 || bounds.Dy() <= 0 {
		t.Errorf("ExtractArtSubImage() invalid bounds: %v", bounds)
	}
}

func TestRenderResizedCard(t *testing.T) {
	card := &Card{
		cardID:   "test-card-1",
		CardName: "Lightning Bolt",
		ManaCost: "{R}",
		Colors:   []string{"R"},
		TypeLine: "Instant",
		Text:     "Lightning Bolt deals 3 damage to any target.",
		CardSet:  CardSet{SetID: "4ed"},
	}

	// Test 100px width (battlefield permanent size)
	img100 := RenderResizedCard(card, 100, CardViewArtOnly)
	if img100 == nil {
		t.Fatalf("RenderResizedCard(100, CardViewArtOnly) returned nil")
	}
	if img100.Bounds().Dx() != 100 {
		t.Errorf("RenderResizedCard(100) width = %d, want 100", img100.Bounds().Dx())
	}
	if img100.Bounds().Dy() <= 0 {
		t.Errorf("RenderResizedCard(100) invalid height = %d", img100.Bounds().Dy())
	}

	// Test 80px width (hand card size)
	img80 := RenderResizedCard(card, 80, CardViewArtOnly)
	if img80 == nil {
		t.Fatalf("RenderResizedCard(80, CardViewArtOnly) returned nil")
	}
	if img80.Bounds().Dx() != 80 {
		t.Errorf("RenderResizedCard(80) width = %d, want 80", img80.Bounds().Dx())
	}

	// Test caching returns same image pointer
	img100Cached := RenderResizedCard(card, 100, CardViewArtOnly)
	if img100Cached != img100 {
		t.Errorf("RenderResizedCard did not return cached image pointer")
	}
}

func TestCardResizedImageMethod(t *testing.T) {
	card := &Card{
		cardID:    "test-serra-angel",
		CardName:  "Serra Angel",
		ManaCost:  "{3}{W}{W}",
		Colors:    []string{"W"},
		TypeLine:  "Creature — Angel",
		Text:      "Flying, vigilance",
		Power:     4,
		Toughness: 4,
		CardSet:   CardSet{SetID: "2ed"},
	}

	resized, err := card.ResizedImage(100, CardViewArtOnly)
	if err != nil {
		t.Fatalf("card.ResizedImage() error = %v", err)
	}
	if resized == nil {
		t.Fatalf("card.ResizedImage() returned nil")
	}
	if resized.Bounds().Dx() != 100 {
		t.Errorf("card.ResizedImage() width = %d, want 100", resized.Bounds().Dx())
	}
}

func TestCardImage_ArtOnlyConstructsImage(t *testing.T) {
	card := &Card{
		cardID:   "test-dark-ritual",
		CardName: "Dark Ritual",
		ManaCost: "{B}",
		Colors:   []string{"B"},
		TypeLine: "Instant",
		Text:     "Add {B}{B}{B}.",
		CardSet:  CardSet{SetID: "4ed"},
	}

	artCard, err := card.CardImage(CardViewArtOnly)
	if err != nil {
		t.Fatalf("card.CardImage(CardViewArtOnly) error = %v", err)
	}
	if artCard == nil {
		t.Fatalf("card.CardImage(CardViewArtOnly) returned nil")
	}
	if artCard.Bounds().Dx() != CardFullWidth {
		t.Errorf("card.CardImage(CardViewArtOnly) width = %d, want %d", artCard.Bounds().Dx(), CardFullWidth)
	}
}
