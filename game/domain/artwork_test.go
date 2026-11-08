package domain

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"slices"
	"testing"

	"github.com/klauspost/compress/zstd"
)

func TestFactoryIncludesAllAntiquitiesSeasons(t *testing.T) {
	card := FindCardByName("Mishra's Factory")
	if card == nil {
		t.Fatal("Mishra's Factory is missing")
	}
	var numbers []string
	for _, art := range card.Artworks {
		if art.SetID == "atq" {
			numbers = append(numbers, art.CollectorNo)
		}
	}
	slices.Sort(numbers)
	if !slices.Equal(numbers, []string{"80a", "80b", "80c", "80d"}) {
		t.Fatalf("Factory seasons = %v, want 80a, 80b, 80c, 80d", numbers)
	}
}

func TestCardDataProvidesAlternateArtwork(t *testing.T) {
	t.Cleanup(ClearCardImageCache)
	data, err := json.Marshal([]CardJSON{{
		CardName: "Lightning Bolt", ManaCost: "{R}",
		Artworks: []ArtworkJSON{
			{SetID: "leb", CollectorNo: "161", BorderCropURL: "https://example.test/leb.jpg"},
			{SetID: "m10", SetName: "Magic 2010", CollectorNo: "146", Artist: "Alternate Artist", PngURL: "https://example.test/m10.png", ArtURL: "https://example.test/m10-crop.jpg", BorderCropURL: "https://example.test/m10.jpg"},
		},
	}})
	if err != nil {
		t.Fatal(err)
	}
	compressor, err := zstd.NewWriter(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer compressor.Close()
	cards := LoadCardDatabase(bytes.NewReader(compressor.EncodeAll(data, nil)))
	if len(cards) != 1 {
		t.Fatal("artwork added a second rules definition")
	}
	card := cards[0]
	if card.CardID() != "Lightning Bolt" {
		t.Fatalf("card identity depends on its printing: %q", card.CardID())
	}
	arts := card.Artworks
	if len(arts) != 2 || arts[1].ID != "m10-146-lightning-bolt" || arts[1].SetID != "m10" {
		t.Fatalf("artwork not loaded from card data: %+v", arts)
	}
	owned := NewOwnedCard(card)
	owned.ArtworkID = arts[1].ID
	printing := owned.Artwork()
	if printing.BorderCropURL != card.Artworks[1].BorderCropURL || printing.ArtURL != "https://example.test/m10-crop.jpg" {
		t.Fatal("artwork must change the image source and retain the card rules")
	}
	archive := cardImageArchive(t, map[string]image.Image{
		"leb-161-200-lightning-bolt.jpg": solidCardImage(color.RGBA{R: 255, A: 255}),
		"m10-146-200-lightning-bolt.jpg": solidCardImage(color.RGBA{B: 255, A: 255}),
	})
	if loaded, err := loadCardImagesFromArchive(archive); err != nil || loaded != 2 {
		t.Fatalf("load artwork archive: %d, %v", loaded, err)
	}
	got, _ := owned.CardImage(CardViewFull)
	want, _ := printing.CardImage(CardViewFull)
	if got != want {
		t.Fatal("owned card did not use its artwork image")
	}
	cc := NewCardCollection()
	cc.AddOwnedCard(owned)
	priority := CollectPriorityCards(&Player{Character: Character{CardCollection: cc}})
	if len(priority) != 1 || priority[0].ID != arts[1].ID {
		t.Fatal("preloading ignored the owned artwork")
	}
	owned.ArtworkID = "removed-1-lightning-bolt"
	fallback, _ := owned.CardImage(CardViewFull)
	defaultImage, _ := card.CardImage(CardViewFull)
	if fallback != defaultImage || owned.ArtworkID != "removed-1-lightning-bolt" {
		t.Fatal("missing artwork must use the default image and retain its ID")
	}
}

func TestCardIdentityDoesNotDependOnArtwork(t *testing.T) {
	first := (&CardJSON{CardName: "Forest", SetID: "leb", CollectorNo: "300"}).ToCard()
	second := (&CardJSON{CardName: "Forest", SetID: "unh", CollectorNo: "140"}).ToCard()
	if first.CardID() != "Forest" || second.CardID() != first.CardID() {
		t.Fatal("printing changed card identity")
	}
}

func TestOwnedCardLoadsLegacyPrintingReference(t *testing.T) {
	card := FindCardByName("Lightning Bolt")
	for _, data := range []string{
		`{"CardID":"leb-162-lightning-bolt"}`,
		`{"CardName":"Lightning Bolt","SetID":"leb","CollectorNo":"162"}`,
		`{"id":"old-copy","card":{"CardID":"leb-162-lightning-bolt"},"deck_index":0}`,
	} {
		var owned OwnedCard
		if err := json.Unmarshal([]byte(data), &owned); err != nil {
			t.Fatal(err)
		}
		if owned.Card != card || owned.ArtworkID != "leb-162-lightning-bolt" {
			t.Fatalf("legacy copy lost its printing: %+v", owned)
		}
	}
}
