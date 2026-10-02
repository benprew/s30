package domain

import (
	"fmt"
	"image"
	"image/color"
	"math"
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
			got := getFrameFilename(tt.card)
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
	if artCard.Bounds().Dx() != CardArtWidth {
		t.Errorf("card.CardImage(CardViewArtOnly) width = %d, want %d", artCard.Bounds().Dx(), CardArtWidth)
	}
}

func TestRendererDoesNotFetchOrCacheCards(t *testing.T) {
	ClearCardImageCache()
	t.Cleanup(ClearCardImageCache)
	card := &Card{cardID: "renderer-only", CardName: "Test", ArtURL: "http://invalid.example/art"}
	art := image.NewRGBA(image.Rect(0, 0, 186, 118))
	first := renderCardImage(card, art, CardViewFull)
	second := renderCardImage(card, nil, CardViewFull)
	if first == nil || second == nil || first == second {
		t.Fatal("renderer did not create separate images")
	}
	if card.ImageLoaded() || len(cardImageVariants.entries) != 0 || len(cardImageVariants.fetches) != 0 {
		t.Fatal("renderer changed cache or fetch state")
	}
}

func TestRendererUsesRequestedViewSize(t *testing.T) {
	card := &Card{CardName: "Serra Angel", ManaCost: "{3}{W}{W}", TypeLine: "Creature — Angel", Text: "Flying, vigilance", Power: 4, Toughness: 4}
	for _, view := range []CardView{CardViewFull, CardViewFullMini, CardViewArtOnly, CardViewArtMini, {300, 420}} {
		t.Run(fmt.Sprint(view), func(t *testing.T) {
			img := renderCardImage(card, nil, view)
			if img == nil || img.Bounds().Size() != image.Point(view) {
				t.Fatalf("wrong rendered size for %v", view)
			}
		})
	}
}

func TestExtractArtSubImageUsesDocumentedLayout(t *testing.T) {
	for _, size := range []image.Point{{200, 300}, {400, 600}, {300, 420}} {
		img := image.NewRGBA(image.Rectangle{Max: size})
		got := ExtractArtSubImage(img).Bounds()
		want := image.Rect(int(math.Round(21*float64(size.X)/200)), int(math.Round(25*float64(size.Y)/300)), int(math.Round(181*float64(size.X)/200)), int(math.Round(164*float64(size.Y)/300)))
		if got != want {
			t.Errorf("size %v: bounds = %v, want %v", size, got, want)
		}
	}
}

func TestCardContrastOverlay(t *testing.T) {
	for _, view := range []CardView{CardViewFull, CardViewFullMini, CardViewArtOnly, CardViewArtMini} {
		artOnly := view == CardViewArtOnly || view == CardViewArtMini
		height := 300.0
		if artOnly {
			height = float64(cardSourceArtHeight)
		}
		overlay := cardContrastOverlay(view, float64(view.X)/200, float64(view.Y)/height)
		for _, p := range []image.Point{{0, 0}, {view.X - 1, 0}, {0, view.Y - 1}, {view.X - 1, view.Y - 1}} {
			if got := overlay.RGBAAt(p.X, p.Y); got != (color.RGBA{A: 255}) {
				t.Errorf("view %v edge %v = %v", view, p, got)
			}
		}
		if !artOnly {
			p := image.Pt(view.X/2, int(225*float64(view.Y)/height))
			if got := overlay.RGBAAt(p.X, p.Y); got.A != 0 {
				t.Errorf("overlay changes the rules panel: %v", got)
			}
		}
		for _, y := range []float64{14, 172} {
			if artOnly && y > 164 {
				continue
			}
			if got := overlay.RGBAAt(view.X/2, int(y*float64(view.Y)/height)); got.A != 0 {
				t.Errorf("overlay changes the label background: %v", got)
			}
		}
		if got := overlay.RGBAAt(view.X/2, int(80*float64(view.Y)/height)); got.A != 0 {
			t.Errorf("overlay covers artwork: %v", got)
		}
	}
}
