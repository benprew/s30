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
	Player      *domain.Player
	BgImage     *ebiten.Image
	ScreenTitle *ebiten.Image
	CardFrame   *ebiten.Image
	PreviewIdx  int             // -1 if not previewing, else index into CardsForSale
	PreviewType string          // "" (none), "card", or "price"
	ErrorMsg    string          // error message to display (e.g. not enough money)
	CardImgs    []*ebiten.Image // images for the cards, indexed by CardKeys
}

func (s *BuyCardsScreen) IsFramed() bool {
	return true
}

func NewBuyCardsScreen(city *domain.City, player *domain.Player) *BuyCardsScreen {
	drawOpts := &ebiten.DrawImageOptions{}
	drawOpts.GeoM.Scale(SCALE, SCALE)
	img, _, err := image.Decode(bytes.NewReader(assets.BuyCards_png))
	if err != nil {
		panic(fmt.Sprintf("Unable to load BuyCards.png: %s", err))
	}

	city.CardsForSale = domain.MkCards()

	sprite := loadButtonMap(assets.BuyCardsSprite_png, assets.BuyCardsSpriteMap_json)
	// fontFace no longer needed, handled by Text element
	title := ebiten.NewImageFromImage(sprite[4])
	txt := "Cards for Sale"
	// Use Text element to draw the title onto the image
	titleText := elements.NewText(16, txt, 0, 0)
	titleText.Draw(title, &ebiten.DrawImageOptions{})

	// Use LoadSpriteSheet to extract the card frame from Buybuttons.spr.png (6x2 grid, frame at 0,0)
	frames, err := sprites.LoadSpriteSheet(6, 2, assets.BuyButtons_png)
	if err != nil {
		panic(fmt.Sprintf("Unable to load Buybuttons.spr.png: %s", err))
	}
	frameImg := frames[0][0]
	screen := &BuyCardsScreen{
		City:        city,
		Player:      player,
		BgImage:     ebiten.NewImageFromImage(img),
		ScreenTitle: title,
		CardFrame:   frameImg,
		PreviewIdx:  -1,
		PreviewType: "",
		ErrorMsg:    "",
		CardImgs:    loadCardImages(city.CardsForSale),
	}
	screen.Buttons = mkCardButtons(SCALE, city, screen.CardImgs)
	return screen
}

func (s *BuyCardsScreen) Draw(screen *ebiten.Image, W, H int, scale float64) {
	cityOpts := &ebiten.DrawImageOptions{}
	cityOpts.GeoM.Scale(SCALE, SCALE)
	cityOpts.GeoM.Translate(100.0, 75.0) // Offset the image
	screen.DrawImage(s.BgImage, cityOpts)

	titleOpts := &ebiten.DrawImageOptions{}
	titleOpts.GeoM.Scale(SCALE, SCALE)
	// Center the ScreenTitle horizontally
	titleWidth := float64(s.ScreenTitle.Bounds().Dx()) * SCALE
	titleX := (float64(W) - titleWidth) / 2.0
	titleOpts.GeoM.Translate(titleX, 100.0)
	screen.DrawImage(s.ScreenTitle, titleOpts)

	frameOpts := &ebiten.DrawImageOptions{}
	frameOpts.GeoM.Scale(scale, scale)
	for _, b := range s.Buttons {
		// Center the 'done' button horizontally
		if b.ID == "done" {
			btnWidth := float64(b.Normal.Bounds().Dx()) * b.Scale
			b.X = int((float64(W) - btnWidth) / 2.0)
		}
		b.Draw(screen, frameOpts, scale)
	}

	// Draw card preview if active
	if s.PreviewIdx >= 0 && s.PreviewIdx < len(s.CardImgs) {
		cardImg := s.CardImgs[s.PreviewIdx]
		fw, fh := s.CardFrame.Bounds().Dx(), s.CardFrame.Bounds().Dy()
		cw, ch := cardImg.Bounds().Dx(), cardImg.Bounds().Dy()
		centerX := (W - fw) / 2
		centerY := (H - fh) / 2
		frameOpts := &ebiten.DrawImageOptions{}
		frameOpts.GeoM.Translate(float64(centerX), float64(centerY))
		screen.DrawImage(s.CardFrame, frameOpts)
		// Draw card image inside frame (centered)
		cardOpts := &ebiten.DrawImageOptions{}
		cardOpts.GeoM.Translate(float64(centerX+(fw-cw)/2), float64(centerY+(fh-ch)/2))
		screen.DrawImage(cardImg, cardOpts)
		// Draw prompt using Text element
		prompt := "Buy Card Y/N?"
		promptX := int(centerX + (fw / 2) - (len(prompt) * 8))
		promptY := int(centerY + fh - 40)
		promptText := elements.NewText(32, prompt, promptX, promptY)
		promptText.Draw(screen, &ebiten.DrawImageOptions{})
		// Draw error message if present using Text element
		if s.ErrorMsg != "" {
			errX := int(centerX + (fw / 2) - (len(s.ErrorMsg) * 7))
			errY := int(centerY + fh - 10)
			errText := elements.NewText(24, s.ErrorMsg, errX, errY)
			errText.Draw(screen, &ebiten.DrawImageOptions{})
		}
	}
}

func (s *BuyCardsScreen) Update(W, H int, scale float64) (screenui.ScreenName, error) {
	options := &ebiten.DrawImageOptions{}

	// If previewing, handle Y/N
	if s.PreviewIdx >= 0 {
		if inpututil.IsKeyJustPressed(ebiten.KeyY) && s.Player != nil && s.PreviewIdx < len(s.CardImgs) {
			s.buyCard()
			return screenui.BuyCardsScr, nil
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyN) || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			s.ErrorMsg = ""
			s.PreviewIdx = -1
			s.PreviewType = ""
			return screenui.BuyCardsScr, nil
		}
		return screenui.BuyCardsScr, nil
	}

	for i := range s.Buttons {
		b := s.Buttons[i]
		b.Update(options, scale)
		if b.ID == "done" && b.State == elements.StateClicked {
			return screenui.CityScr, nil
		}
		// Detect card or price click
		if b.State == elements.StateClicked {
			if len(b.ID) > 5 && b.ID[:5] == "card_" {
				idx := -1
				fmt.Sscanf(b.ID, "card_%d", &idx)
				s.PreviewIdx = idxInCardsForSale(s.City, idx)
				s.PreviewType = "card"
				return screenui.BuyCardsScr, nil
			}
			if len(b.ID) > 6 && b.ID[:6] == "price_" {
				idx := -1
				fmt.Sscanf(b.ID, "price_%d", &idx)
				s.PreviewIdx = idxInCardsForSale(s.City, idx)
				s.PreviewType = "price"
				return screenui.BuyCardsScr, nil
			}
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return screenui.CityScr, nil
	}
	return screenui.BuyCardsScr, nil
}

func (s *BuyCardsScreen) buyCard() {
	s.ErrorMsg = ""
	cardIdx := s.City.CardsForSale[s.PreviewIdx]
	card := domain.CARDS[cardIdx]
	if s.Player.Gold >= card.Price {
		fmt.Println("Buying card:", s.PreviewIdx, "name:", card.Name(), "for", card.Price, "gold")
		s.Player.Gold -= card.Price
		if s.Player.CardCollection == nil {
			s.Player.CardCollection = make(map[int]int)
		}
		s.Player.CardCollection[cardIdx]++
		s.City.CardsForSale[s.PreviewIdx] = -1
		s.CardImgs[s.PreviewIdx] = nil
		s.Buttons = mkCardButtons(SCALE, s.City, s.CardImgs)
		s.PreviewIdx = -1
		s.PreviewType = ""
	} else {
		s.ErrorMsg = "Not enough money!"
	}
}

// Helper to find index in CardsForSale slice
func idxInCardsForSale(city *domain.City, cardIdx int) int {
	i := 0
	for idx := range city.CardsForSale {
		if idx == cardIdx {
			return i
		}
		i++
	}
	return -1
}

func mkCardButtons(scale float64, city *domain.City, cardImgs []*ebiten.Image) []*elements.Button {
	sprite := loadButtonMap(assets.BuyCardsSprite_png, assets.BuyCardsSpriteMap_json)
	fontFace := &text.GoTextFace{
		Source: fonts.MtgFont,
		Size:   32,
	}

	// make card buttons
	cards := make([]*elements.Button, 0)
	for i, cardIdx := range city.CardsForSale {
		if cardIdx == -1 {
			continue
		}
		cardImg := cardImgs[i]
		card := domain.CARDS[cardIdx]
		cardUpperImg := ebiten.NewImage(300, 220)
		cardUpperImg.DrawImage(cardImg, &ebiten.DrawImageOptions{})

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
			Normal:  cardUpperImg,
			Hover:   cardUpperImg,
			Pressed: cardUpperImg,
			X:       120 + (i * 160),
			Y:       200,
			Scale:   0.45,
			ID:      fmt.Sprintf("card_%d", i),
		})

		// Add price label button below the card
		cards = append(cards, &elements.Button{
			Normal:  priceLabel,
			Hover:   priceLabel,
			Pressed: priceLabel,
			X:       120 + (i * 160), // Offset to center under card
			Y:       320,             // Below the card
			ID:      fmt.Sprintf("price_%d", i),
		})
	}

	buttons := []*elements.Button{
		&elements.Button{
			Normal:  sprite[0],
			Hover:   sprite[1],
			Pressed: sprite[2],
			ButtonText: elements.ButtonText{
				Text:      "Done",
				Font:      fontFace,
				TextColor: color.White,
				HAlign:    elements.AlignCenter,
				VAlign:    elements.AlignMiddle,
			},
			State: elements.StateNormal,
			X:     0, // will be centered in Draw
			Y:     420,
			Scale: SCALE,
			ID:    "done",
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

// Load image for this card
func loadCardImages(cards []int) (images []*ebiten.Image) {
	for _, idx := range cards {
		card := domain.CARDS[idx]
		filename := card.Filename()
		data, err := readFromEmbeddedZip(assets.CardImages_zip, "carddata/"+filename)
		if err == nil {
			cardPng, _, err := image.Decode(bytes.NewReader(data))
			if err == nil {
				images = append(images, ebiten.NewImageFromImage(cardPng))
			}
		}
	}

	return images
}

// Helper to load card images from embedded zip (copied from screens)
func readFromEmbeddedZip(zipData []byte, filename string) ([]byte, error) {
	r, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, err
	}
	for _, f := range r.File {
		if f.Name == filename {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			data, err := io.ReadAll(rc)
			if err != nil {
				return nil, err
			}
			return data, nil
		}
	}
	return nil, io.EOF
}
