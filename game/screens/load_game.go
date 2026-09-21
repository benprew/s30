package screens

import (
	"fmt"
	"image"
	"image/color"

	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/save"
	"github.com/benprew/s30/game/ui"
	"github.com/benprew/s30/game/ui/elements"
	"github.com/benprew/s30/game/ui/fonts"
	"github.com/benprew/s30/game/ui/imageutil"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// LoadGameScreen is the transparent overlay displaying saved games to load,
// opened from the in-game menu or the start screen.
type LoadGameScreen struct {
	backdrop     *ebiten.Image
	saves        []save.SaveInfo
	saveButtons  []*elements.Button
	backBtn      *elements.Button
	SelectedSave string
}

func NewLoadGameScreen() *LoadGameScreen {
	backdrop := ebiten.NewImage(1024, 768)
	backdrop.Fill(color.RGBA{18, 14, 10, 220})

	s := &LoadGameScreen{
		backdrop: backdrop,
	}
	s.loadSaveList()
	return s
}

func (s *LoadGameScreen) loadSaveList() {
	saveDir, err := save.SaveDir()
	if err != nil {
		return
	}

	saves, err := save.ListSaves(saveDir)
	if err != nil {
		return
	}

	s.saves = saves
	s.saveButtons = nil

	btnSprites, err := imageutil.LoadSpriteSheet(3, 1, assets.Tradbut1_png)
	if err != nil {
		return
	}
	fontFace := &text.GoTextFace{Source: fonts.MtgFont, Size: 18}

	maxVisible := min(len(saves), 8)
	centerX := 512
	startY := 330

	for i := range maxVisible {
		sv := saves[i]
		label := fmt.Sprintf("%s  -  %s", sv.Name, sv.SavedAt.Format("Jan 02 2006 15:04"))
		btnW, btnH := elements.TextButtonSize(label, fontFace)

		btn := elements.NewButtonFromConfig(elements.ButtonConfig{
			Normal:  btnSprites[0][0],
			Hover:   btnSprites[0][1],
			Pressed: btnSprites[0][2],
			Text:    label,
			Font:    fontFace,
			ID:      fmt.Sprintf("save_%d", i),
			X:       centerX - btnW/2,
			Y:       startY + i*(btnH+10),
		})
		s.saveButtons = append(s.saveButtons, btn)
	}

	backFontFace := &text.GoTextFace{Source: fonts.MtgFont, Size: 24}
	backW, _ := elements.TextButtonSize("Back", backFontFace)
	s.backBtn = elements.NewButtonFromConfig(elements.ButtonConfig{
		Normal:  btnSprites[0][0],
		Hover:   btnSprites[0][1],
		Pressed: btnSprites[0][2],
		Text:    "Back",
		Font:    backFontFace,
		ID:      "back",
		X:       centerX - backW/2,
		Y:       650,
	})
}

func (s *LoadGameScreen) IsFramed() bool  { return false }
func (s *LoadGameScreen) IsOverlay() bool { return true }

func (s *LoadGameScreen) handleSelection(pos image.Point, clicked bool) screenui.ScreenName {
	if !clicked {
		return screenui.LoadGameScr
	}
	if s.backBtn != nil && pos.In(s.backBtn.Bounds) {
		return screenui.PopScr
	}
	for i, btn := range s.saveButtons {
		if pos.In(btn.Bounds) {
			s.SelectedSave = s.saves[i].Path
			return screenui.WorldScr
		}
	}
	return screenui.LoadGameScr
}

func (s *LoadGameScreen) Update(W, H int, scale float64) (screenui.ScreenName, screenui.Screen, error) {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return screenui.PopScr, nil, nil
	}

	opts := &ebiten.DrawImageOptions{}
	for _, btn := range s.saveButtons {
		btn.Update(opts, scale, W, H)
	}
	if s.backBtn != nil {
		s.backBtn.Update(opts, scale, W, H)
	}

	target := s.handleSelection(ui.Position(), ui.Click(image.Rect(0, 0, W, H)))
	return target, nil, nil
}

func (s *LoadGameScreen) Draw(screen *ebiten.Image, W, H int, scale float64) {
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Scale(scale, scale)
	screen.DrawImage(s.backdrop, opts)

	headerFont := &text.GoTextFace{Source: fonts.MtgFont, Size: 30}
	headerText := "Load Game"
	headerW, _ := text.Measure(headerText, headerFont, 0)
	headerOpts := &text.DrawOptions{}
	headerOpts.GeoM.Translate(float64(W)/2-headerW/2, 280)
	headerOpts.GeoM.Scale(scale, scale)
	headerOpts.ColorScale.Scale(1, 1, 1, 1)
	text.Draw(screen, headerText, headerFont, headerOpts)

	if len(s.saveButtons) == 0 {
		noSavesFont := &text.GoTextFace{Source: fonts.MtgFont, Size: 20}
		noSavesText := "No saved games found"
		noSavesW, _ := text.Measure(noSavesText, noSavesFont, 0)
		noSavesOpts := &text.DrawOptions{}
		noSavesOpts.GeoM.Translate(float64(W)/2-noSavesW/2, 380)
		noSavesOpts.GeoM.Scale(scale, scale)
		noSavesOpts.ColorScale.Scale(0.7, 0.7, 0.7, 1)
		text.Draw(screen, noSavesText, noSavesFont, noSavesOpts)
	}

	for _, btn := range s.saveButtons {
		btn.Draw(screen, opts, scale)
	}

	if s.backBtn != nil {
		s.backBtn.Draw(screen, opts, scale)
	}
}
