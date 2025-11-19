package screens

import (
	"fmt"
	"image/color"

	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/ui/elements"
	"github.com/benprew/s30/game/ui/fonts"
	"github.com/benprew/s30/game/ui/layout"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// Adding cards to players collection is handled in DuelAnte.
// This screen is only responsible for showing the cards to the player.

type DuelLoseScreen struct {
	cards      []*domain.Card
	textbox    *elements.Button
	Background *ebiten.Image
}

func (s *DuelLoseScreen) IsFramed() bool { return false }

func NewDuelLoseScreen(cards []*domain.Card) *DuelLoseScreen {
	fontFace := &text.GoTextFace{
		Source: fonts.MtgFont,
		Size:   40,
	}

	textContent := "Lost these cards!"
	if len(cards) == 1 {
		textContent = "Lost this card!"
	}
	textWidth, textHeight := text.Measure(textContent, fontFace, 0)

	paddingX := 180.0
	paddingY := 30.0
	requiredWidth := textWidth + paddingX
	requiredHeight := textHeight + paddingY

	textBg, _ := elements.LoadImage(assets.DuelWinTextBox_png)
	bgBounds := textBg.Bounds()
	fmt.Println("BgBounds ", bgBounds.Dx(), bgBounds.Dy())
	fmt.Println("TextBounds ", requiredWidth, requiredHeight)
	scaleX := requiredWidth / float64(bgBounds.Dx())
	scaleY := requiredHeight / float64(bgBounds.Dy())
	scaledBg := elements.ScaleImageInd(textBg, scaleX, scaleY)

	tb := elements.NewButton(scaledBg, scaledBg, scaledBg, 0, 0, 1.0)
	tb.ButtonText = elements.ButtonText{
		Text:      textContent,
		Font:      fontFace,
		TextColor: color.White,
		HAlign:    elements.AlignCenter,
		VAlign:    elements.AlignMiddle,
	}
	tb.Position = &layout.Position{Anchor: layout.BottomLeft, OffsetX: 40, OffsetY: -200}

	bgImg, _ := elements.LoadImage(assets.DuelLoseBg_png)
	bgImg = elements.ScaleImage(bgImg, 1.6)

	return &DuelLoseScreen{
		cards:      cards,
		Background: bgImg,
		textbox:    tb,
	}
}

func (s *DuelLoseScreen) Draw(screen *ebiten.Image, W, H int, scale float64) {
	screen.DrawImage(s.Background, &ebiten.DrawImageOptions{})

	s.textbox.Draw(screen, &ebiten.DrawImageOptions{}, scale)

	cardOpts := &ebiten.DrawImageOptions{}
	cardOpts.GeoM.Translate(20, 20)
	for _, c := range s.cards {
		img, err := c.CardImage()
		if err != nil {
			fmt.Sprintf("ERR: couldn't load image for %s\n", c.Name())
			continue
		}
		img = elements.ScaleImage(img, 0.75)
		screen.DrawImage(img, cardOpts)
		cardOpts.GeoM.Translate(260, 0)
	}
}

func (s *DuelLoseScreen) Update(W, H int, scale float64) (screenui.ScreenName, error) {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return screenui.WorldScr, nil
	}
	return screenui.DuelLoseScr, nil
}
