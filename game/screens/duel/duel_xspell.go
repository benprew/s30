package duel

import (
	"fmt"
	"image"
	"image/color"
	"uuid"

	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/ui"
	"github.com/benprew/s30/game/ui/elements"
	"github.com/benprew/s30/game/ui/fonts"
	"github.com/benprew/s30/game/ui/imageutil"
	"github.com/benprew/s30/logging"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type xChooserBounds struct {
	StartY  int
	CardY   int
	Minus   image.Rectangle
	Value   image.Rectangle
	Plus    image.Rectangle
	Confirm image.Rectangle
	BottomY int
}

func (s *DuelScreen) enterXChoosingMode(actions []interactive.ActionOption) {
	action := actions[0]
	mx := action.MaxXValue

	s.xChoosingActions = actions
	s.xMaxValue = mx
	s.xCurrentValue = mx
	s.xTyped = false

	btnSprites, err := imageutil.LoadSpriteSheet(3, 1, assets.Tradbut1_png)
	if err != nil {
		logging.Printf(logging.Duel, "Error loading X choice button sprites: %v\n", err)
		return
	}

	fontFace := &text.GoTextFace{Source: fonts.MtgFont, Size: 16}
	btn := elements.NewButton(btnSprites[0][0], btnSprites[0][1], btnSprites[0][2], 0, 0, 1.0)
	btn.ButtonText = elements.ButtonText{
		Text:      "Confirm",
		Font:      fontFace,
		TextColor: color.White,
		HAlign:    elements.AlignCenter,
		VAlign:    elements.AlignMiddle,
	}
	s.xConfirmBtn = btn

	s.loadCardPreviewByName(action.CardName)
}

func (s *DuelScreen) exitXChoosingMode() {
	s.xChoosingActions = nil
	s.xConfirmBtn = nil
	s.xMaxValue = 0
	s.xCurrentValue = 0
	s.xTyped = false
}

func (s *DuelScreen) isChoosingX() bool {
	return s.xChoosingActions != nil
}

func (s *DuelScreen) incrementX() {
	s.xTyped = false
	if s.xCurrentValue < s.xMaxValue {
		s.xCurrentValue++
	}
}

func (s *DuelScreen) decrementX() {
	s.xTyped = false
	if s.xCurrentValue > 0 {
		s.xCurrentValue--
	}
}

func (s *DuelScreen) typeXDigit(digit int) {
	if digit < 0 || digit > 9 {
		return
	}
	if s.xTyped {
		combined := s.xCurrentValue*10 + digit
		if combined <= s.xMaxValue {
			s.xCurrentValue = combined
			return
		}
	}
	if digit <= s.xMaxValue {
		s.xCurrentValue = digit
		s.xTyped = true
	}
}

func (s *DuelScreen) xChooserLayout(W, H, cardH int) xChooserBounds {
	const (
		titleH     = 30
		selectorH  = 40
		minusPlusW = 44
		valueW     = 120
		gap        = 8
	)

	confirmW := 148
	confirmH := 36
	if s.xConfirmBtn != nil && s.xConfirmBtn.Normal != nil {
		confirmW = s.xConfirmBtn.Normal.Bounds().Dx()
		confirmH = s.xConfirmBtn.Normal.Bounds().Dy()
	}

	totalH := titleH + cardH + 14 + selectorH + 12 + confirmH
	startY := max((H-totalH)/2, 12)
	cardY := startY + titleH
	selectorY := cardY + cardH + 14

	centerX := W / 2
	valueMinX := centerX - valueW/2
	valueMaxX := centerX + valueW/2

	minusBounds := image.Rect(valueMinX-gap-minusPlusW, selectorY, valueMinX-gap, selectorY+selectorH)
	valueBounds := image.Rect(valueMinX, selectorY, valueMaxX, selectorY+selectorH)
	plusBounds := image.Rect(valueMaxX+gap, selectorY, valueMaxX+gap+minusPlusW, selectorY+selectorH)

	confirmY := selectorY + selectorH + 12
	confirmBounds := image.Rect(centerX-confirmW/2, confirmY, centerX+confirmW/2, confirmY+confirmH)

	return xChooserBounds{
		StartY:  startY,
		CardY:   cardY,
		Minus:   minusBounds,
		Value:   valueBounds,
		Plus:    plusBounds,
		Confirm: confirmBounds,
		BottomY: confirmY + confirmH,
	}
}

func (s *DuelScreen) updateXChoosingUI(W, H int) {
	for i := range 10 {
		if inpututil.IsKeyJustPressed(ebiten.Key0+ebiten.Key(i)) || inpututil.IsKeyJustPressed(ebiten.KeyKP0+ebiten.Key(i)) {
			s.typeXDigit(i)
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) || inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) ||
		inpututil.IsKeyJustPressed(ebiten.KeyEqual) || inpututil.IsKeyJustPressed(ebiten.KeyKPAdd) {
		s.incrementX()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) || inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) ||
		inpututil.IsKeyJustPressed(ebiten.KeyMinus) || inpututil.IsKeyJustPressed(ebiten.KeyKPSubtract) {
		s.decrementX()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyKPEnter) {
		s.selectXValue(s.xCurrentValue)
		return
	}

	_, wheelY := ebiten.Wheel()
	if wheelY > 0 {
		s.incrementX()
	} else if wheelY < 0 {
		s.decrementX()
	}

	cardH := 0
	if img := s.previewImage(); img != nil {
		cardH = img.Bounds().Dy()
	}
	layout := s.xChooserLayout(W, H, cardH)

	if s.xConfirmBtn != nil {
		s.xConfirmBtn.MoveTo(layout.Confirm.Min.X, layout.Confirm.Min.Y)
		s.xConfirmBtn.Update(&ebiten.DrawImageOptions{}, 1.0, W, H)
		if s.xConfirmBtn.IsClicked() {
			s.selectXValue(s.xCurrentValue)
			return
		}
	}

	pointer := ui.Position()
	clicked := ui.Click(image.Rect(0, 0, W, H))
	s.updateXChoosingMouse(pointer.X, pointer.Y, clicked, W, H)
}

func (s *DuelScreen) updateXChoosingMouse(mx, my int, clicked bool, W, H int) {
	if !clicked {
		return
	}
	cardH := 0
	if img := s.previewImage(); img != nil {
		cardH = img.Bounds().Dy()
	}
	layout := s.xChooserLayout(W, H, cardH)
	pt := image.Pt(mx, my)

	switch {
	case pt.In(layout.Minus):
		s.decrementX()
	case pt.In(layout.Plus):
		s.incrementX()
	case pt.In(layout.Confirm):
		s.selectXValue(s.xCurrentValue)
	}
}

func (s *DuelScreen) selectXValue(xValue int) {
	actions := s.xChoosingActions
	s.exitXChoosingMode()

	if len(actions) == 1 && actions[0].NeedsTarget {
		if tid, ok := s.autoTargetStackSpell(actions[0]); ok {
			pa := actionOptionToPriorityAction(actions[0])
			pa.Targets = []uuid.UUID{tid}
			pa.XValue = xValue
			select {
			case s.human.FromTUI() <- pa:
			default:
			}
			return
		}
	}

	if len(actions) > 1 || actions[0].NeedsTarget {
		s.xChosenValue = xValue
		s.enterTargetingMode(actions[0].CardID, actions[0].CardName, actions)
		return
	}

	action := actions[0]
	pa := actionOptionToPriorityAction(action)
	pa.XValue = xValue
	logging.Printf(logging.Duel, "CLICK: %s -> X=%d\n", action.CardName, xValue)
	select {
	case s.human.FromTUI() <- pa:
	default:
	}
}

func (s *DuelScreen) drawXChoosingUI(screen *ebiten.Image, W, H int) {
	if !s.isChoosingX() {
		return
	}

	vector.FillRect(screen, 0, 0, float32(W), float32(H), color.RGBA{0, 0, 0, 160}, false)

	img := s.previewImage()
	cardH := 0
	if img != nil {
		cardH = img.Bounds().Dy()
	}
	layout := s.xChooserLayout(W, H, cardH)

	title := elements.NewText(24, "Choose X", 0, layout.StartY)
	title.HAlign = elements.AlignCenter
	title.BoundsW = float64(W)
	title.Color = color.White
	title.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)

	if img != nil {
		cardW := float64(img.Bounds().Dx())
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(W)/2-cardW/2, float64(layout.CardY))
		screen.DrawImage(img, op)
	}

	pointer := ui.Position()
	pt := image.Pt(pointer.X, pointer.Y)

	drawXStepperButton(screen, layout.Minus, "-", s.xCurrentValue > 0, pt.In(layout.Minus))
	drawXValueDisplay(screen, layout.Value, fmt.Sprintf("X = %d", s.xCurrentValue))
	drawXStepperButton(screen, layout.Plus, "+", s.xCurrentValue < s.xMaxValue, pt.In(layout.Plus))

	if s.xConfirmBtn != nil {
		s.xConfirmBtn.MoveTo(layout.Confirm.Min.X, layout.Confirm.Min.Y)
		s.xConfirmBtn.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)
	}
}

func drawXStepperButton(screen *ebiten.Image, bounds image.Rectangle, symbol string, enabled, hover bool) {
	bg := color.RGBA{R: 50, G: 45, B: 60, A: 240}
	if hover && enabled {
		bg = color.RGBA{R: 80, G: 70, B: 95, A: 240}
	} else if !enabled {
		bg = color.RGBA{R: 35, G: 30, B: 40, A: 180}
	}
	vector.FillRect(screen, float32(bounds.Min.X), float32(bounds.Min.Y), float32(bounds.Dx()), float32(bounds.Dy()), bg, false)

	border := color.RGBA{R: 160, G: 150, B: 175, A: 220}
	if !enabled {
		border = color.RGBA{R: 75, G: 70, B: 85, A: 150}
	}
	vector.StrokeRect(screen, float32(bounds.Min.X), float32(bounds.Min.Y), float32(bounds.Dx()), float32(bounds.Dy()), 1.5, border, false)

	txt := elements.NewText(22, symbol, bounds.Min.X, bounds.Min.Y)
	txt.HAlign = elements.AlignCenter
	txt.VAlign = elements.AlignMiddle
	txt.BoundsW = float64(bounds.Dx())
	txt.BoundsH = float64(bounds.Dy())
	if enabled {
		txt.Color = color.White
	} else {
		txt.Color = color.RGBA{R: 110, G: 110, B: 120, A: 180}
	}
	txt.Draw(screen, nil, 1.0)
}

func drawXValueDisplay(screen *ebiten.Image, bounds image.Rectangle, label string) {
	vector.FillRect(screen, float32(bounds.Min.X), float32(bounds.Min.Y), float32(bounds.Dx()), float32(bounds.Dy()),
		color.RGBA{R: 28, G: 26, B: 34, A: 240}, false)
	vector.StrokeRect(screen, float32(bounds.Min.X), float32(bounds.Min.Y), float32(bounds.Dx()), float32(bounds.Dy()),
		1.5, color.RGBA{R: 210, G: 185, B: 105, A: 230}, false)

	txt := elements.NewText(20, label, bounds.Min.X, bounds.Min.Y)
	txt.HAlign = elements.AlignCenter
	txt.VAlign = elements.AlignMiddle
	txt.BoundsW = float64(bounds.Dx())
	txt.BoundsH = float64(bounds.Dy())
	txt.Color = color.White
	txt.Draw(screen, nil, 1.0)
}

// xValueForAction returns the chosen X value if in X-choosing flow, 0 otherwise.
// Used by targeting mode when confirming a targeted X spell.
func (s *DuelScreen) xValueForAction() int {
	v := s.xChosenValue
	s.xChosenValue = 0
	return v
}
