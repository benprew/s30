package duel

import (
	"image"
	"image/color"

	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/domain"
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

// DuelWinScreen lets the player pick one reward card from a small set of
// choices after winning a duel. The chosen card is added to the player's
// collection here. Any bonus cards (e.g. a castle defeat) are awarded
// automatically before this screen and shown for reference only.

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

type winChoice struct {
	card *domain.Card
	rect image.Rectangle
}

type DuelWinScreen struct {
	player     *domain.Player
	choices    []winChoice
	selected   int
	bonusImgs  []*ebiten.Image
	textbox    *elements.Button
	doneBtn    *elements.Button
	Background *ebiten.Image
	// ReturnScr is the screen to return to once the player dismisses the win
	// screen. Defaults to the overworld; dungeon duels override it so the player
	// resumes exploring the dungeon.
	ReturnScr screenui.ScreenName
}

func (s *DuelWinScreen) IsFramed() bool { return false }

func (s *DuelWinScreen) IsOverlay() bool { return false }

func NewWinDuelScreen(player *domain.Player, choices []*domain.Card, bonusCards []*domain.Card) *DuelWinScreen {
	fontFace := &text.GoTextFace{
		Source: fonts.MtgFont,
		Size:   40,
	}

	textContent := "Choose a card"

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

	winChoices := layoutChoices(choices)

	doneBtn := elements.NewButtonFromConfig(elements.ButtonConfig{
		Normal: scaledBg,
		Text:   "Done",
		Font:   fontFace,
		ID:     "done",
	})
	doneW := doneBtn.Bounds.Dx()
	doneBtn.MoveTo((winLogicalW-doneW)/2, winChoiceY+winCardH+20)

	bonusImgs := make([]*ebiten.Image, 0, len(bonusCards))
	for _, c := range bonusCards {
		img, err := c.CardImage(domain.CardViewFull)
		if err != nil {
			continue
		}
		bonusImgs = append(bonusImgs, imageutil.ScaleImage(img, winBonusScale))
	}

	return &DuelWinScreen{
		player:     player,
		choices:    winChoices,
		selected:   -1,
		bonusImgs:  bonusImgs,
		Background: bgImg,
		textbox:    tb,
		doneBtn:    doneBtn,
		ReturnScr:  screenui.WorldScr,
	}
}

// layoutChoices positions the choice cards in a centered horizontal row and
// records each card's hit rectangle for click detection.
func layoutChoices(cards []*domain.Card) []winChoice {
	n := len(cards)
	if n == 0 {
		return nil
	}

	totalW := n*winCardW + (n-1)*winChoiceGap
	startX := (winLogicalW - totalW) / 2

	choices := make([]winChoice, 0, n)
	for i, c := range cards {
		x := startX + i*(winCardW+winChoiceGap)
		choices = append(choices, winChoice{
			card: c,
			rect: image.Rect(x, winChoiceY, x+winCardW, winChoiceY+winCardH),
		})
	}
	return choices
}

func (s *DuelWinScreen) Draw(screen *ebiten.Image, W, H int, scale float64) {
	screen.DrawImage(s.Background, &ebiten.DrawImageOptions{})

	s.textbox.Draw(screen, &ebiten.DrawImageOptions{}, scale)

	mx, my := ebiten.CursorPosition()
	mp := image.Pt(mx, my)

	for i, c := range s.choices {
		img, err := c.card.CardImage(domain.CardViewFull)
		if err != nil {
			continue
		}
		if i == s.selected {
			drawSelectionBorder(screen, c.rect)
		}
		opts := &ebiten.DrawImageOptions{}
		if i == s.selected || mp.In(c.rect) {
			opts.ColorScale.Scale(1.2, 1.2, 1.2, 1.0)
		}
		opts.GeoM.Translate(float64(c.rect.Min.X), float64(c.rect.Min.Y))
		screen.DrawImage(img, opts)
	}

	if s.selected >= 0 {
		s.doneBtn.Draw(screen, &ebiten.DrawImageOptions{}, scale)
	}

	s.drawBonus(screen)
}

// drawSelectionBorder draws a highlight frame around the selected choice card.
func drawSelectionBorder(screen *ebiten.Image, rect image.Rectangle) {
	const pad = 6
	border := color.RGBA{R: 255, G: 215, B: 0, A: 255}
	x := float32(rect.Min.X - pad)
	y := float32(rect.Min.Y - pad)
	w := float32(rect.Dx() + 2*pad)
	h := float32(rect.Dy() + 2*pad)
	vector.FillRect(screen, x, y, w, h, border, false)
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
	if len(s.choices) == 0 {
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			return s.ReturnScr, nil, nil
		}
		return screenui.DuelWinScr, nil, nil
	}

	if s.selected >= 0 {
		s.doneBtn.Update(&ebiten.DrawImageOptions{}, scale, W, H)
		if s.doneBtn.IsClicked() {
			s.confirmSelection()
			return s.ReturnScr, nil, nil
		}
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		s.selectCardAt(image.Pt(mx, my))
	}

	return screenui.DuelWinScr, nil, nil
}

// selectCardAt marks the choice under the given point as the pending selection.
func (s *DuelWinScreen) selectCardAt(mp image.Point) {
	for i, c := range s.choices {
		if mp.In(c.rect) {
			s.selected = i
			return
		}
	}
}

// confirmSelection adds the pending card choice to the player's collection.
// It returns false when no card is selected.
func (s *DuelWinScreen) confirmSelection() bool {
	if s.selected < 0 || s.selected >= len(s.choices) {
		return false
	}
	s.player.CardCollection.AddCard(s.choices[s.selected].card, 1)
	return true
}
