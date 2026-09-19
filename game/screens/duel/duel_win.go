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
	winCardW        = domain.CardFullWidth
	winCardH        = 342
	winChoiceGap    = 30
	winChoiceY      = 130
	winBonusGap     = 20
	winCardsPerPage = 3
	winBonusPerPage = 5
	winLogicalW     = 1024
	winLogicalH     = 768
)

type winCard struct {
	card *domain.Card
	rect image.Rectangle
}

type DuelWinScreen struct {
	player       *domain.Player
	reward       domain.DuelReward
	cards        []winCard
	bonusImgs    []*ebiten.Image
	page         int
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
		img, err := c.CardImage(domain.CardViewFullMini)
		if err != nil {
			continue
		}
		bonusImgs = append(bonusImgs, img)
	}

	screen := &DuelWinScreen{
		player:     player,
		reward:     reward,
		cards:      cards,
		bonusImgs:  bonusImgs,
		Background: bgImg,
		textbox:    tb,
		rewardText: rewardLabel,
		doneBtn:    doneBtn,
		ReturnScr:  screenui.WorldScr,
	}
	screen.updatePageLabels()
	return screen
}

// NewWinDuelScreenFromCards is a convenience constructor for creating a win screen
// with a list of cards and default zero gold/amulets.
func NewWinDuelScreenFromCards(player *domain.Player, cards []*domain.Card, bonusCards []*domain.Card) *DuelWinScreen {
	return NewWinDuelScreen(player, domain.DuelReward{Cards: cards}, bonusCards)
}

// layoutCards keeps the full card size on each reward page.
func layoutCards(cards []*domain.Card) []winCard {
	result := make([]winCard, 0, len(cards))
	for i, card := range cards {
		pageStart := i / winCardsPerPage * winCardsPerPage
		count := min(winCardsPerPage, len(cards)-pageStart)
		totalW := count*winCardW + (count-1)*winChoiceGap
		x := (winLogicalW-totalW)/2 + (i%winCardsPerPage)*(winCardW+winChoiceGap)
		result = append(result, winCard{card: card, rect: image.Rect(x, winChoiceY, x+winCardW, winChoiceY+winCardH)})
	}
	return result
}

func (s *DuelWinScreen) cardPageCount() int {
	return (len(s.cards) + winCardsPerPage - 1) / winCardsPerPage
}

func (s *DuelWinScreen) pageCount() int {
	return max(1, s.cardPageCount()+(len(s.bonusImgs)+winBonusPerPage-1)/winBonusPerPage)
}

func (s *DuelWinScreen) updatePageLabels() {
	s.doneBtn.ButtonText.Text = "Done"
	if s.page+1 < s.pageCount() {
		s.doneBtn.ButtonText.Text = "Next"
	}
	s.textbox.ButtonText.Text = "Cards Won"
	if s.page >= s.cardPageCount() && len(s.bonusImgs) > 0 {
		s.textbox.ButtonText.Text = "Bonus Cards"
	}
}

func (s *DuelWinScreen) nextPage() bool {
	if s.page+1 >= s.pageCount() {
		return false
	}
	s.page++
	s.updatePageLabels()
	return true
}

func (s *DuelWinScreen) Draw(screen *ebiten.Image, W, H int, scale float64) {
	screen.DrawImage(s.Background, &ebiten.DrawImageOptions{})

	s.textbox.Draw(screen, &ebiten.DrawImageOptions{}, scale)

	if s.rewardText != nil {
		s.rewardText.Draw(screen, &ebiten.DrawImageOptions{}, scale)
	}

	mp := ui.Position()

	start := min(s.page*winCardsPerPage, len(s.cards))
	for _, c := range s.cards[start:min(start+winCardsPerPage, len(s.cards))] {
		img, err := c.card.CardImage(domain.CardViewFull)
		if err != nil {
			continue
		}
		opts := &ebiten.DrawImageOptions{}
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
	page := s.page - s.cardPageCount()
	if page < 0 || len(s.bonusImgs) == 0 {
		return
	}
	start := page * winBonusPerPage
	images := s.bonusImgs[start:min(start+winBonusPerPage, len(s.bonusImgs))]
	totalW := len(images)*domain.CardFullMiniWidth + (len(images)-1)*winBonusGap
	x := (winLogicalW - totalW) / 2
	for _, img := range images {
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(float64(x), winChoiceY)
		screen.DrawImage(img, opts)
		x += domain.CardFullMiniWidth + winBonusGap
	}
}

func (s *DuelWinScreen) Update(W, H int, scale float64) (screenui.ScreenName, screenui.Screen, error) {
	s.doneBtn.Update(&ebiten.DrawImageOptions{}, scale, W, H)
	if s.doneBtn.IsClicked() ||
		inpututil.IsKeyJustPressed(ebiten.KeyEscape) ||
		inpututil.IsKeyJustPressed(ebiten.KeySpace) ||
		inpututil.IsKeyJustPressed(ebiten.KeyEnter) ||
		ui.Click(image.Rect(0, 0, W, H)) {
		if s.nextPage() {
			return screenui.DuelWinScr, nil, nil
		}
		return s.ReturnScr, s.ReturnScreen, nil
	}

	return screenui.DuelWinScr, nil, nil
}
