package duel

import (
	"fmt"
	"image"
	"image/color"
	"slices"
	"uuid"

	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/benprew/s30/game/ui"
	"github.com/benprew/s30/game/ui/elements"
	"github.com/benprew/s30/game/ui/fonts"
	"github.com/benprew/s30/game/ui/imageutil"
	"github.com/benprew/s30/logging"

	"github.com/benprew/s30/assets"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

func targetsDetermineX(action interactive.ActionOption) bool {
	if !action.NeedsTarget || action.TargetType == nil || len(action.ValidTargets) == 0 {
		return false
	}
	if action.TargetType.Min() != 1 || action.TargetType.Max() != 1 {
		return false
	}
	for _, id := range action.ValidTargets {
		if _, ok := uniqueXForTarget(action, id); !ok {
			return false
		}
	}
	return true
}

func uniqueXForTarget(action interactive.ActionOption, id uuid.UUID) (int, bool) {
	xValue, found := 0, false
	for _, option := range action.XTargetOptions {
		if len(option.Candidates) != 1 {
			return 0, false
		}
		if !slices.Contains(option.Candidates[0], id) {
			continue
		}
		if found && xValue != option.XValue {
			return 0, false
		}
		xValue, found = option.XValue, true
	}
	return xValue, found
}

func targetsForX(action interactive.ActionOption, xValue int) []uuid.UUID {
	if action.XTargetOptions == nil {
		return action.ValidTargets
	}
	for _, option := range action.XTargetOptions {
		if option.XValue == xValue && len(option.Candidates) > 0 {
			return option.Candidates[0]
		}
	}
	return nil
}

func (s *DuelScreen) enterXChoosingMode(actions []interactive.ActionOption) {
	action := actions[0]
	s.xChoosingActions = actions
	s.xMaxValue = action.MaxXValue
	s.xCurrentValue = s.xMaxValue
	s.xTyping = false

	btnSprites, err := imageutil.LoadSpriteSheet(3, 1, assets.Tradbut1_png)
	if err != nil {
		logging.Printf(logging.Duel, "Error loading X choice button sprites: %v\n", err)
		return
	}

	label := "Confirm"
	if len(s.xChoosingTargets) > 0 {
		label = "Cast"
	}
	btn := elements.NewButton(btnSprites[0][0], btnSprites[0][1], btnSprites[0][2], 0, 0, 1.0)
	btn.ButtonText = elements.ButtonText{
		Text:      label,
		Font:      &text.GoTextFace{Source: fonts.MtgFont, Size: 16},
		TextColor: color.White,
		HAlign:    elements.AlignCenter,
		VAlign:    elements.AlignMiddle,
	}
	s.xConfirmButton = btn

	s.loadActionPreview(action)
}

func (s *DuelScreen) exitXChoosingMode() {
	s.xChoosingActions = nil
	s.xChoosingTargets = nil
	s.xConfirmButton = nil
	s.xMaxValue = 0
	s.xCurrentValue = 0
	s.xTyping = false
}

func (s *DuelScreen) isChoosingX() bool {
	return s.xChoosingActions != nil
}

func (s *DuelScreen) adjustX(delta int) {
	s.xCurrentValue = min(s.xMaxValue, max(0, s.xCurrentValue+delta))
	s.xTyping = false
}

func (s *DuelScreen) typeXDigit(digit int) {
	if s.xTyping && s.xCurrentValue*10+digit <= s.xMaxValue {
		s.xCurrentValue = s.xCurrentValue*10 + digit
		return
	}
	if digit <= s.xMaxValue {
		s.xCurrentValue = digit
		s.xTyping = true
	}
}

type xChooserBounds struct {
	titleY, cardY int
	minus, value  image.Rectangle
	plus, confirm image.Rectangle
}

func (s *DuelScreen) xChooserLayout(W, H, cardH int) xChooserBounds {
	const titleH, stepH, stepW, valueW, gap = 30, 40, 44, 120, 8
	button := s.xConfirmButton.Normal.Bounds()
	totalH := titleH + cardH + gap + stepH + gap + button.Dy()
	titleY := max(12, (H-totalH)/2)
	cardY := titleY + titleH
	stepY := cardY + cardH + gap
	center := W / 2
	value := image.Rect(center-valueW/2, stepY, center+valueW/2, stepY+stepH)
	minus := image.Rect(value.Min.X-gap-stepW, stepY, value.Min.X-gap, stepY+stepH)
	plus := image.Rect(value.Max.X+gap, stepY, value.Max.X+gap+stepW, stepY+stepH)
	confirm := button.Sub(button.Min).Add(image.Pt(center-button.Dx()/2, stepY+stepH+gap))
	return xChooserBounds{titleY: titleY, cardY: cardY, minus: minus, value: value, plus: plus, confirm: confirm}
}

func (s *DuelScreen) handleXChooserShortcuts(keyPressed func(ebiten.Key) bool, wheelY float64) {
	for i := range 10 {
		if keyPressed(ebiten.Key0+ebiten.Key(i)) || keyPressed(ebiten.KeyKP0+ebiten.Key(i)) {
			s.typeXDigit(i)
		}
	}
	switch {
	case keyPressed(ebiten.KeyArrowLeft), keyPressed(ebiten.KeyArrowDown),
		keyPressed(ebiten.KeyMinus), keyPressed(ebiten.KeyKPSubtract):
		s.adjustX(-1)
	case keyPressed(ebiten.KeyArrowRight), keyPressed(ebiten.KeyArrowUp),
		keyPressed(ebiten.KeyEqual), keyPressed(ebiten.KeyKPAdd):
		s.adjustX(1)
	case keyPressed(ebiten.KeyEnter), keyPressed(ebiten.KeyKPEnter):
		s.selectXValue(s.xCurrentValue)
		return
	}
	switch {
	case wheelY > 0:
		s.adjustX(1)
	case wheelY < 0:
		s.adjustX(-1)
	}
}

func (s *DuelScreen) updateXChoosingUI(W, H int) {
	_, wheelY := ebiten.Wheel()
	s.handleXChooserShortcuts(inpututil.IsKeyJustPressed, wheelY)
	if s.xConfirmButton == nil {
		return
	}

	img := s.previewImage()
	cardH := 0
	if img != nil {
		cardH = img.Bounds().Dy()
	}

	layout := s.xChooserLayout(W, H, cardH)
	s.xConfirmButton.MoveTo(layout.confirm.Min.X, layout.confirm.Min.Y)
	s.xConfirmButton.Update(&ebiten.DrawImageOptions{}, 1.0, W, H)
	s.handleXChooserClick(layout, ui.Click)
}

func (s *DuelScreen) handleXChooserClick(layout xChooserBounds, clicked func(image.Rectangle) bool) {
	switch {
	case clicked(layout.confirm):
		s.selectXValue(s.xCurrentValue)
	case clicked(layout.minus):
		s.adjustX(-1)
	case clicked(layout.plus):
		s.adjustX(1)
	}
}

func (s *DuelScreen) selectXValue(xValue int) {
	actions := slices.Clone(s.xChoosingActions)
	targets := s.xChoosingTargets
	s.exitXChoosingMode()
	if len(targets) > 0 {
		s.submitXAction(actions[0], xValue, targets)
		return
	}
	for i := range actions {
		actions[i].ValidTargets = targetsForX(actions[i], xValue)
	}

	if len(actions) == 1 && actions[0].NeedsTarget {
		if tid, ok := s.autoTargetStackSpell(actions[0]); ok {
			s.submitXAction(actions[0], xValue, []uuid.UUID{tid})
			return
		}
	}

	if len(actions) > 1 || actions[0].NeedsTarget {
		s.xChosenValue = xValue
		s.enterTargetingMode(actions[0].CardID, actions[0].CardName, actions)
		return
	}

	s.submitXAction(actions[0], xValue, nil)
}

func (s *DuelScreen) submitXAction(action interactive.ActionOption, xValue int, targets []uuid.UUID) {
	pa := actionOptionToPriorityAction(action)
	pa.XValue = xValue
	pa.Targets = targets
	logging.Printf(logging.Duel, "CLICK: %s -> X=%d\n", action.CardName, xValue)
	select {
	case s.human.FromTUI() <- pa:
	default:
	}
}

func (s *DuelScreen) drawXChoosingUI(screen *ebiten.Image, W, H int) {
	if !s.isChoosingX() || s.xConfirmButton == nil {
		return
	}

	vector.FillRect(screen, 0, 0, float32(W), float32(H), color.RGBA{0, 0, 0, 160}, false)

	img := s.previewImage()
	cardH := 0
	if img != nil {
		cardH = img.Bounds().Dy()
	}
	layout := s.xChooserLayout(W, H, cardH)
	titleText := "Choose X"
	count := len(s.xChoosingTargets)
	if count > 0 && s.xChoosingActions[0].CardName == "Fireball" {
		titleText = fmt.Sprintf("Choose X up to %d (%d damage each)", s.xMaxValue, s.xCurrentValue/count)
	}
	title := elements.NewText(24, titleText, 0, layout.titleY)
	title.HAlign = elements.AlignCenter
	title.BoundsW = float64(W)
	title.Color = color.White
	title.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)

	if img != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(W-img.Bounds().Dx())/2, float64(layout.cardY))
		screen.DrawImage(img, op)
	}

	drawXStepButton(screen, layout.minus, "-", s.xCurrentValue > 0)
	drawXStepButton(screen, layout.plus, "+", s.xCurrentValue < s.xMaxValue)
	drawXSelectorText(screen, layout.value, fmt.Sprintf("X = %d", s.xCurrentValue), color.White)
	s.xConfirmButton.MoveTo(layout.confirm.Min.X, layout.confirm.Min.Y)
	s.xConfirmButton.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)
}

func drawXStepButton(screen *ebiten.Image, bounds image.Rectangle, label string, enabled bool) {
	bg := color.RGBA{R: 50, G: 45, B: 60, A: 255}
	fg := color.RGBA{R: 225, G: 220, B: 235, A: 255}
	switch {
	case !enabled:
		fg = color.RGBA{R: 153, G: 147, B: 165, A: 255}
	case ui.Position().In(bounds):
		bg = color.RGBA{R: 80, G: 70, B: 95, A: 255}
	}
	x, y, w, h := float32(bounds.Min.X), float32(bounds.Min.Y), float32(bounds.Dx()), float32(bounds.Dy())
	vector.FillRect(screen, x, y, w, h, bg, false)
	vector.StrokeRect(screen, x, y, w, h, 1, fg, false)
	drawXSelectorText(screen, bounds, label, fg)
}

func drawXSelectorText(screen *ebiten.Image, bounds image.Rectangle, label string, ink color.Color) {
	txt := elements.NewText(22, label, bounds.Min.X, bounds.Min.Y)
	txt.HAlign = elements.AlignCenter
	txt.VAlign = elements.AlignMiddle
	txt.BoundsW = float64(bounds.Dx())
	txt.BoundsH = float64(bounds.Dy())
	txt.Color = ink
	txt.Draw(screen, &ebiten.DrawImageOptions{}, 1)
}

// xValueForAction returns the chosen X value if in X-choosing flow, 0 otherwise.
// Used by targeting mode when confirming a targeted X spell.
func (s *DuelScreen) xValueForAction() int {
	v := s.xChosenValue
	s.xChosenValue = 0
	return v
}
