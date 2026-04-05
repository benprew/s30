package screens

import (
	"fmt"

	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/save"
	"github.com/benprew/s30/game/ui/elements"
	"github.com/benprew/s30/game/ui/fonts"
	"github.com/benprew/s30/game/ui/imageutil"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

type startMode int

const (
	startModeMenu startMode = iota
	startModeLoad
)

type StartScreen struct {
	mode         startMode
	background   *ebiten.Image
	newGameBtn   *elements.Button
	loadGameBtn  *elements.Button
	backBtn      *elements.Button
	saveButtons  []*elements.Button
	saves        []save.SaveInfo
	hasSaves     bool
	SelectedSave string
	NewGame      bool
}

func (s *StartScreen) IsFramed() bool { return false }

func NewStartScreen() *StartScreen {
	btnSprites, err := imageutil.LoadSpriteSheet(3, 1, assets.Tradbut1_png)
	if err != nil {
		panic(fmt.Sprintf("Error loading button sprites: %v", err))
	}
	fontFace := &text.GoTextFace{Source: fonts.MtgFont, Size: 24}

	newGameW, newGameH := elements.TextButtonSize("New Game", fontFace)
	loadGameW, _ := elements.TextButtonSize("Load Game", fontFace)

	centerX := 512
	btnY := 400

	s := &StartScreen{}

	bgImg, _ := imageutil.LoadImage(assets.StartTitle_png)
	bgBounds := bgImg.Bounds()
	scaleX := 1024.0 / float64(bgBounds.Dx())
	scaleY := 768.0 / float64(bgBounds.Dy())
	s.background = imageutil.ScaleImageInd(bgImg, scaleX, scaleY)

	hasSaves := false
	if saveDir, err := save.SaveDir(); err == nil {
		if saves, err := save.ListSaves(saveDir); err == nil && len(saves) > 0 {
			hasSaves = true
		}
	}
	s.hasSaves = hasSaves

	s.newGameBtn = elements.NewButtonFromConfig(elements.ButtonConfig{
		Normal:  btnSprites[0][0],
		Hover:   btnSprites[0][1],
		Pressed: btnSprites[0][2],
		Text:    "New Game",
		Font:    fontFace,
		ID:      "new_game",
		X:       centerX - newGameW/2,
		Y:       btnY,
	})

	s.loadGameBtn = elements.NewButtonFromConfig(elements.ButtonConfig{
		Normal:  btnSprites[0][0],
		Hover:   btnSprites[0][1],
		Pressed: btnSprites[0][2],
		Text:    "Load Game",
		Font:    fontFace,
		ID:      "load_game",
		X:       centerX - loadGameW/2,
		Y:       btnY + newGameH + 20,
	})

	backW, _ := elements.TextButtonSize("Back", fontFace)
	s.backBtn = elements.NewButtonFromConfig(elements.ButtonConfig{
		Normal:  btnSprites[0][0],
		Hover:   btnSprites[0][1],
		Pressed: btnSprites[0][2],
		Text:    "Back",
		Font:    fontFace,
		ID:      "back",
		X:       centerX - backW/2,
		Y:       650,
	})

	return s
}

func (s *StartScreen) Update(W, H int, scale float64) (screenui.ScreenName, screenui.Screen, error) {
	opts := &ebiten.DrawImageOptions{}

	switch s.mode {
	case startModeMenu:
		s.newGameBtn.Update(opts, scale, W, H)
		if s.newGameBtn.IsClicked() {
			s.NewGame = true
			return screenui.WorldScr, nil, nil
		}

		if s.hasSaves {
			s.loadGameBtn.Update(opts, scale, W, H)
			if s.loadGameBtn.IsClicked() {
				s.loadSaveList()
			}
		}

	case startModeLoad:
		for _, btn := range s.saveButtons {
			btn.Update(opts, scale, W, H)
		}
		s.backBtn.Update(opts, scale, W, H)

		for i, btn := range s.saveButtons {
			if btn.IsClicked() {
				s.SelectedSave = s.saves[i].Path
				return screenui.WorldScr, nil, nil
			}
		}

		if s.backBtn.IsClicked() {
			s.mode = startModeMenu
			s.backBtn.State = elements.StateNormal
		}
	}

	return screenui.StartScr, nil, nil
}

func (s *StartScreen) Draw(screen *ebiten.Image, W, H int, scale float64) {
	screen.DrawImage(s.background, &ebiten.DrawImageOptions{})

	opts := &ebiten.DrawImageOptions{}

	switch s.mode {
	case startModeMenu:
		s.newGameBtn.Draw(screen, opts, scale)
		if s.hasSaves {
			s.loadGameBtn.Draw(screen, opts, scale)
		}

	case startModeLoad:
		headerFont := &text.GoTextFace{Source: fonts.MtgFont, Size: 30}
		headerText := "Load Game"
		headerW, _ := text.Measure(headerText, headerFont, 0)
		headerOpts := &text.DrawOptions{}
		headerOpts.GeoM.Translate(float64(W)/2-headerW/2, 280)
		headerOpts.ColorScale.Scale(1, 1, 1, 1)
		text.Draw(screen, headerText, headerFont, headerOpts)

		if len(s.saveButtons) == 0 {
			noSavesFont := &text.GoTextFace{Source: fonts.MtgFont, Size: 20}
			noSavesText := "No saved games found"
			noSavesW, _ := text.Measure(noSavesText, noSavesFont, 0)
			noSavesOpts := &text.DrawOptions{}
			noSavesOpts.GeoM.Translate(float64(W)/2-noSavesW/2, 380)
			noSavesOpts.ColorScale.Scale(0.7, 0.7, 0.7, 1)
			text.Draw(screen, noSavesText, noSavesFont, noSavesOpts)
		}

		for _, btn := range s.saveButtons {
			btn.Draw(screen, opts, scale)
		}

		s.backBtn.Draw(screen, opts, scale)
	}
}

func (s *StartScreen) loadSaveList() {
	s.mode = startModeLoad

	saveDir, err := save.SaveDir()
	if err != nil {
		fmt.Printf("Error getting save directory: %v\n", err)
		return
	}

	saves, err := save.ListSaves(saveDir)
	if err != nil {
		fmt.Printf("Error listing saves: %v\n", err)
		return
	}

	s.saves = saves
	s.saveButtons = nil

	btnSprites, err := imageutil.LoadSpriteSheet(3, 1, assets.Tradbut1_png)
	if err != nil {
		fmt.Printf("Error loading button sprites: %v\n", err)
		return
	}
	fontFace := &text.GoTextFace{Source: fonts.MtgFont, Size: 18}

	maxVisible := 8
	if len(saves) < maxVisible {
		maxVisible = len(saves)
	}

	centerX := 512
	startY := 330

	for i := 0; i < maxVisible; i++ {
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
}
