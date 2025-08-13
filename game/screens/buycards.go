package screens

import (
	"archive/zip"
	"bytes"
	"fmt"
	"image"
	"image/color"
	"io"

	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/sprites"
	"github.com/benprew/s30/game/ui/elements"
	"github.com/benprew/s30/game/ui/fonts"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// For buycards I need:
// 1. cards
// 2. done button
// 3. prices
// 4. player money
// 4.1 player card collection
// 5. Clicking on a card and choosing to buy it

type BuyCardsScreen struct {
	Buttons     []*elements.Button
	City        *domain.City
	BgImage     *ebiten.Image
	ScreenTitle *ebiten.Image
}

func (s *BuyCardsScreen) IsFramed() bool {
	return true
}

func NewBuyCardsScreen(city *domain.City) *BuyCardsScreen {
	drawOpts := &ebiten.DrawImageOptions{}
	drawOpts.GeoM.Scale(SCALE, SCALE)
	img, _, err := image.Decode(bytes.NewReader(assets.BuyCards_png))
	if err != nil {
		panic(fmt.Sprintf("Unable to load BuyCards.png: %s", err))
	}

	sprite := loadButtonMap(assets.BuyCardsSprite_png, assets.BuyCardsSpriteMap_json)
	fontFace := &text.GoTextFace{
		Source: fonts.MtgFont,
		Size:   16,
	}
	cardsForSale := ebiten.NewImageFromImage(sprite[4])
	txt := "Cards for Sale"
	textX, textY := elements.AlignText(cardsForSale, txt, fontFace, 0, 0)
	options := &ebiten.DrawImageOptions{}
	options.GeoM.Translate(textX, textY)

	text.Draw(cardsForSale, "Cards for Sale", fontFace, &text.DrawOptions{DrawImageOptions: *options})

	return &BuyCardsScreen{
		Buttons:     mkCardButtons(SCALE, city),
		City:        city,
		BgImage:     ebiten.NewImageFromImage(img),
		ScreenTitle: cardsForSale,
	}
}

func (s *BuyCardsScreen) Draw(screen *ebiten.Image, W, H int, scale float64) {
	cityOpts := &ebiten.DrawImageOptions{}
	cityOpts.GeoM.Scale(SCALE, SCALE)
	cityOpts.GeoM.Translate(100.0, 75.0) // Offset the image
	screen.DrawImage(s.BgImage, cityOpts)

	titleOpts := &ebiten.DrawImageOptions{}
	titleOpts.GeoM.Scale(SCALE, SCALE)
	titleOpts.GeoM.Translate(1024/2.0, 100.0) // Offset the image
	screen.DrawImage(s.ScreenTitle, titleOpts)

	frameOpts := &ebiten.DrawImageOptions{}
	frameOpts.GeoM.Scale(scale, scale)
	for _, b := range s.Buttons {
		b.Draw(screen, frameOpts, scale)
	}
}

func (s *BuyCardsScreen) Update(W, H int, scale float64) (screenui.ScreenName, error) {
	options := &ebiten.DrawImageOptions{}
	for i := range s.Buttons {
		b := s.Buttons[i]
		b.Update(options, scale)
		if b.ButtonID == "done" && b.State == elements.StateClicked {
			return screenui.CityScr, nil
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return screenui.CityScr, nil
	}
	return screenui.BuyCardsScr, nil
}

func mkCardButtons(scale float64, city *domain.City) []*elements.Button {
	sprite := loadButtonMap(assets.BuyCardsSprite_png, assets.BuyCardsSpriteMap_json)
	fontFace := &text.GoTextFace{
		Source: fonts.MtgFont,
		Size:   32,
	}

	city.MkCards()

	// make card buttons
	cards := make([]*elements.Button, 0)
	for i, cardIdx := range city.CardsForSale {
		card := domain.CARDS[cardIdx]
		filename := card.Filename()
		fmt.Println("BuyCards: Loading card images")
		data, err := readFromEmbeddedZip(assets.CardImages_zip, "carddata/"+filename)
		if err != nil {
			fmt.Printf("failed to read image: %w", err)
			continue
		}

		cardPng, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			fmt.Printf("failed to decode image: %w", err)
			continue
		}

		cardImg := ebiten.NewImageFromImage(cardPng)
		cardUpperImg := ebiten.NewImage(300, 220)
		cardUpperImg.DrawImage(cardImg, &ebiten.DrawImageOptions{})
		fmt.Println("card bounds", cardUpperImg.Bounds())

		// Create price label with same background as "Cards for Sale"
		priceLabel := ebiten.NewImageFromImage(sprite[4])
		priceText := fmt.Sprintf("%d", card.Price)
		priceFontFace := &text.GoTextFace{
			Source: fonts.MtgFont,
			Size:   16,
		}
		textX, textY := elements.AlignText(priceLabel, priceText, priceFontFace, elements.AlignCenter, elements.AlignMiddle)
		priceOptions := &ebiten.DrawImageOptions{}
		priceOptions.GeoM.Translate(textX, textY)
		text.Draw(priceLabel, priceText, priceFontFace, &text.DrawOptions{DrawImageOptions: *priceOptions})

		// if we want to show 5 cards, we need to shrink the image even more
		// (/ 668 5) = 133px wide, assuming game frame is 100px
		cards = append(cards, &elements.Button{
			Normal:   cardUpperImg,
			Hover:    cardUpperImg,
			Pressed:  cardUpperImg,
			State:    elements.StateNormal,
			X:        120 + (i * 160),
			Y:        200,
			Scale:    0.45,
			ButtonID: fmt.Sprintf("card_%d", cardIdx),
		})

		// Add price label button below the card
		cards = append(cards, &elements.Button{
			Normal:   priceLabel,
			Hover:    priceLabel,
			Pressed:  priceLabel,
			State:    elements.StateNormal,
			X:        120 + (i * 160), // Offset to center under card
			Y:        320,             // Below the card
			ButtonID: fmt.Sprintf("price_%d", cardIdx),
		})
	}

	buttons := []*elements.Button{
		&elements.Button{
			Normal:  sprite[0],
			Hover:   sprite[1],
			Pressed: sprite[2],
			ButtonText: elements.ButtonText{
				Text:             "Done",
				Font:             fontFace,
				TextColor:        color.White,
				HorizontalCenter: elements.AlignCenter,
				VerticalCenter:   elements.AlignMiddle,
			},
			State:    elements.StateNormal,
			X:        430,
			Y:        420,
			Scale:    SCALE,
			ButtonID: "done",
		},
	}
	buttons = append(buttons, cards...)

	fmt.Println("Buttons:", len(buttons))

	return buttons
}

func loadButtonMap(spriteFile []byte, mapFile []byte) []*ebiten.Image {
	sprInfo, err := sprites.LoadSprInfoFromJSON(mapFile)
	if err != nil {
		panic(err)
	}

	frameSprite, err := sprites.LoadSubimages(spriteFile, &sprInfo)
	if err != nil {
		panic(err)
	}

	return frameSprite
}

func readFromEmbeddedZip(zipData []byte, filename string) ([]byte, error) {
	// Create a reader from the embedded zip data
	r, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("failed to open embedded zip file: %w", err)
	}

	// Find the file in the zip
	for _, f := range r.File {
		if f.Name == filename {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to open file in zip: %w", err)
			}
			defer rc.Close()

			// Read the file contents
			data, err := io.ReadAll(rc)
			if err != nil {
				return nil, fmt.Errorf("failed to read file contents: %w", err)
			}
			return data, nil
		}
	}

	return nil, fmt.Errorf("file %s not found in zip", filename)
}
