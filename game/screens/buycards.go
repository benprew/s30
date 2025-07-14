package screens

import (
	"bytes"
	"fmt"
	"image"
	"image/color"

	"github.com/benprew/s30/assets/art"
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
	Frame   *ebiten.Image
	Buttons []*elements.Button
	City    *domain.City
	BgImage *ebiten.Image
}

func NewBuyCardsScreen(frame *ebiten.Image, city *domain.City) *BuyCardsScreen {
	img, _, err := image.Decode(bytes.NewReader(art.BuyCards_png))
	if err != nil {
		panic(fmt.Sprintf("Unable to load BuyCards.png: %s", err))
	}

	return &BuyCardsScreen{
		Frame:   frame,
		Buttons: mkCardButtons(SCALE-0.4, city),
		City:    city,
		BgImage: ebiten.NewImageFromImage(img),
	}
}

func (s *BuyCardsScreen) Draw(screen *ebiten.Image, W, H int, scale float64) {
	cityOpts := &ebiten.DrawImageOptions{}
	cityOpts.GeoM.Scale(SCALE, SCALE)
	cityOpts.GeoM.Translate(100.0, 75.0) // Offset the image
	screen.DrawImage(s.BgImage, cityOpts)

	frameOpts := &ebiten.DrawImageOptions{}
	frameOpts.GeoM.Scale(scale, scale)
	screen.DrawImage(s.Frame, frameOpts)

	for _, b := range s.Buttons {
		b.Draw(screen, frameOpts)
	}
}

func (s *BuyCardsScreen) Update(W, H int) (screenui.ScreenName, error) {
	options := &ebiten.DrawImageOptions{}
	for i := range s.Buttons {
		b := s.Buttons[i]
		b.Update(options)
		if b.Text == "Done" && b.State == elements.StateClicked {
			return screenui.CityScr, nil
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return screenui.CityScr, nil
	}
	return screenui.BuyCardsScr, nil
}

func mkCardButtons(scale float64, city *domain.City) []*elements.Button {
	sprite := loadButtonMap(art.BuyCardsSprite_png, art.BuyCardsSpriteMap_json)
	fontFace := &text.GoTextFace{
		Source: fonts.MtgFont,
		Size:   20,
	}

	// buttons := make([]*elements.Button, len(buttonConfigs))
	// for i, config := range buttonConfigs {
	//  btn := mkButton(config, fontFace, Icons, Iconb, scale)
	//  buttons[i] = &btn
	// }

	buttons := []*elements.Button{
		&elements.Button{
			Normal:     sprite[0],
			Hover:      sprite[1],
			Pressed:    sprite[2],
			Text:       "Done",
			Font:       fontFace,
			TextColor:  color.White,
			TextOffset: image.Point{X: 25, Y: 10},
			State:      elements.StateNormal,
			X:          480,
			Y:          420,
		},
	}

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
