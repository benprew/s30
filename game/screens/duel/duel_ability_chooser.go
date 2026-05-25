package duel

import (
	"fmt"
	"image/color"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"
	gameaudio "github.com/benprew/s30/game/audio"
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

func (s *DuelScreen) enterAbilityChoosingMode(actions []interactive.ActionOption) {
	s.abilityChoosingActions = actions

	btnSprites, err := imageutil.LoadSpriteSheet(3, 1, assets.Tradbut1_png)
	if err != nil {
		logging.Printf(logging.Duel, "Error loading ability choice button sprites: %v\n", err)
		return
	}

	fontFace := &text.GoTextFace{Source: fonts.MtgFont, Size: 16}
	s.abilityButtons = make([]*elements.Button, len(actions))
	for i, action := range actions {
		label := fmt.Sprintf("%d. %s", i+1, action.Label)
		btn := elements.NewButton(btnSprites[0][0], btnSprites[0][1], btnSprites[0][2], 0, 0, 1.0)
		btn.ButtonText = elements.ButtonText{
			Text:      label,
			Font:      fontFace,
			TextColor: color.White,
			HAlign:    elements.AlignCenter,
			VAlign:    elements.AlignMiddle,
		}
		s.abilityButtons[i] = btn
	}

	s.loadCardPreviewByName(actions[0].CardName)
}

func (s *DuelScreen) exitAbilityChoosingMode() {
	s.abilityChoosingActions = nil
	s.abilityButtons = nil
}

func (s *DuelScreen) isChoosingAbility() bool {
	return s.abilityChoosingActions != nil
}

func (s *DuelScreen) updateAbilityChoosingUI() {
	for i := range s.abilityButtons {
		key := ebiten.Key1 + ebiten.Key(i)
		if i < 9 && inpututil.IsKeyJustPressed(key) {
			s.selectAbility(i)
			return
		}
	}

	btnW := 0
	if len(s.abilityButtons) > 0 {
		btnW = s.abilityButtons[0].Normal.Bounds().Dx()
	}

	cardH := 0
	if s.cardPreviewImg != nil {
		cardH = s.cardPreviewImg.Bounds().Dy()
	}

	centerX := 512
	titleH := 30
	cardTopY := 768/2 - (cardH+titleH+len(s.abilityButtons)*40)/2
	btnStartY := cardTopY + titleH + cardH + 10

	for i, btn := range s.abilityButtons {
		btnX := centerX - btnW/2
		btnY := btnStartY + i*40
		btn.MoveTo(btnX, btnY)
		opts := &ebiten.DrawImageOptions{}
		btn.Update(opts, 1.0, 1024, 768)
		if btn.IsClicked() {
			s.selectAbility(i)
			return
		}
	}
}

func (s *DuelScreen) selectAbility(index int) {
	action := s.abilityChoosingActions[index]
	s.exitAbilityChoosingMode()

	if action.NeedsTarget {
		s.enterTargetingMode(action.PermanentID, action.CardName, []interactive.ActionOption{action})
		return
	}

	logging.Printf(logging.Duel, "CLICK: %s -> ability=%d\n", action.CardName, action.AbilityIndex)
	pa := actionOptionToPriorityAction(action)
	select {
	case s.human.FromTUI() <- pa:
		if am := gameaudio.Get(); am != nil {
			am.PlaySFX(gameaudio.SFXCast)
		}
	default:
	}
}

func (s *DuelScreen) drawAbilityChoosingUI(screen *ebiten.Image, W, H int) {
	if !s.isChoosingAbility() || s.abilityButtons == nil {
		return
	}

	vector.FillRect(screen, 0, 0, float32(W), float32(H), color.RGBA{0, 0, 0, 160}, false)

	centerX := float64(W) / 2

	cardH := 0
	if s.cardPreviewImg != nil {
		cardH = s.cardPreviewImg.Bounds().Dy()
	}
	titleH := 30
	totalH := titleH + cardH + 10 + len(s.abilityButtons)*40
	startY := float64(H)/2 - float64(totalH)/2

	title := elements.NewText(24, "Choose Ability", 0, int(startY))
	title.HAlign = elements.AlignCenter
	title.BoundsW = float64(W)
	title.Color = color.White
	title.Draw(screen, &ebiten.DrawImageOptions{}, 1.0)

	if s.cardPreviewImg != nil {
		cardW := float64(s.cardPreviewImg.Bounds().Dx())
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(centerX-cardW/2, startY+float64(titleH))
		screen.DrawImage(s.cardPreviewImg, op)
	}

	btnOpts := &ebiten.DrawImageOptions{}
	for _, btn := range s.abilityButtons {
		btn.Draw(screen, btnOpts, 1.0)
	}
}
