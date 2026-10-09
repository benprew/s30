package duel

import (
	"fmt"
	"image"
	"image/color"

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
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// DuelWinScreen displays the cards, gold, and amulets awarded to the player
// after winning a duel. Any bonus cards (e.g. from castle defeat) are also shown.

const (
	winCardW        = domain.CardFullWidth
	winCardH        = 342
	winChoiceGap    = 30
	winChoiceY      = 120
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
	rewardPanel  *ebiten.Image
	rewardPanelX int
	rewardPanelY int
	doneBtn      *elements.Button
	btnSprites   [3]*ebiten.Image
	continueText string
	Background   *ebiten.Image
	ReturnScr    screenui.ScreenName
	ReturnScreen screenui.Screen
}

func winFont(size float64) *text.GoTextFace {
	return &text.GoTextFace{
		Source: fonts.MtgFont,
		Size:   size,
	}
}

func (s *DuelWinScreen) IsFramed() bool { return false }

func (s *DuelWinScreen) IsOverlay() bool { return false }

func NewWinDuelScreen(player *domain.Player, reward domain.DuelReward, bonusCards []*domain.Card) *DuelWinScreen {
	fontFace := winFont(40)

	textContent := "Rewards"

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

	rewardPanel, rewardPanelX, rewardPanelY := createRewardPanel(reward)

	btnSprites, err := imageutil.LoadSpriteSheet(3, 1, assets.Tradbut1_png)
	var sprites [3]*ebiten.Image
	if err == nil && len(btnSprites) > 0 && len(btnSprites[0]) >= 3 {
		sprites = [3]*ebiten.Image{btnSprites[0][0], btnSprites[0][1], btnSprites[0][2]}
	}

	btnFont := winFont(22)
	doneBtn := elements.NewButtonFromConfig(elements.ButtonConfig{
		Normal:  sprites[0],
		Hover:   sprites[1],
		Pressed: sprites[2],
		Text:    "Done",
		Font:    btnFont,
		ID:      "done",
	})

	bonusImgs := make([]*ebiten.Image, 0, len(bonusCards))
	for _, c := range bonusCards {
		img, err := c.CardImage(domain.CardViewFullMini)
		if err != nil {
			continue
		}
		bonusImgs = append(bonusImgs, img)
	}

	screen := &DuelWinScreen{
		player:       player,
		reward:       reward,
		cards:        cards,
		bonusImgs:    bonusImgs,
		Background:   bgImg,
		textbox:      tb,
		rewardPanel:  rewardPanel,
		rewardPanelX: rewardPanelX,
		rewardPanelY: rewardPanelY,
		doneBtn:      doneBtn,
		btnSprites:   sprites,
		ReturnScr:    screenui.WorldScr,
	}
	screen.updatePageLabels()
	return screen
}

// NewWinDuelScreenFromCards is a convenience constructor for creating a win screen
// with a list of cards and default zero gold/amulets.
func NewWinDuelScreenFromCards(player *domain.Player, cards []*domain.Card, bonusCards []*domain.Card) *DuelWinScreen {
	return NewWinDuelScreen(player, domain.DuelReward{Cards: cards}, bonusCards)
}

type rewardItem struct {
	sprite *ebiten.Image
	text   string
	color  color.Color
}

type measuredRewardItem struct {
	item    rewardItem
	width   float64
	textW   float64
	textH   float64
	spriteW float64
	spriteH float64
}

func createRewardPanel(reward domain.DuelReward) (*ebiten.Image, int, int) {
	var items []rewardItem

	if reward.Gold > 0 {
		items = append(items, rewardItem{
			text:  fmt.Sprintf("+%d Gold", reward.Gold),
			color: color.RGBA{R: 255, G: 215, B: 50, A: 255},
		})
	}

	if len(reward.Amulets) > 0 {
		amuletSprs, _ := imageutil.LoadSpriteSheet(5, 1, assets.Amsprite_png)
		amuletColors := []struct {
			mask  domain.ColorMask
			name  string
			index int
			col   color.Color
		}{
			{domain.ColorWhite, "White", 0, color.RGBA{R: 245, G: 245, B: 230, A: 255}},
			{domain.ColorBlue, "Blue", 1, color.RGBA{R: 120, G: 190, B: 255, A: 255}},
			{domain.ColorBlack, "Black", 2, color.RGBA{R: 210, G: 190, B: 230, A: 255}},
			{domain.ColorRed, "Red", 3, color.RGBA{R: 255, G: 130, B: 120, A: 255}},
			{domain.ColorGreen, "Green", 4, color.RGBA{R: 130, G: 240, B: 130, A: 255}},
		}

		counts := make(map[domain.ColorMask]int)
		for _, a := range reward.Amulets {
			counts[a.Color]++
		}

		for _, ac := range amuletColors {
			count := counts[ac.mask]
			if count <= 0 {
				continue
			}
			var label string
			if count == 1 {
				label = fmt.Sprintf("+1 %s Amulet", ac.name)
			} else {
				label = fmt.Sprintf("+%d %s Amulets", count, ac.name)
			}
			var spr *ebiten.Image
			if len(amuletSprs) > 0 && ac.index < len(amuletSprs[0]) {
				spr = imageutil.ScaleImage(amuletSprs[0][ac.index], 1.25)
			}
			items = append(items, rewardItem{
				sprite: spr,
				text:   label,
				color:  ac.col,
			})
		}
	}

	if len(items) == 0 {
		return nil, 0, 0
	}

	fontFace := winFont(20)
	itemGap := 28.0
	totalContentW := 0.0

	measured := make([]measuredRewardItem, len(items))
	for i, item := range items {
		tW, tH := text.Measure(item.text, fontFace, 0)
		sW, sH := 0.0, 0.0
		if item.sprite != nil {
			sW = float64(item.sprite.Bounds().Dx())
			sH = float64(item.sprite.Bounds().Dy())
		}
		w := tW
		if sW > 0 {
			w += sW + 8.0
		}
		measured[i] = measuredRewardItem{
			item:    item,
			width:   w,
			textW:   tW,
			textH:   tH,
			spriteW: sW,
			spriteH: sH,
		}
		totalContentW += w
		if i > 0 {
			totalContentW += itemGap
		}
	}

	panelW := max(340, int(totalContentW+56.0))
	panelH := 46
	panelX := (winLogicalW - panelW) / 2
	panelY := winChoiceY + winCardH + 18

	panelBg := ebiten.NewImage(panelW, panelH)
	panelBg.Fill(color.RGBA{R: 20, G: 18, B: 28, A: 230})
	vector.StrokeRect(panelBg, 1, 1, float32(panelW-2), float32(panelH-2), 1.5, color.RGBA{R: 190, G: 165, B: 95, A: 220}, false)
	vector.StrokeRect(panelBg, 4, 4, float32(panelW-8), float32(panelH-8), 1, color.RGBA{R: 95, G: 80, B: 50, A: 160}, false)

	startX := (float64(panelW) - totalContentW) / 2
	currX := startX

	for _, m := range measured {
		if m.spriteW > 0 {
			sprY := (float64(panelH) - m.spriteH) / 2
			sOpts := &ebiten.DrawImageOptions{}
			sOpts.GeoM.Translate(currX, sprY)
			panelBg.DrawImage(m.item.sprite, sOpts)
			currX += m.spriteW + 8.0
		}
		textY := (float64(panelH) - m.textH) / 2

		shOpts := &text.DrawOptions{}
		shOpts.GeoM.Translate(currX+1, textY+1)
		shOpts.ColorScale.ScaleWithColor(color.RGBA{R: 0, G: 0, B: 0, A: 200})
		text.Draw(panelBg, m.item.text, fontFace, shOpts)

		tOpts := &text.DrawOptions{}
		tOpts.GeoM.Translate(currX, textY)
		tOpts.ColorScale.ScaleWithColor(m.item.color)
		text.Draw(panelBg, m.item.text, fontFace, tOpts)

		currX += m.textW + itemGap
	}

	return panelBg, panelX, panelY
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

// SetContinueText sets the button label for the final reward page.
func (s *DuelWinScreen) SetContinueText(label string) {
	s.continueText = label
	s.updatePageLabels()
}

func (s *DuelWinScreen) updatePageLabels() {
	if s.doneBtn == nil {
		return
	}
	label := "Done"
	if s.continueText != "" {
		label = s.continueText
	}
	if s.page+1 < s.pageCount() {
		label = "Next"
	}
	s.doneBtn.ButtonText.Text = label
	if s.textbox != nil {
		s.textbox.ButtonText.Text = "Rewards"
		if s.page >= s.cardPageCount() && len(s.bonusImgs) > 0 {
			s.textbox.ButtonText.Text = "Bonus Cards"
		}
	}
	s.updateDoneButton()
}

func (s *DuelWinScreen) updateDoneButton() {
	if s.doneBtn == nil || s.btnSprites[0] == nil {
		return
	}
	fontFace := winFont(22)
	textW, _ := text.Measure(s.doneBtn.ButtonText.Text, fontFace, 0)
	buttonW := max(260, int(textW+70))
	buttonH := 42

	scaleX := float64(buttonW) / float64(s.btnSprites[0].Bounds().Dx())
	scaleY := float64(buttonH) / float64(s.btnSprites[0].Bounds().Dy())

	s.doneBtn.Normal = imageutil.ScaleImageInd(s.btnSprites[0], scaleX, scaleY)
	s.doneBtn.Hover = imageutil.ScaleImageInd(s.btnSprites[1], scaleX, scaleY)
	s.doneBtn.Pressed = imageutil.ScaleImageInd(s.btnSprites[2], scaleX, scaleY)

	btnX := (winLogicalW - buttonW) / 2
	btnY := winChoiceY + winCardH + 40
	if s.rewardPanel != nil {
		btnY = s.rewardPanelY + s.rewardPanel.Bounds().Dy() + 32
	}
	s.doneBtn.Bounds = image.Rect(btnX, btnY, btnX+buttonW, btnY+buttonH)
	s.doneBtn.ButtonText.Font = fontFace
	s.doneBtn.ButtonText.TextColor = color.White
	s.doneBtn.ButtonText.HAlign = elements.AlignCenter
	s.doneBtn.ButtonText.VAlign = elements.AlignMiddle
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

	if s.textbox != nil {
		s.textbox.Draw(screen, &ebiten.DrawImageOptions{}, scale)
	}

	if s.rewardPanel != nil {
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(float64(s.rewardPanelX), float64(s.rewardPanelY))
		screen.DrawImage(s.rewardPanel, opts)
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

	if s.doneBtn != nil {
		s.doneBtn.Draw(screen, &ebiten.DrawImageOptions{}, scale)
	}

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
	if s.doneBtn != nil {
		s.doneBtn.Update(&ebiten.DrawImageOptions{}, scale, W, H)
	}
	if (s.doneBtn != nil && s.doneBtn.IsClicked()) ||
		inpututil.IsKeyJustPressed(ebiten.Key1) ||
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
