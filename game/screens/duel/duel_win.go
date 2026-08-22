package duel

import (
	"fmt"
	"image"
	"image/color"
	"strings"

	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/ui"
	"github.com/benprew/s30/game/ui/elements"
	"github.com/benprew/s30/game/ui/fonts"
	"github.com/benprew/s30/game/ui/imageutil"
	"github.com/benprew/s30/game/ui/layout"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// DuelWinScreen displays the cards, gold, and amulets awarded to the player
// after winning a duel. Any bonus cards (e.g. from castle defeat) are also shown.

const (
	winCardW      = domain.CardFullWidth
	winCardH      = 342
	winChoiceGap  = 30
	winChoiceY    = 130
	winBonusScale = 0.45
	winBonusGap   = 20
	winLogicalW   = 1024
	winLogicalH   = 768
)

type winCard struct {
	card  *domain.Card
	rect  image.Rectangle
	scale float64
}

type DuelWinScreen struct {
	player       *domain.Player
	reward       domain.DuelReward
	cards        []winCard
	bonusImgs    []*ebiten.Image
	textbox      *elements.Button
	rewardText   *elements.Text
	doneBtn      *elements.Button
	Background   *ebiten.Image
	ReturnScr    screenui.ScreenName
	ReturnScreen screenui.Screen
}

func (s *DuelWinScreen) IsFramed() bool { return false }

func (s *DuelWinScreen) IsOverlay() bool { return false }

func NewWinDuelScreen(player *domain.Player, reward domain.DuelReward, bonusCards []*domain.Card) *DuelWinScreen {
	fontFace := &text.GoTextFace{
		Source: fonts.MtgFont,
		Size:   40,
	}

	textContent := "Cards Won"

	textWidth, textHeight := text.Measure(textContent, fontFace, 0)

	paddingX := 180.0
	paddingY := 30.0
	requiredWidth := textWidth + paddingX
	requiredHeight := textHeight + paddingY

	textBg, _ := imageutil.LoadImage(assets.DuelWinTextBox_png)
	bgBounds := textBg.Bounds()
	scaleX := requiredWidth / float64(bgBounds.Dx())
	scaleY := requiredHeight / float64(bgBounds.Dy())
	scaledBg := imageutil.ScaleImageInd(textBg, scaleX, scaleY)

	tb := elements.NewButton(scaledBg, scaledBg, scaledBg, 0, 0, 1.0)
	tb.ButtonText = elements.ButtonText{
		Text:      textContent,
		Font:      fontFace,
		TextColor: color.White,
		HAlign:    elements.AlignCenter,
		VAlign:    elements.AlignMiddle,
	}
	tb.Position = &layout.Position{Anchor: layout.TopCenter, OffsetX: -int(requiredWidth / 2), OffsetY: 20}

	bgImg, _ := imageutil.LoadImage(assets.DuelWinBg_png)
	bgImg = imageutil.ScaleImage(bgImg, 1.6)

	cards := layoutCards(reward.Cards)

	doneBtn := elements.NewButtonFromConfig(elements.ButtonConfig{
		Normal: scaledBg,
		Text:   "Done",
		Font:   fontFace,
		ID:     "done",
	})
	doneW := doneBtn.Bounds.Dx()
	doneBtn.MoveTo((winLogicalW-doneW)/2, winChoiceY+winCardH+20)

	var rewardParts []string
	if reward.Gold > 0 {
		rewardParts = append(rewardParts, fmt.Sprintf("+%d Gold", reward.Gold))
	}
	if len(reward.Amulets) > 0 {
		counts := make(map[string]int)
		for _, a := range reward.Amulets {
			colorStr := domain.ColorMaskToString(a.Color)
			counts[colorStr]++
		}
		for colorStr, count := range counts {
			if count == 1 {
				rewardParts = append(rewardParts, fmt.Sprintf("+1 %s Amulet", colorStr))
			} else {
				rewardParts = append(rewardParts, fmt.Sprintf("+%d %s Amulets", count, colorStr))
			}
		}
	}

	var rewardLabel *elements.Text
	if len(rewardParts) > 0 {
		rewardStr := strings.Join(rewardParts, "   ")
		rewardLabel = elements.NewText(24, rewardStr, 0, winChoiceY-35)
		rewardLabel.Color = color.RGBA{R: 255, G: 230, B: 150, A: 255}
		rewardLabel.HAlign = elements.AlignCenter
		rewardLabel.BoundsW = winLogicalW
	}

	bonusImgs := make([]*ebiten.Image, 0, len(bonusCards))
	for _, c := range bonusCards {
		img, err := c.CardImage(domain.CardViewFull)
		if err != nil {
			continue
		}
		bonusImgs = append(bonusImgs, imageutil.ScaleImage(img, winBonusScale))
	}

	return &DuelWinScreen{
		player:       player,
		reward:       reward,
		cards:        cards,
		bonusImgs:    bonusImgs,
		Background:   bgImg,
		textbox:      tb,
		rewardText:   rewardLabel,
		doneBtn:      doneBtn,
		ReturnScr:    screenui.WorldScr,
	}
}

// NewWinDuelScreenFromCards is a convenience constructor for creating a win screen
// with a list of cards and default zero gold/amulets.
func NewWinDuelScreenFromCards(player *domain.Player, cards []*domain.Card, bonusCards []*domain.Card) *DuelWinScreen {
	return NewWinDuelScreen(player, domain.DuelReward{Cards: cards}, bonusCards)
}

// layoutCards positions the reward cards in a centered horizontal row and
// scales them dynamically to fit within the logical screen width.
func layoutCards(cards []*domain.Card) []winCard {
	n := len(cards)
	if n == 0 {
		return nil
	}

	availW := float64(winLogicalW - 80)
	cardScale := 1.0
	gap := float64(winChoiceGap)

	rawTotalW := float64(n)*float64(winCardW) + float64(n-1)*gap
	if rawTotalW > availW {
		cardScale = availW / (float64(n)*float64(winCardW) + float64(n-1)*gap)
		if cardScale > 1.0 {
			cardScale = 1.0
		}
		gap = gap * cardScale
	}

	scaledCardW := float64(winCardW) * cardScale
	scaledCardH := float64(winCardH) * cardScale
	totalW := float64(n)*scaledCardW + float64(n-1)*gap
	startX := (float64(winLogicalW) - totalW) / 2.0
	y := float64(winChoiceY)
	if cardScale < 1.0 {
		y += (float64(winCardH) - scaledCardH) / 4.0
	}

	res := make([]winCard, 0, n)
	for i, c := range cards {
		x := startX + float64(i)*(scaledCardW+gap)
		res = append(res, winCard{
			card:  c,
			scale: cardScale,
			rect:  image.Rect(int(x), int(y), int(x+scaledCardW), int(y+scaledCardH)),
		})
	}
	return res
}

func (s *DuelWinScreen) Draw(screen *ebiten.Image, W, H int, scale float64) {
	screen.DrawImage(s.Background, &ebiten.DrawImageOptions{})

	s.textbox.Draw(screen, &ebiten.DrawImageOptions{}, scale)

	if s.rewardText != nil {
		s.rewardText.Draw(screen, &ebiten.DrawImageOptions{}, scale)
	}

	mp := ui.Position()

	for _, c := range s.cards {
		img, err := c.card.CardImage(domain.CardViewFull)
		if err != nil {
			continue
		}
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Scale(c.scale, c.scale)
		if mp.In(c.rect) {
			opts.ColorScale.Scale(1.15, 1.15, 1.15, 1.0)
		}
		opts.GeoM.Translate(float64(c.rect.Min.X), float64(c.rect.Min.Y))
		screen.DrawImage(img, opts)
	}

	s.doneBtn.Draw(screen, &ebiten.DrawImageOptions{}, scale)

	s.drawBonus(screen)
}

func (s *DuelWinScreen) drawBonus(screen *ebiten.Image) {
	if len(s.bonusImgs) == 0 {
		return
	}

	bonusW := s.bonusImgs[0].Bounds().Dx()
	bonusH := s.bonusImgs[0].Bounds().Dy()
	n := len(s.bonusImgs)
	totalW := n*bonusW + (n-1)*winBonusGap
	startX := (winLogicalW - totalW) / 2
	y := winLogicalH - bonusH - 30

	label := elements.NewText(24, "Bonus", winLogicalW/2-40, y-30)
	label.Color = color.White
	label.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)

	for i, img := range s.bonusImgs {
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(float64(startX+i*(bonusW+winBonusGap)), float64(y))
		screen.DrawImage(img, opts)
	}
}

func (s *DuelWinScreen) Update(W, H int, scale float64) (screenui.ScreenName, screenui.Screen, error) {
	s.doneBtn.Update(&ebiten.DrawImageOptions{}, scale, W, H)
	if s.doneBtn.IsClicked() ||
		inpututil.IsKeyJustPressed(ebiten.KeyEscape) ||
		inpututil.IsKeyJustPressed(ebiten.KeySpace) ||
		inpututil.IsKeyJustPressed(ebiten.KeyEnter) ||
		ui.Click(image.Rect(0, 0, W, H)) {
		return s.ReturnScr, s.ReturnScreen, nil
	}

	return screenui.DuelWinScr, nil, nil
}

