package screens

import (
	"bytes"
	"fmt"
	"image"
	"image/color"

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

func NewBuyCardsScreen(city *domain.City, player *domain.Player, W, H int) *BuyCardsScreen {
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
	titleText.Draw(title, &ebiten.DrawImageOptions{}, 1.0)

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
	screen.Buttons = mkCardButtons(SCALE, city, screen.CardImgs, W, H)
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
		promptText.Draw(screen, &ebiten.DrawImageOptions{}, scale)
		// Draw error message if present using Text element
		if s.ErrorMsg != "" {
			errX := int(centerX + (fw / 2) - (len(s.ErrorMsg) * 7))
			errY := int(centerY + fh - 10)
			errText := elements.NewText(24, s.ErrorMsg, errX, errY)
			errText.Draw(screen, &ebiten.DrawImageOptions{}, scale)
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
		b.Update(options, scale, W, H)
		if b.ID == "done" && b.IsClicked() {
			return screenui.CityScr, nil
		}
		// Detect card or price click
		if b.IsClicked() {
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
	card := s.City.CardsForSale[s.PreviewIdx]
	if s.Player.Gold >= card.Price {
		fmt.Println("Buying card:", s.PreviewIdx, "name:", card.Name(), "for", card.Price, "gold")
		s.Player.Gold -= card.Price
		if s.Player.CardCollection == nil {
			s.Player.CardCollection = make(map[*domain.Card]int)
		}
		s.Player.CardCollection[card]++
		s.City.CardsForSale[s.PreviewIdx] = nil
		s.CardImgs[s.PreviewIdx] = nil
		s.Buttons = mkCardButtons(SCALE, s.City, s.CardImgs, 1024, 768) // TODO remove hardcoded W/H
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

func mkCardButtons(scale float64, city *domain.City, cardImgs []*ebiten.Image, W, H int) []*elements.Button {
	sprite := loadButtonMap(assets.BuyCardsSprite_png, assets.BuyCardsSpriteMap_json)
	fontFace := &text.GoTextFace{
		Source: fonts.MtgFont,
		Size:   32,
	}

	// make card buttons
	cards := make([]*elements.Button, 0)
	for i, card := range city.CardsForSale {
		if card == nil {
			continue
		}
		cardImg := cardImgs[i]
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

		x := 120 + (i * 160)
		cardBtn := elements.NewButton(cardUpperImg, cardUpperImg, cardUpperImg, x, 200, 0.45)
		cardBtn.ID = fmt.Sprintf("card_%d", i)
		cards = append(cards, cardBtn)

		priceBtn := elements.NewButton(priceLabel, priceLabel, priceLabel, x, 320, 1.0)
		priceBtn.ID = fmt.Sprintf("price_%d", i)
		cards = append(cards, priceBtn)
	}

	btnWidth := float64(sprite[0].Bounds().Dx())
	x := int((float64(W) - (btnWidth * SCALE)) / 2.0)

	doneBtn := elements.NewButton(sprite[0], sprite[1], sprite[2], x, 420, SCALE)
	doneBtn.ButtonText = elements.ButtonText{
		Text:      "Done",
		Font:      fontFace,
		TextColor: color.White,
		HAlign:    elements.AlignCenter,
		VAlign:    elements.AlignMiddle,
	}
	doneBtn.ID = "done"

	buttons := []*elements.Button{doneBtn}
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
func loadCardImages(cards []*domain.Card) (images []*ebiten.Image) {
	for _, card := range cards {
		cardPng, err := card.CardImage()
		if err == nil {
			images = append(images, cardPng)
		} else {
			panic(fmt.Sprintf("unable to load card image: %s: %v", card.Filename(), err))
		}
	}
	return images
}
