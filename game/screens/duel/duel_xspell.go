package duel

import (
	"fmt"
	"image/color"
	"slices"
	"uuid"

	"github.com/benprew/mage-go/pkg/mage/interactive"
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
	mx := action.MaxXValue

	s.xChoosingActions = actions
	s.xMaxValue = mx

	btnSprites, err := imageutil.LoadSpriteSheet(3, 1, assets.Tradbut1_png)
	if err != nil {
		logging.Printf(logging.Duel, "Error loading X choice button sprites: %v\n", err)
		return
	}

	fontFace := &text.GoTextFace{Source: fonts.MtgFont, Size: 16}
	s.xButtons = make([]*elements.Button, mx+1)
	for i := range mx + 1 {
		label := fmt.Sprintf("X = %d", i)
		btn := elements.NewButton(btnSprites[0][0], btnSprites[0][1], btnSprites[0][2], 0, 0, 1.0)
		btn.ButtonText = elements.ButtonText{
			Text:      label,
			Font:      fontFace,
			TextColor: color.White,
			HAlign:    elements.AlignCenter,
			VAlign:    elements.AlignMiddle,
		}
		s.xButtons[i] = btn
	}

	s.loadCardPreviewByName(action.CardName)
}

func (s *DuelScreen) exitXChoosingMode() {
	s.xChoosingActions = nil
	s.xButtons = nil
	s.xMaxValue = 0
}

func (s *DuelScreen) isChoosingX() bool {
	return s.xChoosingActions != nil
}

func (s *DuelScreen) updateXChoosingUI() {
	for i := range s.xButtons {
		key := ebiten.Key0 + ebiten.Key(i)
		if i <= 9 && inpututil.IsKeyJustPressed(key) {
			s.selectXValue(i)
			return
		}
	}

	btnW := 0
	if len(s.xButtons) > 0 {
		btnW = s.xButtons[0].Normal.Bounds().Dx()
	}

	img := s.previewImage()
	cardH := 0
	if img != nil {
		cardH = img.Bounds().Dy()
	}

	centerX := 512
	titleH := 30
	cardTopY := 768/2 - (cardH+titleH+len(s.xButtons)*40)/2
	btnStartY := cardTopY + titleH + cardH + 10

	for i, btn := range s.xButtons {
		btnX := centerX - btnW/2
		btnY := btnStartY + i*40
		btn.MoveTo(btnX, btnY)
		opts := &ebiten.DrawImageOptions{}
		btn.Update(opts, 1.0, 1024, 768)
		if btn.IsClicked() {
			s.selectXValue(i)
			return
		}
	}
}

func (s *DuelScreen) selectXValue(xValue int) {
	actions := slices.Clone(s.xChoosingActions)
	s.exitXChoosingMode()
	for i := range actions {
		actions[i].ValidTargets = targetsForX(actions[i], xValue)
	}

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
	if !s.isChoosingX() || s.xButtons == nil {
		return
	}

	vector.FillRect(screen, 0, 0, float32(W), float32(H), color.RGBA{0, 0, 0, 160}, false)

	centerX := float64(W) / 2

	img := s.previewImage()
	cardH := 0
	if img != nil {
		cardH = img.Bounds().Dy()
	}
	titleH := 30
	totalH := titleH + cardH + 10 + len(s.xButtons)*40
	startY := float64(H)/2 - float64(totalH)/2

	title := elements.NewText(24, "Choose X", 0, int(startY))
	title.HAlign = elements.AlignCenter
	title.BoundsW = float64(W)
	title.Color = color.White
	title.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)

	if img != nil {
		cardW := float64(img.Bounds().Dx())
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(centerX-cardW/2, startY+float64(titleH))
		screen.DrawImage(img, op)
	}

	btnOpts := &ebiten.DrawImageOptions{}
	for _, btn := range s.xButtons {
		btn.Draw(screen, btnOpts, 1.0)
	}
}

// xValueForAction returns the chosen X value if in X-choosing flow, 0 otherwise.
// Used by targeting mode when confirming a targeted X spell.
func (s *DuelScreen) xValueForAction() int {
	v := s.xChosenValue
	s.xChosenValue = 0
	return v
}
